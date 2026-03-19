// Package chat implements the AI film concierge service.
// It uses a two-pass approach:
//  1. Intent extraction — Claude parses the conversation into structured filters.
//  2. Recommendation — Neo4j candidates are injected into Claude's context;
//     Claude selects 3-5 films and writes personalised reasons.
package chat

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/foundrylab-app/cinova/backend/internal/config"
	"github.com/foundrylab-app/cinova/backend/internal/graph"
	"github.com/foundrylab-app/cinova/backend/internal/langfuse"
	"github.com/foundrylab-app/cinova/backend/internal/models"
	"github.com/foundrylab-app/cinova/backend/internal/store"
)

const (
	chatModel       = "claude-haiku-4-5-20251001"
	intentModel     = "claude-haiku-4-5-20251001"
	intentTimeout   = 15 * time.Second
	recommendTimeout = 60 * time.Second
	historyLimit    = 10 // turns kept as context
	defaultMinScore = 40.0
)

// ── Axon wire types ──────────────────────────────────────────────────────────

type axonMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type axonThinking struct {
	Type string `json:"type"`
}

type axonRequest struct {
	Model     string        `json:"model"`
	Messages  []axonMessage `json:"messages"`
	MaxTokens int           `json:"max_tokens"`
	Thinking  *axonThinking `json:"thinking,omitempty"`
}

type axonResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

type axonStreamRequest struct {
	Model     string        `json:"model"`
	Messages  []axonMessage `json:"messages"`
	MaxTokens int           `json:"max_tokens"`
	Stream    bool          `json:"stream"`
	Thinking  *axonThinking `json:"thinking,omitempty"`
}

type axonStreamChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
}

// ── Intent extraction ────────────────────────────────────────────────────────

type intentResult struct {
	NeedsClarification bool     `json:"needs_clarification"`
	ClarifyingQuestion string   `json:"clarifying_question,omitempty"`
	Genres             []string `json:"genres,omitempty"`
	Themes             []string `json:"themes,omitempty"`
	Moods              []string `json:"moods,omitempty"`
	Providers          []string `json:"providers,omitempty"`
	Directors          []string `json:"directors,omitempty"`
	Actors             []string `json:"actors,omitempty"`
	Language           string   `json:"language,omitempty"`
	MaxRuntime         int      `json:"max_runtime,omitempty"`
	MinYear            int      `json:"min_year,omitempty"`
	MaxYear            int      `json:"max_year,omitempty"`
	MinScore           float64  `json:"min_score,omitempty"`
}

const intentSystemPrompt = `You are an intent extraction assistant for a movie recommendation chatbot.
Analyze the conversation and extract what the user wants to watch.

Return ONLY valid JSON — no markdown, no code blocks:
{
  "needs_clarification": false,
  "clarifying_question": "",
  "genres": [],
  "themes": [],
  "moods": [],
  "providers": [],
  "directors": [],
  "actors": [],
  "language": "",
  "max_runtime": 0,
  "min_year": 0,
  "max_year": 0,
  "min_score": 40.0
}

Available genres: Action, Adventure, Animation, Comedy, Crime, Documentary, Drama, Fantasy, Horror, Mystery, Romance, Science Fiction, Thriller, Western
Example moods: Tense, Melancholic, Uplifting, Darkly Comic, Suspenseful, Whimsical, Intense, Heartwarming
Example themes: Redemption, Power & Corruption, Family Bonds, Survival, Identity, Coming of Age

Rules:
- ALWAYS set needs_clarification=false and return default filters if the message is meta/off-topic (e.g. "are you there", "hello", "ok", "thanks")
- Set needs_clarification=true ONLY when the message is a single word like "something" or "anything" with zero context AND it is the very first message
- "What should I watch tonight?", "I want a movie", "suggest something good" → needs_clarification=false, use default filters
- After ANY prior assistant message in history, NEVER ask for clarification — always produce filters
- Only ask ONE clarifying question, never multiple
- If user mentions a specific movie, infer its characteristics to find similar titles
- If user says "on Netflix" / "on HBO" set providers accordingly
- Default min_score is 40; raise to 70+ if user says "great" or "best"
- If user names a DIRECTOR (e.g. "Nuri Bilge Ceylan films", "Kubrick movies", "directed by Tarkovsky"), set directors to their name; do NOT filter by genre/mood unless also specified
- If user names an ACTOR (e.g. "movies with Tom Hanks", "starring Cate Blanchett"), set actors to their name
- IMPORTANT: Always write director and actor names using plain ASCII Latin letters only — no accents, diacritics, or special characters. Examples: "Francois Truffaut" not "François Truffaut", "Nuri Bilge Ceylan" not "Nurı Bilge Ceylan", "Alejandro Inarritu" not "Alejandro Iñárritu", "Yilmaz Guney" not "Yılmaz Güney"
- If user asks for films in a specific language (e.g. "Turkish films", "French cinema", "Japanese movies"), set language to the ISO 639-1 code: tr, fr, ja, ko, it, de, es, pt, ar, zh, etc.
- When directors, actors, or language are set, do NOT raise min_score — keep it at 40 or lower so niche/arthouse films are included`

