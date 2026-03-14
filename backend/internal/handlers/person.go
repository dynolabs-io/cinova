package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"

	"github.com/foundrylab-app/cinova/backend/internal/graph"
)

// PersonHandler handles person (actor/director) API requests.
type PersonHandler struct {
	repo *graph.MovieRepository
}

// NewPersonHandler creates a new PersonHandler.
func NewPersonHandler(repo *graph.MovieRepository) *PersonHandler {
	return &PersonHandler{repo: repo}
}

// GetPerson handles GET /api/v1/person/{id}
func (h *PersonHandler) GetPerson(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeError(w, "bad_request", "invalid person id", http.StatusBadRequest)
		return
	}

	person, err := h.repo.GetPerson(r.Context(), id)
	if err != nil {
		log.Error().Err(err).Int("id", id).Msg("GetPerson")
		if isNotFound(err) {
			writeError(w, "not_found", fmt.Sprintf("person %d not found", id), http.StatusNotFound)
			return
		}
		writeError(w, "internal_error", "failed to fetch person", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, person)
}
