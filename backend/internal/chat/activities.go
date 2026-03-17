package chat

import (
	"context"

	"github.com/foundrylab-app/cinova/backend/internal/graph"
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
}

// WriteRecsInput is the input for the WriteRecommendations activity.
type WriteRecsInput struct {
	History            []models.ChatMessage
	Message            string
	UserOwnerType      string
	UserOwnerID        string
	Candidates         []models.MovieSummary
	Country            string
	RequestedProviders []string
	ProviderDropped    bool
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
	result, err := s.extractIntent(ctx, input.History, input.Message)
	if err != nil {
		return nil, err
	}
	return &ExtractIntentOutput{
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
	candidates, providerDropped := s.fetchCandidates(ctx, input.Filters, input.Country)
	return &FetchCandidatesOutput{
		Candidates:      candidates,
		ProviderDropped: providerDropped,
	}, nil
}

// WriteRecsActivity wraps writeRecommendations for Temporal activity use.
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

	recs := make([]RecEntry, len(out.Recommendations))
	for i, r := range out.Recommendations {
		recs[i] = RecEntry{TMDBID: r.TMDBID, Title: r.Title, Reason: r.Reason}
	}
	return &WriteRecsOutput{Reply: out.Reply, Recommendations: recs}, nil
}