// ── Recommendation output ────────────────────────────────────────────────────

type recEntry struct {
	TMDBID int64  `json:"tmdb_id"`
	Title  string `json:"title"`
	Reason string `json:"reason"`
}

type recOutput struct {
	Reply           string     `json:"reply"`
	Recommendations []recEntry `json:"recommendations"`
}

// recStreamSystemPromptTemplate is the streaming variant — reply text precedes
// the "|||" separator, JSON array follows. This lets us forward text deltas
// immediately while collecting the JSON silently.
const recStreamSystemPromptTemplate = `You are Cinova's AI film concierge — knowledgeable, warm, and opinionated.

%s

From the candidate films below select 3-5 that best match the conversation.

Respond in EXACTLY this format — nothing else:
Write a warm 1-2 sentence intro (plain text).|||
[{"tmdb_id":123,"title":"Title","reason":"2-3 sentence personalised reason"}]

Rules:
- If the user message is meta/off-topic (e.g. "are you there", "hello"), respond warmly and pivot to suggesting the candidates as if they asked "what should I watch?"
- Reference streaming availability when mentioning a film ("Available on Netflix")
- Be specific and opinionated — never say just "it's a great movie"
- Do NOT reveal major spoilers or endings`

const recSystemPromptTemplate = `You are Cinova's AI film concierge — knowledgeable, warm, and opinionated.

%s

From the candidate films below select 3-5 that best match the conversation. Write personalised reasons.

Return ONLY valid JSON — no markdown:
{
  "reply": "1-2 sentence warm intro acknowledging what the user wants",
  "recommendations": [
    { "tmdb_id": 123, "title": "Title", "reason": "2-3 sentence personalised reason" }
  ]
}

Rules:
- If the user message is meta/off-topic (e.g. "are you there", "hello", "ok"), respond warmly and pivot to suggesting the candidates as if they asked "what should I watch?"
- Reference streaming availability when mentioning a film ("Available on Netflix")
- Acknowledge user's emotional state briefly if expressed before recommending
- Reference user's saved/rated titles if relevant ("Given you saved Inception...")
- Be specific and opinionated — never say just "it's a great movie"
- Do NOT reveal major spoilers or endings`

// ── Service ──────────────────────────────────────────────────────────────────

// Service is the AI film concierge chat service.
type Service struct {
	repo     *graph.MovieRepository
	pg       *store.PostgresStore
	cfg      *config.Config
	client   *http.Client
	tracer   *langfuse.Client
}

// New creates a new chat Service.
func New(repo *graph.MovieRepository, pg *store.PostgresStore, cfg *config.Config) *Service {
	return &Service{
		repo:   repo,
		pg:     pg,
		cfg:    cfg,
		client: &http.Client{Timeout: recommendTimeout},
		tracer: langfuse.NewClient(cfg.LangfuseURL, cfg.LangfusePublicKey, cfg.LangfuseSecretKey),
	}
}

