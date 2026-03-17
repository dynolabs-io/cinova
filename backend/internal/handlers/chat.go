package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
	"go.temporal.io/sdk/client"
	temporalclient "go.temporal.io/sdk/client"

	"github.com/foundrylab-app/cinova/backend/internal/auth"
	"github.com/foundrylab-app/cinova/backend/internal/chat"
	"github.com/foundrylab-app/cinova/backend/internal/models"
	"github.com/foundrylab-app/cinova/backend/internal/store"
	"github.com/foundrylab-app/cinova/backend/internal/workflow"
)

// ChatHandler handles POST /api/v1/me/chat and POST /api/v1/me/chat/stream.
type ChatHandler struct {
	svc            *chat.Service
	pg             *store.PostgresStore
	temporalClient temporalclient.Client // optional — nil means direct execution
}

// NewChatHandler creates a new ChatHandler.
// Pass a non-nil temporalClient to route non-streaming Chat through Temporal.
func NewChatHandler(svc *chat.Service, pg *store.PostgresStore, temporalClient client.Client) *ChatHandler {
	return &ChatHandler{svc: svc, pg: pg, temporalClient: temporalClient}
}

// Chat handles a single conversation turn.
// When a Temporal client is available the pipeline runs as a durable workflow
// (visible in the Temporal WebUI). Otherwise it falls back to direct execution.
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
		log.Warn().Err(err).Str("conv_id", convID).Msg("chat: failed to load history")
		history = []models.ChatMessage{}
	}

	if h.temporalClient != nil {
		h.chatViaWorkflow(w, r, userID, sessionID, convID, country, history, req.Message)
		return
	}

	resp, err := h.svc.Chat(r.Context(), userID, sessionID, convID, country, history, req.Message)
	if err != nil {
		log.Error().Err(err).Msg("chat: service error")
		writeError(w, "internal_error", "chat service unavailable", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// chatViaWorkflow runs the chat pipeline as a Temporal workflow and blocks
// until the workflow completes (up to 90s timeout).
func (h *ChatHandler) chatViaWorkflow(
	w http.ResponseWriter,
	r *http.Request,
	userID, sessionID, convID, country string,
	history []models.ChatMessage,
	message string,
) {
	wfInput := workflow.ChatInput{
		UserID:    userID,
		SessionID: sessionID,
		ConvID:    convID,
		Country:   country,
		History:   history,
		Message:   message,
	}

	wfOptions := temporalclient.StartWorkflowOptions{
		TaskQueue:                workflow.TaskQueue,
		WorkflowExecutionTimeout: 120 * time.Second,
	}

	ctx := r.Context()
	run, err := h.temporalClient.ExecuteWorkflow(ctx, wfOptions, workflow.ChatWorkflow, wfInput)
	if err != nil {
		log.Error().Err(err).Msg("chat workflow: failed to start")
		writeError(w, "internal_error", "workflow unavailable", http.StatusInternalServerError)
		return
	}

	var output workflow.ChatOutput
	if err := run.Get(ctx, &output); err != nil {
		log.Error().Err(err).Str("workflow_id", run.GetID()).Msg("chat workflow: execution failed")
		writeError(w, "internal_error", "workflow execution failed", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, &models.ChatResponse{
		Reply:       output.Reply,
		Suggestions: output.Suggestions,
		ConvID:      output.ConvID,
	})
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
