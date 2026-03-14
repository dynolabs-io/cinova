package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/rs/zerolog/log"

	"github.com/foundrylab-app/cinova/backend/internal/models"
)

// writeJSON encodes v as JSON and writes it to w with the given status code.
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Error().Err(err).Msg("write json response")
	}
}

// writeError writes a standard JSON error response.
func writeError(w http.ResponseWriter, errCode, message string, status int) {
	writeJSON(w, status, models.ErrorResponse{
		Error:   errCode,
		Message: message,
		Code:    status,
	})
}
