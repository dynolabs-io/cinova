package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/rs/zerolog/log"

	"github.com/dynolabs-io/cinova/backend/internal/auth"
	"github.com/dynolabs-io/cinova/backend/internal/graph"
	"github.com/dynolabs-io/cinova/backend/internal/models"
	"github.com/dynolabs-io/cinova/backend/internal/store"
)

// InteractionHandler handles user interaction endpoints (rate, save, dismiss, watchlist).
type InteractionHandler struct {
	repo  *graph.MovieRepository
	pg    *store.PostgresStore
	redis *store.RedisStore
}

// NewInteractionHandler creates a new InteractionHandler.
func NewInteractionHandler(repo *graph.MovieRepository, pg *store.PostgresStore, redis *store.RedisStore) *InteractionHandler {
	return &InteractionHandler{repo: repo, pg: pg, redis: redis}
}

// ownerInfo derives the Neo4j owner node label and ID from the request context.
// Anonymous sessions use Session nodes; registered users use User nodes.
func ownerInfo(r *http.Request) (ownerID, ownerType string) {
	if auth.IsAnonymousFromCtx(r.Context()) {
		return auth.SessionIDFromCtx(r.Context()), "Session"
	}
	return auth.UserIDFromCtx(r.Context()), "User"
}

// Rate handles POST /api/v1/me/rate
// Body: {"tmdb_id": 123, "media_type": "movie", "rating": "like"|"dislike"}
func (h *InteractionHandler) Rate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TMDBID    int64  `json:"tmdb_id"`
		MediaType string `json:"media_type"`
		Rating    string `json:"rating"` // "like" | "dislike"
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "bad_request", "cannot decode request body", http.StatusBadRequest)
		return
	}
	if req.TMDBID == 0 {
		writeError(w, "bad_request", "tmdb_id is required", http.StatusBadRequest)
		return
	}
	if req.Rating != "like" && req.Rating != "dislike" {
		writeError(w, "bad_request", "rating must be 'like' or 'dislike'", http.StatusBadRequest)
		return
	}

	score := 1.0 // like
	if req.Rating == "dislike" {
		score = -1.0
	}

	ownerID, ownerType := ownerInfo(r)

	if err := h.repo.RateTitle(r.Context(), ownerID, ownerType, int(req.TMDBID), score); err != nil {
		log.Error().Err(err).Str("owner", ownerID).Int64("tmdb_id", req.TMDBID).Msg("Rate")
		writeError(w, "internal_error", "failed to rate title", http.StatusInternalServerError)
		return
	}

	// Invalidate recommendation cache so next fetch reflects new rating.
	if err := h.redis.InvalidateRecommendations(r.Context(), ownerID); err != nil {
		log.Warn().Err(err).Str("owner", ownerID).Msg("invalidate rec cache after rate")
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok":      true,
		"tmdb_id": req.TMDBID,
		"rating":  req.Rating,
	})
}

// Save handles POST /api/v1/me/save
// Body: {"tmdb_id": 123, "media_type": "movie"}
func (h *InteractionHandler) Save(w http.ResponseWriter, r *http.Request) {
	var req models.SaveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "bad_request", "cannot decode request body", http.StatusBadRequest)
		return
	}
	if req.TMDBID == 0 {
		writeError(w, "bad_request", "tmdb_id is required", http.StatusBadRequest)
		return
	}

	ownerID, ownerType := ownerInfo(r)

	if err := h.repo.SaveTitle(r.Context(), ownerID, ownerType, int(req.TMDBID)); err != nil {
		log.Error().Err(err).Str("owner", ownerID).Int64("tmdb_id", req.TMDBID).Msg("Save")
		writeError(w, "internal_error", "failed to save title", http.StatusInternalServerError)
		return
	}

	// Invalidate recommendation cache.
	if err := h.redis.InvalidateRecommendations(r.Context(), ownerID); err != nil {
		log.Warn().Err(err).Str("owner", ownerID).Msg("invalidate rec cache after save")
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok":      true,
		"tmdb_id": req.TMDBID,
	})
}

// Unsave handles DELETE /api/v1/me/save
// Body: {"tmdb_id": 123, "media_type": "movie"}
func (h *InteractionHandler) Unsave(w http.ResponseWriter, r *http.Request) {
	var req models.SaveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "bad_request", "cannot decode request body", http.StatusBadRequest)
		return
	}
	if req.TMDBID == 0 {
		writeError(w, "bad_request", "tmdb_id is required", http.StatusBadRequest)
		return
	}

	ownerID, ownerType := ownerInfo(r)

	if err := h.repo.UnsaveTitle(r.Context(), ownerID, ownerType, int(req.TMDBID)); err != nil {
		log.Error().Err(err).Str("owner", ownerID).Int64("tmdb_id", req.TMDBID).Msg("Unsave")
		writeError(w, "internal_error", "failed to unsave title", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok":      true,
		"tmdb_id": req.TMDBID,
	})
}

// Dismiss handles POST /api/v1/me/dismiss
// Body: {"tmdb_id": 123, "media_type": "movie"}
func (h *InteractionHandler) Dismiss(w http.ResponseWriter, r *http.Request) {
	var req models.DismissRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "bad_request", "cannot decode request body", http.StatusBadRequest)
		return
	}
	if req.TMDBID == 0 {
		writeError(w, "bad_request", "tmdb_id is required", http.StatusBadRequest)
		return
	}

	ownerID, ownerType := ownerInfo(r)

	if err := h.repo.DismissTitle(r.Context(), ownerID, ownerType, int(req.TMDBID)); err != nil {
		log.Error().Err(err).Str("owner", ownerID).Int64("tmdb_id", req.TMDBID).Msg("Dismiss")
		writeError(w, "internal_error", "failed to dismiss title", http.StatusInternalServerError)
		return
	}

	// Invalidate recommendation cache so dismissed titles are excluded immediately.
	if err := h.redis.InvalidateRecommendations(r.Context(), ownerID); err != nil {
		log.Warn().Err(err).Str("owner", ownerID).Msg("invalidate rec cache after dismiss")
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok":      true,
		"tmdb_id": req.TMDBID,
	})
}

// GetWatchlist handles GET /api/v1/me/watchlist?page=1&limit=20
func (h *InteractionHandler) GetWatchlist(w http.ResponseWriter, r *http.Request) {
	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 {
			limit = v
		}
	}
	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}

	ownerID, ownerType := ownerInfo(r)

	movies, err := h.repo.GetWatchlist(r.Context(), ownerID, ownerType)
	if err != nil {
		log.Error().Err(err).Str("owner", ownerID).Msg("GetWatchlist")
		writeError(w, "internal_error", "failed to fetch watchlist", http.StatusInternalServerError)
		return
	}

	// Apply pagination in-memory (GetWatchlist returns all, ordered by saved_at DESC)
	start := (page - 1) * limit
	if start >= len(movies) {
		movies = nil
	} else {
		end := start + limit
		if end > len(movies) {
			end = len(movies)
		}
		movies = movies[start:end]
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"results": movies,
		"page":    page,
		"limit":   limit,
		"total":   len(movies),
	})
}