// Chat handles a single user turn. It saves both the user message and the
// assistant reply to postgres, then returns the reply with movie suggestions.
func (s *Service) Chat(
	ctx context.Context,
	userID, sessionID, convID, country string,
	history []models.ChatMessage,
	newMessage string,
) (*models.ChatResponse, error) {

	// ── Pass 1: extract intent ─────────────────────────────────────────────
	intent, _, err := s.extractIntent(ctx, history, newMessage)
	if err != nil {
		log.Warn().Err(err).Msg("chat: intent extraction failed, using defaults")
		intent = &intentResult{MinScore: defaultMinScore}
	}

	// If Claude wants to ask a clarifying question, still fetch popular candidates
	// so the user sees movie cards alongside the question (never a blank response).
	if intent.NeedsClarification && intent.ClarifyingQuestion != "" {
		popular, _ := s.repo.GetChatCandidates(ctx, graph.ChatFilters{MinScore: defaultMinScore}, country)
		s.persistMessages(ctx, userID, sessionID, newMessage, intent.ClarifyingQuestion)
		return &models.ChatResponse{
			Reply:       intent.ClarifyingQuestion,
			Suggestions: popular,
			ConvID:      convID,
		}, nil
	}

	// ── Fetch personalisation context ─────────────────────────────────────
	ownerType := "Session"
	ownerID := sessionID
	if userID != "" {
		ownerType = "User"
		ownerID = userID
	}
	userCtx, _ := s.repo.GetUserChatContext(ctx, ownerID, ownerType)

	// ── Pass 1b: build graph filters ──────────────────────────────────────
	minScore := intent.MinScore
	if minScore <= 0 {
		minScore = defaultMinScore
	}
	// No score floor when the user is searching by specific person or language —
	// every ingested title by that director/actor/in that language must be shown.
	if len(intent.Directors) > 0 || len(intent.Actors) > 0 || intent.Language != "" {
		minScore = 0
	}
	filters := graph.ChatFilters{
		Genres:     intent.Genres,
		Themes:     intent.Themes,
		Moods:      intent.Moods,
		Providers:  intent.Providers,
		Directors:  intent.Directors,
		Actors:     intent.Actors,
		Language:   intent.Language,
		MaxRuntime: intent.MaxRuntime,
		MinYear:    intent.MinYear,
		MaxYear:    intent.MaxYear,
		MinScore:   minScore,
	}

	// ── Query Neo4j for candidates (tiered fallback) ──────────────────────
	candidates, providerDropped := s.fetchCandidates(ctx, filters, country)

	// ── Pass 2: write recommendations ─────────────────────────────────────
	recOut, err := s.writeRecommendations(ctx, history, newMessage, userCtx, candidates, country, filters.Providers, providerDropped)
	if err != nil {
		log.Error().Err(err).Msg("chat: recommendation generation failed")
		// Graceful fallback: return candidate titles without AI commentary
		recOut = s.buildFallbackResponse(candidates)
	}

	// Map recommended TMDB IDs back to enriched candidates
	suggMap := make(map[int64]models.MovieSummary, len(candidates))
	for _, c := range candidates {
		suggMap[c.TMDBID] = c
	}
	suggestions := make([]models.MovieSummary, 0, len(recOut.Recommendations))
	for _, rec := range recOut.Recommendations {
		if c, ok := suggMap[rec.TMDBID]; ok {
			c.Reason = rec.Reason
			suggestions = append(suggestions, c)
		}
	}

	assistantRecord := recOut.Reply
	if len(suggestions) > 0 {
		titles := make([]string, 0, len(suggestions))
		for _, sg := range suggestions {
			titles = append(titles, sg.Title)
		}
		assistantRecord += "\n\n[Recommended: " + strings.Join(titles, ", ") + "]"
	}
	s.persistMessages(ctx, userID, sessionID, newMessage, assistantRecord)

	return &models.ChatResponse{
		Reply:       recOut.Reply,
		Suggestions: suggestions,
		ConvID:      convID,
	}, nil
}

// ── Internal helpers ─────────────────────────────────────────────────────────

// extractIntent calls Claude to parse the conversation into structured filters.
// It also returns the full messages array sent to Axon so callers can attach it to traces.
func (s *Service) extractIntent(ctx context.Context, history []models.ChatMessage, newMsg string) (*intentResult, []axonMessage, error) {
	messages := []axonMessage{{Role: "system", Content: intentSystemPrompt}}
	for _, h := range history {
		messages = append(messages, axonMessage{Role: h.Role, Content: h.Content})
	}
	messages = append(messages, axonMessage{Role: "user", Content: newMsg})

	intentCtx, cancel := context.WithTimeout(ctx, intentTimeout)
	defer cancel()

	raw, err := s.callAxonWithModel(intentCtx, messages, 400, intentModel)
	if err != nil {
		return nil, messages, err
	}

	var result intentResult
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return nil, messages, fmt.Errorf("parse intent: %w (raw: %.200s)", err, raw)
	}
	return &result, messages, nil
}

