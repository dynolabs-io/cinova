package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"go.temporal.io/sdk/client"
	temporalclient "go.temporal.io/sdk/client"

	"github.com/dynolabs-io/cinova/backend/internal/auth"
	"github.com/dynolabs-io/cinova/backend/internal/chat"
	"github.com/dynolabs-io/cinova/backend/internal/langflow"
	"github.com/dynolabs-io/cinova/backend/internal/models"
	"github.com/dynolabs-io/cinova/backend/internal/store"
	"github.com/dynolabs-io/cinova/backend/internal/workflow"
)

// ChatHandler handles POST /api/v1/me/chat and POST /api/v1/me/chat/stream.
type ChatHandler struct {
	svc            *chat.Service
	pg             *store.PostgresStore
	langflowClient *langflow.Client     // preferred — nil falls back to Temporal or direct
	temporalClient temporalclient.Client // optional — nil means direct execution
}

// NewChatHandler creates a new ChatHandler.
// Pass a non-nil langflowClient to route chat through Langflow.
// Pass a non-nil temporalClient to route chat through Temporal (legacy).
func NewChatHandler(svc *chat.Service, pg *store.PostgresStore, lf *langflow.Client, temporalClient client.Client) *ChatHandler {
	return &ChatHandler{svc: svc, pg: pg, langflowClient: lf, temporalClient: temporalClient}
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

	if h.langflowClient != nil {
		h.chatViaLangflow(w, r, userID, sessionID, convID, country, history, req.Message)
		return
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
		TraceID:   uuid.NewString(),
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

	if h.langflowClient != nil {
		h.streamViaLangflow(r.Context(), w, flusher, userID, sessionID, convID, country, history, req.Message)
		return
	}

	if h.temporalClient != nil {
		h.streamViaWorkflow(r.Context(), w, flusher, userID, sessionID, convID, country, history, req.Message)
		return
	}

	if err := h.svc.StreamChat(r.Context(), userID, sessionID, convID, country, history, req.Message, w, flusher); err != nil {
		log.Error().Err(err).Msg("chat stream: service error")
	}
}

// streamViaWorkflow runs a ChatWorkflow and emits SSE progress events at each
// activity boundary, then sends the full reply + suggestions when done.
// Token-by-token streaming is NOT available — the full reply arrives at once.
func (h *ChatHandler) streamViaWorkflow(
	ctx context.Context,
	w http.ResponseWriter,
	flusher http.Flusher,
	userID, sessionID, convID, country string,
	history []models.ChatMessage,
	message string,
) {
	sendSSE := func(v interface{}) {
		b, _ := json.Marshal(v)
		fmt.Fprintf(w, "data: %s\n\n", b)
		flusher.Flush()
	}

	wfOptions := temporalclient.StartWorkflowOptions{
		TaskQueue:                workflow.TaskQueue,
		WorkflowExecutionTimeout: 120 * time.Second,
	}
	wfInput := workflow.ChatInput{
		UserID: userID, SessionID: sessionID, ConvID: convID,
		Country: country, History: history, Message: message,
		TraceID: uuid.NewString(),
	}

	run, err := h.temporalClient.ExecuteWorkflow(ctx, wfOptions, workflow.ChatWorkflow, wfInput)
	if err != nil {
		log.Error().Err(err).Msg("chat stream workflow: failed to start")
		sendSSE(map[string]string{"type": "error", "text": "workflow unavailable"})
		sendSSE(map[string]string{"type": "done"})
		return
	}

	// Collect result in background goroutine.
	type result struct {
		out workflow.ChatOutput
		err error
	}
	resultCh := make(chan result, 1)
	go func() {
		var out workflow.ChatOutput
		err := run.Get(context.Background(), &out)
		resultCh <- result{out, err}
	}()

	stepLabels := map[string]string{
		"extracting_intent":        "Analyzing your request…",
		"fetching_candidates":      "Searching films…",
		"generating_recommendations": "Writing recommendations…",
	}
	var lastStep string
	ticker := time.NewTicker(400 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case r := <-resultCh:
			if r.err != nil {
				log.Error().Err(r.err).Str("workflow_id", run.GetID()).Msg("chat stream workflow: failed")
				sendSSE(map[string]string{"type": "error", "text": "workflow execution failed"})
				sendSSE(map[string]string{"type": "done"})
				return
			}
			sendSSE(map[string]string{"type": "delta", "text": r.out.Reply})
			sendSSE(map[string]interface{}{"type": "suggestions", "items": r.out.Suggestions, "conv_id": r.out.ConvID})
			sendSSE(map[string]string{"type": "done"})
			return

		case <-ctx.Done():
			return

		case <-ticker.C:
			val, err := h.temporalClient.QueryWorkflow(ctx, run.GetID(), run.GetRunID(), "progress")
			if err != nil {
				continue
			}
			var step string
			if err := val.Get(&step); err != nil || step == lastStep {
				continue
			}
			lastStep = step
			if label, ok := stepLabels[step]; ok {
				sendSSE(map[string]string{"type": "status", "text": label})
			}
		}
	}
}

