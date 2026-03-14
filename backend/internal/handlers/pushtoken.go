package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/foundrylab-app/cinova/backend/internal/auth"
	"github.com/foundrylab-app/cinova/backend/internal/store"
	"github.com/rs/zerolog/log"
)

// PushTokenHandler handles device push token registration.
type PushTokenHandler struct {
	pg *store.PostgresStore
}

func NewPushTokenHandler(pg *store.PostgresStore) *PushTokenHandler {
	return &PushTokenHandler{pg: pg}
}

type registerPushTokenReq struct {
	Token    string `json:"token"`
	Platform string `json:"platform"` // "ios" | "android"
}

// RegisterPushToken stores the device's Expo push token for the authenticated user/session.
// POST /api/v1/me/push-token
func (h *PushTokenHandler) RegisterPushToken(w http.ResponseWriter, r *http.Request) {
	var req registerPushTokenReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Token == "" {
		http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
		return
	}

	// Accept user ID or fall back to anonymous session ID
	userID := auth.UserIDFromCtx(r.Context())
	if userID == "" {
		userID = auth.SessionIDFromCtx(r.Context())
	}
	if userID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	platform := req.Platform
	if platform == "" {
		platform = "unknown"
	}

	// Upsert push token — one token per user (last-write wins)
	_, err := h.pg.Pool().Exec(r.Context(),
		`INSERT INTO push_tokens (user_id, token, platform, updated_at)
		 VALUES ($1, $2, $3, NOW())
		 ON CONFLICT (user_id) DO UPDATE
		   SET token      = EXCLUDED.token,
		       platform   = EXCLUDED.platform,
		       updated_at = NOW()`,
		userID, req.Token, platform,
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to upsert push token")
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"registered"}`))
}