// writeRecommendations builds the candidate list and calls Claude to produce
// natural language recommendations. requestedProviders and providerDropped let
// Claude acknowledge when requested streaming services had no matches.
func (s *Service) writeRecommendations(
	ctx context.Context,
	history []models.ChatMessage,
	newMsg string,
	userCtx *graph.UserChatContext,
	candidates []models.MovieSummary,
	country string,
	requestedProviders []string,
	providerDropped bool,
) (*recOutput, error) {

	// Build user context section
	var ctxParts []string
	if userCtx != nil {
		if len(userCtx.TopThemes) > 0 {
			ctxParts = append(ctxParts, "User's favourite themes: "+strings.Join(userCtx.TopThemes, ", "))
		}
		if len(userCtx.SavedTitles) > 0 {
			ctxParts = append(ctxParts, "User's watchlist includes: "+strings.Join(userCtx.SavedTitles, ", "))
		}
	}
	ctxSection := ""
	if len(ctxParts) > 0 {
		ctxSection = "User context:\n" + strings.Join(ctxParts, "\n")
	}
	sysPrompt := fmt.Sprintf(recSystemPromptTemplate, ctxSection)

	// Build candidate list for the prompt
	var candidateSB strings.Builder
	candidateSB.WriteString("Candidate films:\n")
	for _, c := range candidates {
		genres := make([]string, 0, len(c.Genres))
		for _, g := range c.Genres {
			genres = append(genres, g.Name)
		}
		providers := make([]string, 0, len(c.Providers))
		seen := map[string]bool{}
		for _, p := range c.Providers {
			if p.Type == "flatrate" && !seen[p.ProviderName] {
				providers = append(providers, p.ProviderName)
				seen[p.ProviderName] = true
			}
		}
		// Use a short teaser (first 2 sentences) to keep prompt tokens low.
		// The full synopsis is returned to the client separately.
		synopsis := c.CinovaSynopsis
		if synopsis == "" {
			synopsis = c.Overview
		}
		if sentences := strings.SplitN(synopsis, ". ", 3); len(sentences) >= 2 {
			synopsis = sentences[0] + ". " + sentences[1] + "."
		}
		line := fmt.Sprintf("- [%d] %s (%s) Score:%.0f Genres:%s Streaming:%s | %s",
			c.TMDBID, c.Title, c.ReleaseYear, c.CinovaScore,
			strings.Join(genres, "/"),
			strings.Join(providers, "/"),
			synopsis,
		)
		candidateSB.WriteString(line + "\n")
	}

	// If the user asked for a specific provider but it couldn't be satisfied,
	// prepend a note so Claude doesn't falsely claim the films are on that service.
	providerNote := ""
	if providerDropped && len(requestedProviders) > 0 {
		providerNote = fmt.Sprintf(
			"[System note: No titles matching %s were found in this region. "+
				"The candidates below are from other services. Acknowledge this briefly and "+
				"still recommend the best matches.]\n\n",
			strings.Join(requestedProviders, "/"),
		)
	}

	// Combine history + new message + candidates
	messages := []axonMessage{{Role: "system", Content: sysPrompt}}
	for _, h := range history {
		messages = append(messages, axonMessage{Role: h.Role, Content: h.Content})
	}
	userContent := providerNote + newMsg + "\n\n" + candidateSB.String()
	messages = append(messages, axonMessage{Role: "user", Content: userContent})

	raw, err := s.callAxon(ctx, messages, 1500)
	if err != nil {
		return nil, err
	}

	var out recOutput
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, fmt.Errorf("parse recommendations: %w (raw: %.200s)", err, raw)
	}
	return &out, nil
}

// callAxon sends a chat completions request and returns the text content.
func (s *Service) callAxon(ctx context.Context, messages []axonMessage, maxTokens int) (string, error) {
	return s.callAxonWithModel(ctx, messages, maxTokens, chatModel)
}

func (s *Service) callAxonWithModel(ctx context.Context, messages []axonMessage, maxTokens int, model string) (string, error) {
	reqBody := axonRequest{
		Model:     model,
		Messages:  messages,
		MaxTokens: maxTokens,
		Thinking:  &axonThinking{Type: "disabled"},
	}
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal axon request: %w", err)
	}

	endpoint, err := url.JoinPath(s.cfg.AxonURL, "/v1/chat/completions")
	if err != nil {
		return "", fmt.Errorf("build axon url: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("build axon request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if s.cfg.AxonAPIKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.cfg.AxonAPIKey)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("axon http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return "", fmt.Errorf("axon returned %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}

	var axonResp axonResponse
	if err := json.NewDecoder(resp.Body).Decode(&axonResp); err != nil {
		return "", fmt.Errorf("decode axon response: %w", err)
	}
	if len(axonResp.Choices) == 0 {
		return "", fmt.Errorf("empty choices in axon response")
	}

	content := axonResp.Choices[0].Message.Content
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	return strings.TrimSpace(content), nil
}

// PersistMessages is the exported version of persistMessages for use by external handlers.
func (s *Service) PersistMessages(ctx context.Context, userID, sessionID, userMsg, assistantMsg string) {
	s.persistMessages(ctx, userID, sessionID, userMsg, assistantMsg)
}