// ── Langflow-backed methods ────────────────────────────────────────────────────

// chatViaLangflow runs the chat pipeline through Langflow and returns JSON.
func (h *ChatHandler) chatViaLangflow(
	w http.ResponseWriter,
	r *http.Request,
	userID, sessionID, convID, country string,
	history []models.ChatMessage,
	message string,
) {
	input := langflow.PipelineInput{
		Message:   message,
		Country:   country,
		SessionID: sessionID,
		ConvID:    convID,
		UserID:    userID,
		History:   history,
	}

	ctx := r.Context()
	out, err := h.langflowClient.Run(ctx, input)
	if err != nil {
		log.Error().Err(err).Msg("chat langflow: pipeline failed")
		writeError(w, "internal_error", "chat pipeline unavailable", http.StatusInternalServerError)
		return
	}

	// Persist conversation history.
	h.svc.PersistMessages(ctx, userID, sessionID, message, out.Reply)

	writeJSON(w, http.StatusOK, &models.ChatResponse{
		Reply:       out.Reply,
		Suggestions: out.Suggestions,
		ConvID:      convID,
	})
}

// streamViaLangflow runs the Langflow pipeline and emits SSE events.
// Sends status events during execution, then the full reply + suggestions.
func (h *ChatHandler) streamViaLangflow(
	ctx context.Context,
	w http.ResponseWriter,
	flusher http.Flusher,
	userID, sessionID, convID, country string,
	history []models.ChatMessage,
	message string,
) {
	sendSSE := func(v interface{}) {
		b, _ := json.Marshal(v)
		fmt.Fprintf(w, "data: %s\n\n", b)
		flusher.Flush()
	}

	sendSSE(map[string]string{"type": "status", "text": "Analyzing your request…"})

	input := langflow.PipelineInput{
		Message:   message,
		Country:   country,
		SessionID: sessionID,
		ConvID:    convID,
		UserID:    userID,
		History:   history,
	}

	// Run Langflow in a goroutine so we can send intermediate status events.
	type result struct {
		out *langflow.PipelineOutput
		err error
	}
	resultCh := make(chan result, 1)
	go func() {
		out, err := h.langflowClient.Run(ctx, input)
		resultCh <- result{out, err}
	}()

	// Send status updates while waiting.
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	statuses := []string{"Searching films…", "Writing recommendations…"}
	statusIdx := 0

	for {
		select {
		case r := <-resultCh:
			if r.err != nil {
				log.Error().Err(r.err).Msg("chat stream langflow: pipeline failed")
				sendSSE(map[string]string{"type": "error", "text": "pipeline execution failed"})
				sendSSE(map[string]string{"type": "done"})
				return
			}

			// Persist conversation history.
			h.svc.PersistMessages(ctx, userID, sessionID, message, r.out.Reply)

			sendSSE(map[string]string{"type": "delta", "text": r.out.Reply})
			sendSSE(map[string]interface{}{
				"type":    "suggestions",
				"items":   r.out.Suggestions,
				"conv_id": convID,
			})
			sendSSE(map[string]string{"type": "done"})
			return

		case <-ctx.Done():
			return

		case <-ticker.C:
			if statusIdx < len(statuses) {
				sendSSE(map[string]string{"type": "status", "text": statuses[statusIdx]})
				statusIdx++
			}
		}
	}
}
