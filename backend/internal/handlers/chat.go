package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/foundrylab-app/cinova/backend/internal/auth"
	"github.com/foundrylab-app/cinova/backend/internal/chat"
	"github.com/foundrylab-app/cinova/backend/internal/models"
	"github.com/foundrylab-app/cinova/backend/internal/store"
)

// ChatHandler handles POST /api/v1/me/chat.
type ChatHandler struct {
	svc *chat.Service
	pg  *store.PostgresStore
}

// NewChatHandler creates a new ChatHandler.
func NewChatHandler(svc *chat.Service, pg *store.PostgresStore) *ChatHandler {
	return &ChatHandler{svc: svc, pg: pg}
}

// Chat handles a single conversation turn.
func (h *ChatHandler) Chat(w http.ResponseWriter, r *http.Request) {
	var req models.ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "invalid_request", "invalid JSON body", http.StatusBadRequest)
		return
	}
	if req.Message == "" {
		writeError(w, "invalid_request", "message is required", http.StatusBadRequest)
		return
	}

	country := r.URL.Query().Get("country")
	if country == "" {
		country = "US"
	}

	// Identify caller
	var userID string
	sessionID := auth.SessionIDFromCtx(r.Context())
	if !auth.IsAnonymousFromCtx(r.Context()) {
		userID = auth.UserIDFromCtx(r.Context())
	}

	// Use session ID as conv ID if none provided
	convID := req.ConvID
	if convID == "" {
		convID = sessionID
	}

	// Load conversation history
	history, err := h.pg.GetChatHistory(r.Context(), convID, 10)
	if err != nil {
		log.Warn().Err(err).Str("conv_id", convID).Msg("chat: failed to load history")
		history = []models.ChatMessage{}
	}

	resp, err := h.svc.Chat(r.Context(), userID, sessionID, convID, country, history, req.Message)
	if err != nil {
		log.Error().Err(err).Msg("chat: service error")
		writeError(w, "internal_error", "chat service unavailable", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// ChatStream handles POST /api/v1/me/chat/stream — SSE streaming variant.
// Sends delta text chunks immediately as Claude generates them, then fires
// a "suggestions" event once recommendations are ready.
func (h *ChatHandler) ChatStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, "streaming_unsupported", "streaming not supported", http.StatusInternalServerError)
		return
	}

	var req models.ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "invalid_request", "invalid JSON body", http.StatusBadRequest)
		return
	}
	if req.Message == "" {
		writeError(w, "invalid_request", "message is required", http.StatusBadRequest)
		return
	}

	country := r.URL.Query().Get("country")
	if country == "" {
		country = "US"
	}

	var userID string
	sessionID := auth.SessionIDFromCtx(r.Context())
	if !auth.IsAnonymousFromCtx(r.Context()) {
		userID = auth.UserIDFromCtx(r.Context())
	}

	convID := req.ConvID
	if convID == "" {
		convID = sessionID
	}

	history, err := h.pg.GetChatHistory(r.Context(), convID, 10)
	if err != nil {
		log.Warn().Err(err).Str("conv_id", convID).Msg("chat stream: failed to load history")
		history = []models.ChatMessage{}
	}

	// SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // disable nginx proxy buffering

	// Disable write deadline for long-lived streaming connection
	rc := http.NewResponseController(w)
	_ = rc.SetWriteDeadline(time.Time{})

	if err := h.svc.StreamChat(r.Context(), userID, sessionID, convID, country, history, req.Message, w, flusher); err != nil {
		log.Error().Err(err).Msg("chat stream: service error")
	}
}