// persistMessages saves the user message and assistant reply to postgres.
// Errors are logged but not returned — persistence failure should not block the user.
func (s *Service) persistMessages(ctx context.Context, userID, sessionID, userMsg, assistantMsg string) {
	if _, err := s.pg.SaveChatMessage(ctx, userID, sessionID, "user", userMsg); err != nil {
		log.Warn().Err(err).Msg("chat: failed to save user message")
	}
	if _, err := s.pg.SaveChatMessage(ctx, userID, sessionID, "assistant", assistantMsg); err != nil {
		log.Warn().Err(err).Msg("chat: failed to save assistant message")
	}
}

// ── Streaming ─────────────────────────────────────────────────────────────────

// StreamChat runs the full chat pipeline and writes SSE events to w:
//
//	data: {"type":"delta","text":"..."}                — reply text chunks
//	data: {"type":"suggestions","items":[...],"conv_id":"..."}
//	data: {"type":"done"}
func (s *Service) StreamChat(
	ctx context.Context,
	userID, sessionID, convID, country string,
	history []models.ChatMessage,
	newMessage string,
	w http.ResponseWriter,
	flusher http.Flusher,
) error {
	traceID := uuid.NewString()
	trace := langfuse.ChatTrace{
		TraceID:      traceID,
		UserID:       userID,
		SessionID:    sessionID,
		Country:      country,
		UserMessage:  newMessage,
		RecommendModel: chatModel,
		IntentModel:    intentModel,
	}

	sendSSE := func(v interface{}) {
		b, _ := json.Marshal(v)
		fmt.Fprintf(w, "data: %s\n\n", b)
		flusher.Flush()
	}

	// Send initial status so the client knows we're working
	sendSSE(map[string]string{"type": "status", "text": "Analyzing your request…"})

	// Pass 1: extract intent
	intentStart := time.Now()
	intent, intentMessages, err := s.extractIntent(ctx, history, newMessage)
	trace.IntentStart = intentStart
	trace.IntentEnd = time.Now()
	trace.IntentInput = intentMessages // full messages array — visible in Langfuse
	if err != nil {
		log.Warn().Err(err).Msg("chat stream: intent extraction failed, using defaults")
		intent = &intentResult{MinScore: defaultMinScore}
		trace.IntentOutput = map[string]string{"error": err.Error()}
	} else {
		trace.IntentOutput = intent
	}

	// Clarification path — stream the question then send popular suggestions
	if intent.NeedsClarification && intent.ClarifyingQuestion != "" {
		popular, _ := s.repo.GetChatCandidates(ctx, graph.ChatFilters{MinScore: defaultMinScore}, country)
		sendSSE(map[string]string{"type": "delta", "text": intent.ClarifyingQuestion})
		sendSSE(map[string]interface{}{"type": "suggestions", "items": popular, "conv_id": convID})
		sendSSE(map[string]string{"type": "done"})
		s.persistMessages(ctx, userID, sessionID, newMessage, intent.ClarifyingQuestion)
		return nil
	}

	// Personalisation context
	ownerType, ownerID := "Session", sessionID
	if userID != "" {
		ownerType, ownerID = "User", userID
	}
	userCtx, _ := s.repo.GetUserChatContext(ctx, ownerID, ownerType)

	// Build filters
	minScore := intent.MinScore
	if minScore <= 0 {
		minScore = defaultMinScore
	}
	// No score floor for director/actor/language searches
	if len(intent.Directors) > 0 || len(intent.Actors) > 0 || intent.Language != "" {
		minScore = 0
	}
	filters := graph.ChatFilters{
		Genres:     intent.Genres,
		Themes:     intent.Themes,
		Moods:      intent.Moods,
		Providers:  intent.Providers,
		Directors:  intent.Directors,
		Actors:     intent.Actors,
		Language:   intent.Language,
		MaxRuntime: intent.MaxRuntime,
		MinYear:    intent.MinYear,
		MaxYear:    intent.MaxYear,
		MinScore:   minScore,
	}

	sendSSE(map[string]string{"type": "status", "text": "Searching films…"})

	// Neo4j query (tiered fallback)
	candidateStart := time.Now()
	candidates, providerDropped := s.fetchCandidates(ctx, filters, country)
	trace.CandidateStart = candidateStart
	trace.CandidateEnd = time.Now()
	trace.CandidateFilters = filters
	trace.CandidateCount = len(candidates)
	trace.ProviderDropped = providerDropped

	sendSSE(map[string]string{"type": "status", "text": "Writing recommendations…"})

	// Build candidate lookup map
	suggMap := make(map[int64]models.MovieSummary, len(candidates))
	for _, c := range candidates {
		suggMap[c.TMDBID] = c
	}

	// Build Axon messages (history truncated to prevent context bloat)
	messages := s.buildStreamMessages(history, newMessage, userCtx, candidates, filters.Providers, providerDropped)

	// Stream recommendation — deltas forwarded to client as text arrives
	replyCtx, cancel := context.WithTimeout(ctx, recommendTimeout)
	defer cancel()

	trace.RecommendInput = map[string]interface{}{
		"candidate_count": len(candidates),
		"history_len":     len(history),
		"provider_dropped": providerDropped,
	}
	recommendStart := time.Now()
	reply, recJSONStr, err := s.streamAxonToSSE(replyCtx, messages, 1500, w, flusher)
	trace.RecommendStart = recommendStart
	trace.RecommendEnd = time.Now()
	if err != nil {
		log.Error().Err(err).Msg("chat stream: axon error, sending fallback")
		trace.Error = err.Error()
		s.tracer.Send(trace)
		fallback := s.buildFallbackResponse(candidates)
		sendSSE(map[string]string{"type": "delta", "text": fallback.Reply})
		sendSSE(map[string]interface{}{"type": "suggestions", "items": candidates[:min(5, len(candidates))], "conv_id": convID})
		sendSSE(map[string]string{"type": "done"})
		return nil
	}

	// Parse recommendations JSON
	var recs []recEntry
	if err := json.Unmarshal([]byte(recJSONStr), &recs); err != nil {
		log.Warn().Err(err).Str("raw", recJSONStr[:min(200, len(recJSONStr))]).Msg("chat stream: parse recs failed, sending candidates as-is")
	}

	// Map recs to enriched candidates
	suggestions := make([]models.MovieSummary, 0, len(recs))
	for _, rec := range recs {
		if c, ok := suggMap[rec.TMDBID]; ok {
			c.Reason = rec.Reason
			suggestions = append(suggestions, c)
		}
	}
	// Fallback if no matching recs
	if len(suggestions) == 0 {
		for i, c := range candidates {
			if i >= 5 {
				break
			}
			suggestions = append(suggestions, c)
		}
	}

	sendSSE(map[string]interface{}{"type": "suggestions", "items": suggestions, "conv_id": convID})
	sendSSE(map[string]string{"type": "done"})

	// Persist with recommended titles appended so future history turns
	// let Claude know which specific films were already shown.
	assistantRecord := reply
	titles := make([]string, 0, len(suggestions))
	for _, sg := range suggestions {
		titles = append(titles, sg.Title)
	}
	if len(titles) > 0 {
		assistantRecord += "\n\n[Recommended: " + strings.Join(titles, ", ") + "]"
	}
	s.persistMessages(ctx, userID, sessionID, newMessage, assistantRecord)

	// Fire trace to Langfuse (async, non-blocking)
	trace.Reply = reply
	trace.SuggestionCount = len(suggestions)
	trace.RecommendOutput = map[string]interface{}{
		"reply":         reply,
		"titles":        titles,
		"rec_json_size": len(recJSONStr),
	}
	s.tracer.Send(trace)

	return nil
}

