package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"

	"github.com/foundrylab-app/cinova/backend/internal/graph"
	"github.com/foundrylab-app/cinova/backend/internal/store"
)

const (
	trendingCacheTTL = 15 * time.Minute
	popularCacheTTL  = 15 * time.Minute
	reelsCacheTTL    = 15 * time.Minute
)

// MovieHandler handles movie and TV-related API requests.
type MovieHandler struct {
	repo  *graph.MovieRepository
	redis *store.RedisStore
}

// NewMovieHandler creates a new MovieHandler.
func NewMovieHandler(repo *graph.MovieRepository, redis *store.RedisStore) *MovieHandler {
	return &MovieHandler{repo: repo, redis: redis}
}

// GetMovie handles GET /api/v1/movie/{id}
func (h *MovieHandler) GetMovie(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeError(w, "bad_request", "invalid movie id", http.StatusBadRequest)
		return
	}

	movie, err := h.repo.GetMovie(r.Context(), id)
	if err != nil {
		log.Error().Err(err).Int("id", id).Msg("GetMovie")
		if isNotFound(err) {
			writeError(w, "not_found", fmt.Sprintf("movie %d not found", id), http.StatusNotFound)
			return
		}
		writeError(w, "internal_error", "failed to fetch movie", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, movie)
}

// GetTV handles GET /api/v1/tv/{id}
func (h *MovieHandler) GetTV(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeError(w, "bad_request", "invalid tv id", http.StatusBadRequest)
		return
	}

	show, err := h.repo.GetTVShow(r.Context(), id)
	if err != nil {
		log.Error().Err(err).Int("id", id).Msg("GetTV")
		if isNotFound(err) {
			writeError(w, "not_found", fmt.Sprintf("tv show %d not found", id), http.StatusNotFound)
			return
		}
		writeError(w, "internal_error", "failed to fetch tv show", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, show)
}

// GetTrending handles GET /api/v1/trending?country=US&limit=20
func (h *MovieHandler) GetTrending(w http.ResponseWriter, r *http.Request) {
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

	cacheKey := fmt.Sprintf("cinova:trending:%s:%d", country, limit)

	// Try cache first
	if cached, err := h.redis.Get(r.Context(), cacheKey); err == nil && cached != "" {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cache", "HIT")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(cached))
		return
	}

	movies, err := h.repo.GetTrending(r.Context(), country, limit)
	if err != nil {
		log.Error().Err(err).Str("country", country).Msg("GetTrending")
		writeError(w, "internal_error", "failed to fetch trending", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"results": movies,
		"country": country,
		"total":   len(movies),
	}

	data, err := json.Marshal(response)
	if err == nil {
		if err := h.redis.Set(r.Context(), cacheKey, string(data), trendingCacheTTL); err != nil {
			log.Warn().Err(err).Msg("failed to cache trending results")
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Cache", "MISS")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// GetPopular handles GET /api/v1/popular?country=US&limit=20&page=1
func (h *MovieHandler) GetPopular(w http.ResponseWriter, r *http.Request) {
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
	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	offset := (page - 1) * limit

	cacheKey := fmt.Sprintf("cinova:popular:%s:%d:%d", country, limit, page)

	// Try cache first
	if cached, err := h.redis.Get(r.Context(), cacheKey); err == nil && cached != "" {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cache", "HIT")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(cached))
		return
	}

	movies, err := h.repo.GetPopular(r.Context(), country, limit, offset)
	if err != nil {
		log.Error().Err(err).Str("country", country).Msg("GetPopular")
		writeError(w, "internal_error", "failed to fetch popular", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"results": movies,
		"country": country,
		"page":    page,
		"limit":   limit,
		"total":   len(movies),
	}

	data, err := json.Marshal(response)
	if err == nil {
		if err := h.redis.Set(r.Context(), cacheKey, string(data), popularCacheTTL); err != nil {
			log.Warn().Err(err).Msg("failed to cache popular results")
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Cache", "MISS")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// GetMovieProviders handles GET /api/v1/movie/{id}/providers?country=US
func (h *MovieHandler) GetMovieProviders(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeError(w, "bad_request", "invalid movie id", http.StatusBadRequest)
		return
	}
	country := r.URL.Query().Get("country")
	if country == "" {
		country = "US"
	}

	providers, err := h.repo.GetStreamingProviders(r.Context(), id, country)
	if err != nil {
		log.Error().Err(err).Int("id", id).Str("country", country).Msg("GetMovieProviders")
		writeError(w, "internal_error", "failed to fetch providers", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"tmdb_id":   id,
		"country":   country,
		"providers": providers,
	})
}

// GetReels handles GET /api/v1/discover/reels?country=US
// Returns trending movies with backdrop_path, overview, genres, cinova_score, and providers.
func (h *MovieHandler) GetReels(w http.ResponseWriter, r *http.Request) {
	country := r.URL.Query().Get("country")
	if country == "" {
		country = "US"
	}
	limit := 30

	cacheKey := fmt.Sprintf("cinova:reels:%s", country)

	// Try cache first
	if cached, err := h.redis.Get(r.Context(), cacheKey); err == nil && cached != "" {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cache", "HIT")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(cached))
		return
	}

	movies, err := h.repo.GetReels(r.Context(), country, limit)
	if err != nil {
		log.Error().Err(err).Str("country", country).Msg("GetReels")
		writeError(w, "internal_error", "failed to fetch reels", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"reels":   movies,
		"country": country,
		"total":   len(movies),
	}

	data, err := json.Marshal(response)
	if err == nil {
		if err := h.redis.Set(r.Context(), cacheKey, string(data), reelsCacheTTL); err != nil {
			log.Warn().Err(err).Msg("failed to cache reels")
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Cache", "MISS")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// isNotFound returns true if the error message contains "not found".
func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	for i := 0; i+9 <= len(s); i++ {
		if s[i:i+9] == "not found" {
			return true
		}
	}
	return false
}
