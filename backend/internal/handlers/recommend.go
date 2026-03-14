package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/foundrylab-app/cinova/backend/internal/auth"
	"github.com/foundrylab-app/cinova/backend/internal/graph"
	"github.com/foundrylab-app/cinova/backend/internal/store"
)

const recCacheTTL = 30 * time.Minute

// RecommendHandler handles personalized recommendation requests.
type RecommendHandler struct {
	repo  *graph.MovieRepository
	redis *store.RedisStore
}

// NewRecommendHandler creates a new RecommendHandler.
func NewRecommendHandler(repo *graph.MovieRepository, redis *store.RedisStore) *RecommendHandler {
	return &RecommendHandler{repo: repo, redis: redis}
}

// GetRecommendations handles GET /api/v1/recommend?country=US&limit=20
// Returns personalized recommendations based on the user's interaction history.
// Falls back to trending if no interaction history exists.
func (h *RecommendHandler) GetRecommendations(w http.ResponseWriter, r *http.Request) {
	country := r.URL.Query().Get("country")
	if country == "" {
		country = "US"
	}
	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 {
			limit = v
		}
	}

	// Determine owner for personalization
	var subjectID string
	var ownerType string
	if auth.IsAnonymousFromCtx(r.Context()) {
		subjectID = auth.SessionIDFromCtx(r.Context())
		ownerType = "Session"
	} else {
		subjectID = auth.UserIDFromCtx(r.Context())
		ownerType = "User"
	}

	cacheKey := fmt.Sprintf("cinova:rec:%s:%s", subjectID, country)

	// Try cache first
	if cached, err := h.redis.GetCachedRecommendations(r.Context(), subjectID, country); err == nil && cached != nil {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cache", "HIT")
		w.WriteHeader(http.StatusOK)
		w.Write(cached)
		return
	}

	movies, err := h.repo.GetRecommendations(r.Context(), subjectID, ownerType, country, limit)
	if err != nil {
		log.Error().Err(err).Str("subject", subjectID).Msg("GetRecommendations")
		writeError(w, "internal_error", "failed to fetch recommendations", http.StatusInternalServerError)
		return
	}

	// Fall back to trending if no personalized results
	source := "personalized"
	if len(movies) == 0 {
		source = "trending"
		movies, err = h.repo.GetTrending(r.Context(), country, limit)
		if err != nil {
			log.Error().Err(err).Str("country", country).Msg("GetRecommendations fallback to trending")
			writeError(w, "internal_error", "failed to fetch recommendations", http.StatusInternalServerError)
			return
		}
	}

	response := map[string]interface{}{
		"results": movies,
		"source":  source,
		"country": country,
		"total":   len(movies),
	}

	// Cache the result
	if data, err := json.Marshal(response); err == nil {
		if err := h.redis.Set(r.Context(), cacheKey, string(data), recCacheTTL); err != nil {
			log.Warn().Err(err).Str("subject", subjectID).Msg("failed to cache recommendations")
		}
	}

	writeJSON(w, http.StatusOK, response)
}