// buildStreamMessages constructs Axon messages using the streaming prompt
// format and truncated history to prevent context bloat.
func (s *Service) buildStreamMessages(
	history []models.ChatMessage,
	newMsg string,
	userCtx *graph.UserChatContext,
	candidates []models.MovieSummary,
	requestedProviders []string,
	providerDropped bool,
) []axonMessage {
	var ctxParts []string
	if userCtx != nil {
		if len(userCtx.TopThemes) > 0 {
			ctxParts = append(ctxParts, "User's favourite themes: "+strings.Join(userCtx.TopThemes, ", "))
		}
		if len(userCtx.SavedTitles) > 0 {
			ctxParts = append(ctxParts, "User's watchlist includes: "+strings.Join(userCtx.SavedTitles, ", "))
		}
	}
	ctxSection := ""
	if len(ctxParts) > 0 {
		ctxSection = "User context:\n" + strings.Join(ctxParts, "\n")
	}
	sysPrompt := fmt.Sprintf(recStreamSystemPromptTemplate, ctxSection)

	var sb strings.Builder
	sb.WriteString("Candidate films:\n")
	for _, c := range candidates {
		genres := make([]string, 0, len(c.Genres))
		for _, g := range c.Genres {
			genres = append(genres, g.Name)
		}
		providers := make([]string, 0)
		seen := map[string]bool{}
		for _, p := range c.Providers {
			if p.Type == "flatrate" && !seen[p.ProviderName] {
				providers = append(providers, p.ProviderName)
				seen[p.ProviderName] = true
			}
		}
		synopsis := c.CinovaSynopsis
		if synopsis == "" {
			synopsis = c.Overview
		}
		if parts := strings.SplitN(synopsis, ". ", 3); len(parts) >= 2 {
			synopsis = parts[0] + ". " + parts[1] + "."
		}
		line := fmt.Sprintf("- [%d] %s (%s) Score:%.0f Genres:%s Streaming:%s | %s",
			c.TMDBID, c.Title, c.ReleaseYear, c.CinovaScore,
			strings.Join(genres, "/"),
			strings.Join(providers, "/"),
			synopsis,
		)
		sb.WriteString(line + "\n")
	}

	msgs := []axonMessage{{Role: "system", Content: sysPrompt}}
	for _, h := range history {
		content := h.Content
		// Truncate prior assistant messages to 2 sentences to prevent context bloat
		if h.Role == "assistant" {
			if parts := strings.SplitN(content, ". ", 3); len(parts) >= 2 {
				content = parts[0] + ". " + parts[1] + "."
			}
		}
		msgs = append(msgs, axonMessage{Role: h.Role, Content: content})
	}
	providerNote := ""
	if providerDropped && len(requestedProviders) > 0 {
		providerNote = fmt.Sprintf(
			"[System note: No titles matching %s were found in this region. "+
				"The candidates below are from other services. Acknowledge this briefly and "+
				"still recommend the best matches.]\n\n",
			strings.Join(requestedProviders, "/"),
		)
	}
	msgs = append(msgs, axonMessage{Role: "user", Content: providerNote + newMsg + "\n\n" + sb.String()})
	return msgs
}

