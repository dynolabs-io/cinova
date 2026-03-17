package chat

import (
	"context"
	"time"

	"github.com/foundrylab-app/cinova/backend/internal/graph"
	"github.com/foundrylab-app/cinova/backend/internal/langfuse"
	"github.com/foundrylab-app/cinova/backend/internal/models"
)

// ── Temporal Activity types ───────────────────────────────────────────────────
// Exported so the workflow package can reference them.

// ExtractIntentInput is the input for the ExtractIntent activity.
type ExtractIntentInput struct {
	History []models.ChatMessage
	Message string
}

// ExtractIntentOutput mirrors intentResult with exported fields.
type ExtractIntentOutput struct {
	NeedsClarification bool
	ClarifyingQuestion string
	Genres             []string
	Themes             []string
	Moods              []string
	Providers          []string
	Directors          []string
	Actors             []string
	Language           string
	MaxRuntime         int
	MinYear            int
	MaxYear            int
	MinScore           float64
	// Trace timing — set by the activity, forwarded to WriteRecsInput.
	Start time.Time
	End   time.Time
}

// FetchCandidatesInput is the input for the FetchCandidates activity.
type FetchCandidatesInput struct {
	Filters graph.ChatFilters
	Country string
}

// FetchCandidatesOutput is the result of the FetchCandidates activity.
type FetchCandidatesOutput struct {
	Candidates      []models.MovieSummary
	ProviderDropped bool
	// Trace timing — forwarded to WriteRecsInput.
	Start time.Time
	End   time.Time
}

// WriteRecsInput is the input for the WriteRecommendations activity.
type WriteRecsInput struct {
	History            []models.ChatMessage
	Message            string
	UserID             string // empty for anonymous
	SessionID          string
	UserOwnerType      string
	UserOwnerID        string
	Candidates         []models.MovieSummary
	Country            string
	RequestedProviders []string
	ProviderDropped    bool

	// Trace fields — collected by the workflow and forwarded here so
	// WriteRecsActivity can send the full Langfuse trace in one shot.
	TraceID           string
	IntentStart       time.Time
	IntentEnd         time.Time
	IntentOutput      interface{}
	CandidateStart    time.Time
	CandidateEnd      time.Time
	CandidateFilters  interface{}
	CandidateCount    int
}

// WriteRecsOutput is the result of the WriteRecommendations activity.
type WriteRecsOutput struct {
	Reply           string
	Recommendations []RecEntry
}

// RecEntry is an exported recommendation (TMDB ID + title + AI reason).
type RecEntry struct {
	TMDBID int64
	Title  string
	Reason string
}

// ── Activity methods on *Service ─────────────────────────────────────────────

// ExtractIntentActivity wraps extractIntent for Temporal activity use.
func (s *Service) ExtractIntentActivity(ctx context.Context, input ExtractIntentInput) (*ExtractIntentOutput, error) {
	start := time.Now()
	result, err := s.extractIntent(ctx, input.History, input.Message)
	end := time.Now()
	if err != nil {
		return nil, err
	}
	return &ExtractIntentOutput{
		Start: start,
		End:   end,
		NeedsClarification: result.NeedsClarification,
		ClarifyingQuestion: result.ClarifyingQuestion,
		Genres:             result.Genres,
		Themes:             result.Themes,
		Moods:              result.Moods,
		Providers:          result.Providers,
		Directors:          result.Directors,
		Actors:             result.Actors,
		Language:           result.Language,
		MaxRuntime:         result.MaxRuntime,
		MinYear:            result.MinYear,
		MaxYear:            result.MaxYear,
		MinScore:           result.MinScore,
	}, nil
}

// FetchCandidatesActivity wraps fetchCandidates for Temporal activity use.
func (s *Service) FetchCandidatesActivity(ctx context.Context, input FetchCandidatesInput) (*FetchCandidatesOutput, error) {
	start := time.Now()
	candidates, providerDropped := s.fetchCandidates(ctx, input.Filters, input.Country)
	return &FetchCandidatesOutput{
		Candidates:      candidates,
		ProviderDropped: providerDropped,
		Start:           start,
		End:             time.Now(),
	}, nil
}

// WriteRecsActivity wraps writeRecommendations for Temporal activity use.
// It also persists the user + assistant messages to Postgres so that
// conversation history is available in subsequent workflow runs.
func (s *Service) WriteRecsActivity(ctx context.Context, input WriteRecsInput) (*WriteRecsOutput, error) {
	userCtx, _ := s.repo.GetUserChatContext(ctx, input.UserOwnerID, input.UserOwnerType)

	out, err := s.writeRecommendations(
		ctx,
		input.History,
		input.Message,
		userCtx,
		input.Candidates,
		input.Country,
		input.RequestedProviders,
		input.ProviderDropped,
	)
	if err != nil {
		return nil, err
	}

	// Persist conversation so history is available in subsequent turns.
	if _, err := s.pg.SaveChatMessage(ctx, input.UserID, input.SessionID, "user", input.Message); err != nil {
		_ = err
	}
	if _, err := s.pg.SaveChatMessage(ctx, input.UserID, input.SessionID, "assistant", out.Reply); err != nil {
		_ = err
	}

	// Send Langfuse trace (non-blocking, best-effort).
	if input.TraceID != "" {
		recommendEnd := time.Now()
		s.tracer.Send(langfuse.ChatTrace{
			TraceID:        input.TraceID,
			UserID:         input.UserID,
			SessionID:      input.SessionID,
			Country:        input.Country,
			UserMessage:    input.Message,
			IntentModel:    intentModel,
			RecommendModel: chatModel,
			IntentStart:    input.IntentStart,
			IntentEnd:      input.IntentEnd,
			IntentInput:    map[string]interface{}{"history_len": len(input.History), "message": input.Message},
			IntentOutput:   input.IntentOutput,
			CandidateStart:   input.CandidateStart,
			CandidateEnd:     input.CandidateEnd,
			CandidateFilters: input.CandidateFilters,
			CandidateCount:   input.CandidateCount,
			ProviderDropped:  input.ProviderDropped,
			RecommendStart:  recommendEnd, // approximate — activity start not tracked separately
			RecommendEnd:    recommendEnd,
			Reply:           out.Reply,
			SuggestionCount: len(out.Recommendations),
		})
	}

	recs := make([]RecEntry, len(out.Recommendations))
	for i, r := range out.Recommendations {
		recs[i] = RecEntry{TMDBID: r.TMDBID, Title: r.Title, Reason: r.Reason}
	}
	return &WriteRecsOutput{Reply: out.Reply, Recommendations: recs}, nil
}
