// Package langfuse provides a lightweight client for the Langfuse tracing API.
// It captures chat pipeline traces — intent extraction, Neo4j candidate fetch,
// and LLM recommendation generation — and sends them asynchronously so the
// hot path is never blocked.
package langfuse

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// Client sends trace events to a Langfuse server.
type Client struct {
	baseURL    string
	publicKey  string
	secretKey  string
	httpClient *http.Client
}

// NewClient creates a Langfuse client. If baseURL is empty, the client is a
// no-op (tracing disabled) so the app runs fine without Langfuse configured.
func NewClient(baseURL, publicKey, secretKey string) *Client {
	return &Client{
		baseURL:   baseURL,
		publicKey: publicKey,
		secretKey: secretKey,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// Enabled returns true if tracing is configured.
func (c *Client) Enabled() bool {
	return c.baseURL != "" && c.publicKey != "" && c.secretKey != ""
}

// ── Wire types ────────────────────────────────────────────────────────────────

type event struct {
	ID        string      `json:"id"`
	Type      string      `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	Body      interface{} `json:"body"`
}

type traceBody struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	UserID    string                 `json:"userId,omitempty"`
	SessionID string                 `json:"sessionId,omitempty"`
	Input     interface{}            `json:"input,omitempty"`
	Output    interface{}            `json:"output,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

type generationBody struct {
	ID                 string                 `json:"id"`
	TraceID            string                 `json:"traceId"`
	ParentObservationID string               `json:"parentObservationId,omitempty"`
	Name               string                 `json:"name"`
	StartTime          time.Time              `json:"startTime"`
	EndTime            time.Time              `json:"endTime"`
	Model              string                 `json:"model,omitempty"`
	Input              interface{}            `json:"input,omitempty"`
	Output             interface{}            `json:"output,omitempty"`
	Usage              *usage                 `json:"usage,omitempty"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
	Level              string                 `json:"level,omitempty"`
}

type spanBody struct {
	ID        string                 `json:"id"`
	TraceID   string                 `json:"traceId"`
	Name      string                 `json:"name"`
	StartTime time.Time              `json:"startTime"`
	EndTime   time.Time              `json:"endTime"`
	Input     interface{}            `json:"input,omitempty"`
	Output    interface{}            `json:"output,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

type usage struct {
	Input  int    `json:"input,omitempty"`
	Output int    `json:"output,omitempty"`
	Unit   string `json:"unit,omitempty"`
}

// ── ChatTrace is the domain model for one chat turn ───────────────────────────

// ChatTrace captures all the signals from a single chat pipeline execution.
type ChatTrace struct {
	TraceID   string
	UserID    string
	SessionID string
	Country   string

	// Input
	UserMessage string

	// Pass 1 — intent extraction
	IntentStart  time.Time
	IntentEnd    time.Time
	IntentModel  string
	IntentInput  interface{} // messages sent to haiku
	IntentOutput interface{} // parsed intentResult JSON

	// Pass 1b — Neo4j candidate fetch
	CandidateStart    time.Time
	CandidateEnd      time.Time
	CandidateFilters  interface{} // ChatFilters
	CandidateCount    int
	ProviderDropped   bool

	// Pass 2 — recommendation generation
	RecommendStart  time.Time
	RecommendEnd    time.Time
	RecommendModel  string
	RecommendInput  interface{} // messages sent to sonnet (truncated)
	RecommendOutput interface{} // reply + recommendations

	// Final output
	Reply           string
	SuggestionCount int
	Error           string
}

// Send ships the trace to Langfuse asynchronously. Errors are logged, never returned.
func (c *Client) Send(trace ChatTrace) {
	if !c.Enabled() {
		return
	}
	go func() {
		if err := c.send(trace); err != nil {
			log.Warn().Err(err).Str("trace_id", trace.TraceID).Msg("langfuse: send failed")
		}
	}()
}

func (c *Client) send(t ChatTrace) error {
	now := time.Now()
	traceID := t.TraceID
	if traceID == "" {
		traceID = uuid.NewString()
	}

	meta := map[string]interface{}{
		"country":          t.Country,
		"provider_dropped": t.ProviderDropped,
		"candidate_count":  t.CandidateCount,
	}
	if t.Error != "" {
		meta["error"] = t.Error
	}

	events := []event{
		// ── Trace ────────────────────────────────────────────────────────────
		{
			ID:        uuid.NewString(),
			Type:      "trace-create",
			Timestamp: now,
			Body: traceBody{
				ID:        traceID,
				Name:      "chat",
				UserID:    t.UserID,
				SessionID: t.SessionID,
				Input:     map[string]string{"message": t.UserMessage},
				Output: map[string]interface{}{
					"reply":            t.Reply,
					"suggestion_count": t.SuggestionCount,
				},
				Metadata: meta,
			},
		},
		// ── Generation: intent extraction ─────────────────────────────────
		{
			ID:        uuid.NewString(),
			Type:      "generation-create",
			Timestamp: now,
			Body: generationBody{
				ID:        uuid.NewString(),
				TraceID:   traceID,
				Name:      "intent-extraction",
				StartTime: t.IntentStart,
				EndTime:   t.IntentEnd,
				Model:     t.IntentModel,
				Input:     t.IntentInput,
				Output:    t.IntentOutput,
				Metadata: map[string]interface{}{
					"latency_ms": t.IntentEnd.Sub(t.IntentStart).Milliseconds(),
				},
			},
		},
		// ── Span: Neo4j candidate fetch ───────────────────────────────────
		{
			ID:        uuid.NewString(),
			Type:      "span-create",
			Timestamp: now,
			Body: spanBody{
				ID:        uuid.NewString(),
				TraceID:   traceID,
				Name:      "neo4j-candidates",
				StartTime: t.CandidateStart,
				EndTime:   t.CandidateEnd,
				Input:     t.CandidateFilters,
				Output: map[string]interface{}{
					"count":            t.CandidateCount,
					"provider_dropped": t.ProviderDropped,
				},
				Metadata: map[string]interface{}{
					"latency_ms": t.CandidateEnd.Sub(t.CandidateStart).Milliseconds(),
				},
			},
		},
		// ── Generation: recommendation writing ────────────────────────────
		{
			ID:        uuid.NewString(),
			Type:      "generation-create",
			Timestamp: now,
			Body: generationBody{
				ID:        uuid.NewString(),
				TraceID:   traceID,
				Name:      "recommendation-generation",
				StartTime: t.RecommendStart,
				EndTime:   t.RecommendEnd,
				Model:     t.RecommendModel,
				Input:     t.RecommendInput,
				Output:    t.RecommendOutput,
				Level:     func() string {
					if t.Error != "" {
						return "ERROR"
					}
					return "DEFAULT"
				}(),
				Metadata: map[string]interface{}{
					"latency_ms":       t.RecommendEnd.Sub(t.RecommendStart).Milliseconds(),
					"suggestion_count": t.SuggestionCount,
				},
			},
		},
	}

	payload := map[string]interface{}{"batch": events}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		c.baseURL+"/api/public/ingestion",
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(c.publicKey, c.secretKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("langfuse returned %d", resp.StatusCode)
	}
	return nil
}