/// streamAxonToSSE sends a streaming request to Axon and:
//   - Forwards text deltas (before the "|||" separator) to the SSE client immediately
//   - Collects the JSON recommendations (after "|||") and returns them
//
// Returns (replyText, recJSON, error).
func (s *Service) streamAxonToSSE(
	ctx context.Context,
	messages []axonMessage,
	maxTokens int,
	w http.ResponseWriter,
	flusher http.Flusher,
) (reply string, recJSON string, err error) {
	reqBody := axonStreamRequest{
		Model:     chatModel,
		Messages:  messages,
		MaxTokens: maxTokens,
		Stream:    true,
		Thinking:  &axonThinking{Type: "disabled"},
	}
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return "", "", fmt.Errorf("marshal stream request: %w", err)
	}

	endpoint, err := url.JoinPath(s.cfg.AxonURL, "/v1/chat/completions")
	if err != nil {
		return "", "", fmt.Errorf("build axon url: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return "", "", fmt.Errorf("build stream request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	if s.cfg.AxonAPIKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.cfg.AxonAPIKey)
	}

	// No global timeout on streaming client — context controls it
	streamClient := &http.Client{}
	resp, err := streamClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("axon stream http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return "", "", fmt.Errorf("axon stream returned %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}

	const separator = "|||"
	var accumulated strings.Builder
	sentBytes := 0      // bytes of accumulated text already forwarded as deltas
	foundSep := false   // true once "|||" detected
	separatorPos := -1  // byte offset of "|||" in accumulated

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var chunk axonStreamChunk
		if jsonErr := json.Unmarshal([]byte(data), &chunk); jsonErr != nil {
			continue
		}
		if len(chunk.Choices) == 0 {
			continue
		}
		text := chunk.Choices[0].Delta.Content
		if text == "" {
			continue
		}

		accumulated.WriteString(text)
		full := accumulated.String()

		if foundSep {
			// After separator — just accumulate, don't forward
			continue
		}

		sepIdx := strings.Index(full, separator)
		if sepIdx >= 0 {
			// Found separator — send any remaining pre-separator text
			foundSep = true
			separatorPos = sepIdx
			if sepIdx > sentBytes {
				delta := strings.TrimSpace(full[sentBytes:sepIdx])
				if delta != "" {
					b, _ := json.Marshal(map[string]string{"type": "delta", "text": delta})
					fmt.Fprintf(w, "data: %s\n\n", b)
					flusher.Flush()
				}
				sentBytes = sepIdx
			}
		} else {
			// Separator not yet found — forward new content, keeping last 2 bytes
			// buffered to avoid splitting "|||" across chunks.
			// Retreat safeEnd to a valid UTF-8 rune start to avoid sending
			// incomplete multibyte characters (which show as diamond question marks).
			safeEnd := len(full) - (len(separator) - 1)
			for safeEnd > sentBytes && safeEnd < len(full) && !utf8.RuneStart(full[safeEnd]) {
				safeEnd--
			}
			if safeEnd > sentBytes {
				delta := full[sentBytes:safeEnd]
				if delta != "" {
					b, _ := json.Marshal(map[string]string{"type": "delta", "text": delta})
					fmt.Fprintf(w, "data: %s\n\n", b)
					flusher.Flush()
					sentBytes = safeEnd
				}
			}
		}
	}

	if scanErr := scanner.Err(); scanErr != nil {
		return "", "", fmt.Errorf("stream scan: %w", scanErr)
	}

	full := accumulated.String()

	if !foundSep {
		// No separator — treat all content as reply text
		remaining := strings.TrimSpace(full[sentBytes:])
		if remaining != "" {
			b, _ := json.Marshal(map[string]string{"type": "delta", "text": remaining})
			fmt.Fprintf(w, "data: %s\n\n", b)
			flusher.Flush()
		}
		return strings.TrimSpace(full), "", nil
	}

	replyText := strings.TrimSpace(full[:separatorPos])
	afterSep := strings.TrimSpace(full[separatorPos+len(separator):])

	// Extract JSON array from after-separator content
	jsonStart := strings.Index(afterSep, "[")
	jsonEnd := strings.LastIndex(afterSep, "]")
	if jsonStart >= 0 && jsonEnd > jsonStart {
		afterSep = afterSep[jsonStart : jsonEnd+1]
	} else {
		afterSep = ""
	}

	return replyText, afterSep, nil
}

