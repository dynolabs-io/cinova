package enrichment

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/foundrylab-app/cinova/backend/internal/graph"
	"github.com/foundrylab-app/cinova/backend/internal/models"
)

const (
	axonBatchTimeout = 300 * time.Second
	batchSize        = 10 // smaller batch — Sonnet 4.6 generates more tokens per item

	// Sonnet 4.6 — superior semantic understanding for enrichment quality
	enrichmentModel = "claude-sonnet-4-6"

	enrichmentSystemPrompt = `You are an editorial enrichment assistant for Cinova, a film and TV discovery platform. You receive structured metadata for each title including: title, tagline, TMDB overview, keywords, and when available a full Wikipedia plot_summary (400-800 words).

For each item produce ALL of the following:

- themes: 3-6 precise thematic labels with confidence scores. Be specific and meaningful — e.g. "Power & Corruption", "Father-Son Legacy", "Moral Decline", "Survival Against Odds". Avoid generic single words like "Love" or "War".

- moods: 2-4 emotional/tonal labels with confidence scores that reflect the actual viewing experience — e.g. "Tense", "Melancholic", "Darkly Comic", "Epic", "Dreamlike", "Unsettling".

- cinova_synopsis: 3-4 sentences in present tense. A spoiler-free hook using the full context. Cover what the story is fundamentally about, the emotional register, and what makes it distinctive. Do NOT reveal endings, deaths, or major twists. Write in English regardless of original language.

- cinova_editorial: 150-250 words. A rich editorial written in Cinova's voice — authoritative, warm, and precise. Structure it as: (1) what kind of film/show this is and who made it, (2) what the story explores thematically (draw on plot_summary for accuracy), (3) what the viewing experience feels like, (4) who will love it and why it matters. Present tense. No spoilers. English only.

Return ONLY a valid JSON object — no markdown, no code blocks, no explanation:
{"results": [{"tmdb_id": <number>, "themes": [{"name": <string>, "score": <0.0-1.0>}], "moods": [{"name": <string>, "score": <0.0-1.0>}], "cinova_synopsis": <string>, "cinova_editorial": <string>}]}`
)

// Client calls the Axon AI service (OpenAI-compatible) to extract themes and moods.
type Client struct {
	axonURL    string
	axonAPIKey string
	model      string
	httpClient *http.Client
}

// NewClient creates an enrichment Client.
func NewClient(axonURL, axonAPIKey string) *Client {
	return &Client{
		axonURL:    axonURL,
		axonAPIKey: axonAPIKey,
		model:      enrichmentModel,
		httpClient: &http.Client{Timeout: axonBatchTimeout},
	}
}

// NewClientWithModel creates an enrichment Client with a custom model.
func NewClientWithModel(axonURL, axonAPIKey, model string) *Client {
	c := NewClient(axonURL, axonAPIKey)
	if model != "" {
		c.model = model
	}
	return c
}

// ---- OpenAI-compatible request/response structs ----

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model          string        `json:"model"`
	Messages       []chatMessage `json:"messages"`
	MaxTokens      int           `json:"max_tokens"`
	ResponseFormat *respFormat   `json:"response_format,omitempty"`
}

