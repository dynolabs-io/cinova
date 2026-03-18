package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/rs/zerolog/log"

	"github.com/foundrylab-app/cinova/backend/internal/chat"
	"github.com/foundrylab-app/cinova/backend/internal/graph"
)

// CandidatesHandler exposes Neo4j candidate lookup to internal callers (e.g. Langflow).
// This endpoint is NOT exposed via the public ingress — it is internal-only.
type CandidatesHandler struct {
	svc *chat.Service
}

// NewCandidatesHandler creates a CandidatesHandler.
func NewCandidatesHandler(svc *chat.Service) *CandidatesHandler {
	return &CandidatesHandler{svc: svc}
}

// candidatesRequest is the POST body for /internal/v1/candidates.
type candidatesRequest struct {
	Intent  candidatesIntent `json:"intent"`
	Country string           `json:"country"`
}

// candidatesIntent mirrors the intent fields that drive Neo4j filtering.
type candidatesIntent struct {
	Genres     []string `json:"genres"`
	Themes     []string `json:"themes"`
	Moods      []string `json:"moods"`
	Providers  []string `json:"providers"`
	Directors  []string `json:"directors"`
	Actors     []string `json:"actors"`
	Language   string   `json:"language"`
	MaxRuntime int      `json:"max_runtime"`
	MinYear    int      `json:"min_year"`
	MaxYear    int      `json:"max_year"`
	MinScore   float64  `json:"min_score"`
}

// candidatesResponse is returned by /internal/v1/candidates.
type candidatesResponse struct {
	Candidates      interface{} `json:"candidates"`
	ProviderDropped bool        `json:"provider_dropped"`
}

// GetCandidates handles POST /internal/v1/candidates.
// It applies the same filter + tiered fallback logic as the chat service.
func (h *CandidatesHandler) GetCandidates(w http.ResponseWriter, r *http.Request) {
	var req candidatesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "invalid_request", "invalid JSON body", http.StatusBadRequest)
		return
	}

	country := req.Country
	if country == "" {
		country = "US"
	}

	minScore := req.Intent.MinScore
	if minScore <= 0 {
		minScore = 40.0
	}
	// No score floor when filtering by specific person or language.
	if len(req.Intent.Directors) > 0 || len(req.Intent.Actors) > 0 || req.Intent.Language != "" {
		minScore = 0
	}

	filters := graph.ChatFilters{
		Genres:     req.Intent.Genres,
		Themes:     req.Intent.Themes,
		Moods:      req.Intent.Moods,
		Providers:  req.Intent.Providers,
		Directors:  req.Intent.Directors,
		Actors:     req.Intent.Actors,
		Language:   req.Intent.Language,
		MaxRuntime: req.Intent.MaxRuntime,
		MinYear:    req.Intent.MinYear,
		MaxYear:    req.Intent.MaxYear,
		MinScore:   minScore,
	}

	candidates, providerDropped := h.svc.FetchCandidates(r.Context(), filters, country)

	log.Debug().
		Int("candidates", len(candidates)).
		Bool("provider_dropped", providerDropped).
		Str("country", country).
		Msg("internal/candidates: fetched")

	writeJSON(w, http.StatusOK, candidatesResponse{
		Candidates:      candidates,
		ProviderDropped: providerDropped,
	})
}