// fetchCandidates runs a tiered fallback to always return ≥3 candidates:
//  1. Full filters (genres + providers + score)
//  2. Provider kept, content filters relaxed (if < 3)
//  3. All filters dropped (if still < 3) — providerDropped=true so the caller
//     can tell Claude the provider couldn't be satisfied
func (s *Service) fetchCandidates(ctx context.Context, filters graph.ChatFilters, country string) (candidates []models.MovieSummary, providerDropped bool) {
	var err error
	candidates, err = s.repo.GetChatCandidates(ctx, filters, country)
	if err != nil {
		log.Error().Err(err).Msg("chat: candidate query failed")
		candidates = []models.MovieSummary{}
	}

	// Tier 2: keep provider, relax content filters
	if len(candidates) < 3 && len(filters.Providers) > 0 {
		log.Info().Int("n", len(candidates)).Msg("chat: relaxing content filters, keeping provider")
		providerOnly := graph.ChatFilters{
			Providers:  filters.Providers,
			MinScore:   defaultMinScore,
			ExcludeIDs: filters.ExcludeIDs,
		}
		if more, err2 := s.repo.GetChatCandidates(ctx, providerOnly, country); err2 == nil && len(more) > len(candidates) {
			candidates = more
		}
	}

	// Tier 3: drop all filters — skip if a specific director/actor was requested
	// and we already have at least one matching candidate; the AI handles the thin set gracefully.
	hasPersonFilter := len(filters.Directors) > 0 || len(filters.Actors) > 0
	if len(candidates) < 3 && !(hasPersonFilter && len(candidates) > 0) {
		log.Info().Int("n", len(candidates)).Msg("chat: dropping all filters")
		bare := graph.ChatFilters{MinScore: defaultMinScore, ExcludeIDs: filters.ExcludeIDs}
		if more, err2 := s.repo.GetChatCandidates(ctx, bare, country); err2 == nil && len(more) > len(candidates) {
			if len(filters.Providers) > 0 {
				providerDropped = true
			}
			candidates = more
		}
	}
	return candidates, providerDropped
}

// FetchCandidates is the exported version of fetchCandidates for use by the
// internal candidates endpoint (called by Langflow during pipeline execution).
func (s *Service) FetchCandidates(ctx context.Context, filters graph.ChatFilters, country string) ([]models.MovieSummary, bool) {
	return s.fetchCandidates(ctx, filters, country)
}

// buildFallbackResponse produces a plain list response when AI generation fails.
func (s *Service) buildFallbackResponse(candidates []models.MovieSummary) *recOutput {
	recs := make([]recEntry, 0, 5)
	for i, c := range candidates {
		if i >= 5 {
			break
		}
		recs = append(recs, recEntry{TMDBID: c.TMDBID, Title: c.Title, Reason: c.CinovaSynopsis})
	}
	return &recOutput{
		Reply:           "Here are some films you might enjoy:",
		Recommendations: recs,
	}
}
