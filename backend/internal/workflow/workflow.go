// Package workflow implements the Cinova chat pipeline as a Temporal workflow.
//
// The pipeline has three activities executed sequentially:
//  1. ExtractIntent  — Haiku parses the user message into structured filters
//  2. FetchCandidates — Neo4j returns matching film candidates
//  3. WriteRecommendations — Sonnet selects + writes personalised reasons
package workflow

import (
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"github.com/foundrylab-app/cinova/backend/internal/chat"
	"github.com/foundrylab-app/cinova/backend/internal/graph"
	"github.com/foundrylab-app/cinova/backend/internal/models"
)

// TaskQueue is the Temporal task queue name for all cinova chat workflows.
const TaskQueue = "cinova-chat"

// ChatInput carries everything needed to run a full chat pipeline turn.
type ChatInput struct {
	UserID    string
	SessionID string
	ConvID    string
	Country   string
	History   []models.ChatMessage
	Message   string
}

// ChatOutput is the result of a complete chat pipeline run.
type ChatOutput struct {
	Reply       string
	Suggestions []models.MovieSummary
	ConvID      string
}

// ChatWorkflow orchestrates the three-activity chat pipeline.
// Activity timeouts are generous to handle peak LLM latency.
func ChatWorkflow(ctx workflow.Context, input ChatInput) (*ChatOutput, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("ChatWorkflow started", "session_id", input.SessionID)

	// ── Progress query handler (used by SSE streaming path) ───────────────
	progress := "extracting_intent"
	_ = workflow.SetQueryHandler(ctx, "progress", func() (string, error) {
		return progress, nil
	})

	// ── Activity 1: Extract Intent ────────────────────────────────────────
	intentAO := workflow.ActivityOptions{
		StartToCloseTimeout: 20 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts:    3,
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
		},
	}
	var intentOut chat.ExtractIntentOutput
	err := workflow.ExecuteActivity(
		workflow.WithActivityOptions(ctx, intentAO),
		(*chat.Service).ExtractIntentActivity,
		chat.ExtractIntentInput{History: input.History, Message: input.Message},
	).Get(ctx, &intentOut)
	if err != nil {
		logger.Warn("intent extraction failed, using defaults", "error", err)
		intentOut = chat.ExtractIntentOutput{MinScore: 40}
	}
	progress = "fetching_candidates"

	// ── Build ChatFilters (mirrors logic in chat.Service) ─────────────────
	minScore := intentOut.MinScore
	if minScore <= 0 {
		minScore = 40
	}
	if len(intentOut.Directors) > 0 || len(intentOut.Actors) > 0 || intentOut.Language != "" {
		minScore = 0
	}
	filters := graph.ChatFilters{
		Genres:     intentOut.Genres,
		Themes:     intentOut.Themes,
		Moods:      intentOut.Moods,
		Providers:  intentOut.Providers,
		Directors:  intentOut.Directors,
		Actors:     intentOut.Actors,
		Language:   intentOut.Language,
		MaxRuntime: intentOut.MaxRuntime,
		MinYear:    intentOut.MinYear,
		MaxYear:    intentOut.MaxYear,
		MinScore:   minScore,
	}

	// ── Activity 2: Fetch Candidates ──────────────────────────────────────
	fetchAO := workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 3,
			InitialInterval: time.Second,
		},
	}
	var fetchOut chat.FetchCandidatesOutput
	if err := workflow.ExecuteActivity(
		workflow.WithActivityOptions(ctx, fetchAO),
		(*chat.Service).FetchCandidatesActivity,
		chat.FetchCandidatesInput{Filters: filters, Country: input.Country},
	).Get(ctx, &fetchOut); err != nil {
		return nil, err
	}
	progress = "generating_recommendations"

	// ── Activity 3: Generate Recommendations ─────────────────────────────
	ownerType, ownerID := "Session", input.SessionID
	if input.UserID != "" {
		ownerType, ownerID = "User", input.UserID
	}
	recAO := workflow.ActivityOptions{
		StartToCloseTimeout: 90 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 2,
			InitialInterval: 2 * time.Second,
		},
	}
	var recOut chat.WriteRecsOutput
	recErr := workflow.ExecuteActivity(
		workflow.WithActivityOptions(ctx, recAO),
		(*chat.Service).WriteRecsActivity,
		chat.WriteRecsInput{
			History:            input.History,
			Message:            input.Message,
			UserID:             input.UserID,
			SessionID:          input.SessionID,
			UserOwnerType:      ownerType,
			UserOwnerID:        ownerID,
			Candidates:         fetchOut.Candidates,
			Country:            input.Country,
			RequestedProviders: filters.Providers,
			ProviderDropped:    fetchOut.ProviderDropped,
		},
	).Get(ctx, &recOut)
	if recErr != nil {
		return nil, recErr
	}

	// ── Map recommendations to enriched MovieSummary objects ──────────────
	suggMap := make(map[int64]models.MovieSummary, len(fetchOut.Candidates))
	for _, c := range fetchOut.Candidates {
		suggMap[c.TMDBID] = c
	}
	suggestions := make([]models.MovieSummary, 0, len(recOut.Recommendations))
	for _, rec := range recOut.Recommendations {
		if c, ok := suggMap[rec.TMDBID]; ok {
			c.Reason = rec.Reason
			suggestions = append(suggestions, c)
		}
	}

	logger.Info("ChatWorkflow complete", "suggestions", len(suggestions))
	return &ChatOutput{
		Reply:       recOut.Reply,
		Suggestions: suggestions,
		ConvID:      input.ConvID,
	}, nil
}