type respFormat struct {
	Type string `json:"type"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// ---- Enrichment result structs ----

// labelField captures both "name" and "label" since the model sometimes uses either.
type labelField struct {
	Name  string  `json:"name"`
	Label string  `json:"label"`
	Score float64 `json:"score"`
}

func (l labelField) text() string {
	if l.Name != "" {
		return l.Name
	}
	return l.Label
}

type enrichItem struct {
	TMDBID          int64        `json:"tmdb_id"`
	Themes          []labelField `json:"themes"`
	Moods           []labelField `json:"moods"`
	CinovaSynopsis  string       `json:"cinova_synopsis"`
	CinovaEditorial string       `json:"cinova_editorial"`
}

type enrichResponse struct {
	Results []enrichItem `json:"results"`
}

// batchInput is a single item sent to the model.
// Include as much context as available — Sonnet uses all of it for richer output.
type batchInput struct {
	TMDBID      int64    `json:"tmdb_id"`
	Title       string   `json:"title"`
	Tagline     string   `json:"tagline,omitempty"`
	Overview    string   `json:"overview"`
	Keywords    []string `json:"keywords,omitempty"`
	PlotSummary string   `json:"plot_summary,omitempty"`
}

// ProcessMovieBatch enriches a batch of movies with AI-extracted themes and moods,
// then upserts Theme/Mood nodes and relationships into Neo4j.
func (c *Client) ProcessMovieBatch(ctx context.Context, movies []models.Movie, repo *graph.MovieRepository) error {
	for i := 0; i < len(movies); i += batchSize {
		end := i + batchSize
		if end > len(movies) {
			end = len(movies)
		}
		sub := movies[i:end]

		if err := c.enrichBatch(ctx, sub, repo, "movie"); err != nil {
			log.Warn().Err(err).Int("offset", i).Msg("batch enrichment failed, continuing")
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}
	return nil
}

// ProcessTVBatch enriches a batch of TV shows with AI-extracted themes and moods.
func (c *Client) ProcessTVBatch(ctx context.Context, shows []models.TVShow, repo *graph.MovieRepository) error {
	// Convert TVShow to Movie-like structs for unified enrichment
	movies := make([]models.Movie, 0, len(shows))
	for _, s := range shows {
		movies = append(movies, models.Movie{
			TMDBID:      s.TMDBID,
			Title:       s.Name,
			Tagline:     s.Tagline,
			Overview:    s.Overview,
			Keywords:    s.Keywords,
			PlotSummary: s.PlotSummary,
		})
	}
	for i := 0; i < len(movies); i += batchSize {
		end := i + batchSize
		if end > len(movies) {
			end = len(movies)
		}
		sub := movies[i:end]

		if err := c.enrichBatch(ctx, sub, repo, "tvshow"); err != nil {
			log.Warn().Err(err).Int("offset", i).Msg("tv batch enrichment failed, continuing")
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}
	return nil
}

// enrichBatch sends one batch of up to batchSize items to Axon and upserts results.
func (c *Client) enrichBatch(ctx context.Context, movies []models.Movie, repo *graph.MovieRepository, mediaType string) error {
	// Build input — skip movies with no overview
	inputs := make([]batchInput, 0, len(movies))
	for _, m := range movies {
		if m.Overview == "" {
			continue
		}
		kwNames := make([]string, 0, len(m.Keywords))
		for _, kw := range m.Keywords {
			kwNames = append(kwNames, kw.Name)
		}
		inputs = append(inputs, batchInput{
			TMDBID:      m.TMDBID,
			Title:       m.Title,
			Tagline:     m.Tagline,
			Overview:    m.Overview,
			Keywords:    kwNames,
			PlotSummary: m.PlotSummary,
		})
	}
	if len(inputs) == 0 {
		return nil
	}

	inputJSON, err := json.Marshal(inputs)
	if err != nil {
		return fmt.Errorf("marshal batch input: %w", err)
	}

	userContent := fmt.Sprintf("Enrich these %d items:\n%s", len(inputs), string(inputJSON))

	results, err := c.callChatCompletions(ctx, enrichmentSystemPrompt, userContent, 12000)
	if err != nil {
		return fmt.Errorf("chat completions: %w", err)
	}

	items, err := parseEnrichResponse(results)
	if err != nil {
		return fmt.Errorf("parse enrichment response: %w (raw: %.200s)", err, results)
	}

	for _, item := range items {
		for _, t := range item.Themes {
			if name := t.text(); name != "" {
				if err := repo.UpsertTheme(ctx, int(item.TMDBID), name, t.Score, mediaType); err != nil {
					log.Warn().Err(err).Str("theme", name).Int64("tmdb_id", item.TMDBID).Msg("upsert theme failed")
				}
			}
		}
		for _, m := range item.Moods {
			if name := m.text(); name != "" {
				if err := repo.UpsertMood(ctx, int(item.TMDBID), name, m.Score, mediaType); err != nil {
					log.Warn().Err(err).Str("mood", name).Int64("tmdb_id", item.TMDBID).Msg("upsert mood failed")
				}
			}
		}
		if item.CinovaSynopsis != "" || item.CinovaEditorial != "" {
			if mediaType == "movie" {
				if err := repo.UpdateMovieEnrichmentText(ctx, item.TMDBID, item.CinovaSynopsis, item.CinovaEditorial); err != nil {
					log.Warn().Err(err).Int64("tmdb_id", item.TMDBID).Msg("update movie enrichment text failed")
				}
			} else {
				if err := repo.UpdateTVShowEnrichmentText(ctx, item.TMDBID, item.CinovaSynopsis, item.CinovaEditorial); err != nil {
					log.Warn().Err(err).Int64("tmdb_id", item.TMDBID).Msg("update tvshow enrichment text failed")
				}
			}
		}
	}

	log.Debug().Int("batch_size", len(inputs)).Int("enriched", len(items)).Msg("batch enriched")
	return nil
}

// parseEnrichResponse handles both {"results":[...]} and bare [...] array responses,
// and also strips leading/trailing non-JSON text the model sometimes adds.
func parseEnrichResponse(raw string) ([]enrichItem, error) {
	// Trim whitespace
	raw = strings.TrimSpace(raw)

	// Try {"results":[...]} first
	var resp enrichResponse
	if err := json.Unmarshal([]byte(raw), &resp); err == nil {
		return resp.Results, nil
	}

	// Try bare array [...]
	var items []enrichItem
	if err := json.Unmarshal([]byte(raw), &items); err == nil {
		return items, nil
	}

	// Try extracting the outermost JSON object
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start >= 0 && end > start {
		var resp2 enrichResponse
		if err := json.Unmarshal([]byte(raw[start:end+1]), &resp2); err == nil {
			return resp2.Results, nil
		}
	}

	// Try extracting a bare array
	aStart := strings.Index(raw, "[")
	aEnd := strings.LastIndex(raw, "]")
	if aStart >= 0 && aEnd > aStart {
		var items2 []enrichItem
		if err := json.Unmarshal([]byte(raw[aStart:aEnd+1]), &items2); err == nil {
			return items2, nil
		}
	}

	return nil, fmt.Errorf("no valid JSON found in response")
}

// callChatCompletions sends a chat completions request to Axon and returns the text content.
func (c *Client) callChatCompletions(ctx context.Context, systemPrompt, userContent string, maxTokens int) (string, error) {
	reqBody := chatRequest{
		Model: c.model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userContent},
		},
		MaxTokens:      maxTokens,
		ResponseFormat: &respFormat{Type: "json_object"},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal chat request: %w", err)
	}

	endpoint, err := url.JoinPath(c.axonURL, "/v1/chat/completions")
	if err != nil {
		return "", fmt.Errorf("build axon url: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("build axon request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.axonAPIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.axonAPIKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("axon http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return "", fmt.Errorf("axon returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}

	var chatResp chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", fmt.Errorf("decode chat response: %w", err)
	}
	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("empty choices in chat response")
	}
	return chatResp.Choices[0].Message.Content, nil
}
