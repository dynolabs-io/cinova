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
	axonBatchTimeout = 120 * time.Second
	batchSize        = 20 // smaller batch — Sonnet 4.6 generates more tokens per item

	// Sonnet 4.6 — superior semantic understanding for enrichment quality
	enrichmentModel = "claude-sonnet-4-6"

	enrichmentSystemPrompt = `You are a film/TV metadata enrichment assistant.
For each item in the input array, produce:
- themes: 2-5 broad thematic labels (e.g. "Redemption", "Identity", "Power & Corruption", "Survival")
- moods: 2-4 emotional/tonal labels (e.g. "Tense", "Melancholic", "Darkly Comic", "Uplifting")
- cinova_synopsis: a 2-sentence spoiler-free hook that would make someone want to watch it.
  Write in present tense. Do NOT reveal plot twists or the ending. Always output English regardless of input language.

Return ONLY a valid JSON object with this exact structure:
{"results": [{"tmdb_id": <number>, "themes": [{"name": <string>, "score": <0.0-1.0>}], "moods": [{"name": <string>, "score": <0.0-1.0>}], "cinova_synopsis": <string>}]}`
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

type enrichItem struct {
	TMDBID int64 `json:"tmdb_id"`
	Themes []struct {
		Name  string  `json:"name"`
		Score float64 `json:"score"`
	} `json:"themes"`
	Moods []struct {
		Name  string  `json:"name"`
		Score float64 `json:"score"`
	} `json:"moods"`
	CinovaSynopsis string `json:"cinova_synopsis"`
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
			TMDBID:   s.TMDBID,
			Title:    s.Name,
			Tagline:  s.Tagline,
			Overview: s.Overview,
			Keywords: s.Keywords,
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

	results, err := c.callChatCompletions(ctx, enrichmentSystemPrompt, userContent, 6000)
	if err != nil {
		return fmt.Errorf("chat completions: %w", err)
	}

	var enrichResp enrichResponse
	if err := json.Unmarshal([]byte(results), &enrichResp); err != nil {
		// Try to extract JSON from response if model added surrounding text
		start := strings.Index(results, "{")
		end := strings.LastIndex(results, "}")
		if start >= 0 && end > start {
			if err2 := json.Unmarshal([]byte(results[start:end+1]), &enrichResp); err2 != nil {
				return fmt.Errorf("parse enrichment response: %w (raw: %.200s)", err, results)
			}
		} else {
			return fmt.Errorf("parse enrichment response: %w (raw: %.200s)", err, results)
		}
	}

	for _, item := range enrichResp.Results {
		for _, t := range item.Themes {
			if err := repo.UpsertTheme(ctx, int(item.TMDBID), t.Name, t.Score, mediaType); err != nil {
				log.Warn().Err(err).Str("theme", t.Name).Int64("tmdb_id", item.TMDBID).Msg("upsert theme failed")
			}
		}
		for _, m := range item.Moods {
			if err := repo.UpsertMood(ctx, int(item.TMDBID), m.Name, m.Score, mediaType); err != nil {
				log.Warn().Err(err).Str("mood", m.Name).Int64("tmdb_id", item.TMDBID).Msg("upsert mood failed")
			}
		}
		if item.CinovaSynopsis != "" && mediaType == "movie" {
			if err := repo.UpdateMovieEnrichmentText(ctx, item.TMDBID, "", item.CinovaSynopsis); err != nil {
				log.Warn().Err(err).Int64("tmdb_id", item.TMDBID).Msg("update synopsis failed")
			}
		}
	}

	log.Debug().Int("batch_size", len(inputs)).Int("enriched", len(enrichResp.Results)).Msg("batch enriched")
	return nil
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
