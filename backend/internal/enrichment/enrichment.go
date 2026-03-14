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
	axonTimeout    = 30 * time.Second
	maxBatchSize   = 20
)

// Client calls the Axon AI service to extract themes and moods from plot summaries.
type Client struct {
	axonURL    string
	axonAPIKey string
	httpClient *http.Client
}

// NewClient creates an enrichment Client.
func NewClient(axonURL, axonAPIKey string) *Client {
	return &Client{
		axonURL:    axonURL,
		axonAPIKey: axonAPIKey,
		httpClient: &http.Client{Timeout: axonTimeout},
	}
}

// ---- Axon request/response structs ----

type axonThemeRequest struct {
	Plot string `json:"plot"`
}

type axonThemeResponse struct {
	Themes []struct {
		Name  string  `json:"name"`
		Score float64 `json:"score"`
	} `json:"themes"`
	Moods []struct {
		Name  string  `json:"name"`
		Score float64 `json:"score"`
	} `json:"moods"`
}

// ExtractThemes calls Axon to extract thematic tags from a plot summary.
func (c *Client) ExtractThemes(ctx context.Context, plotSummary string) ([]string, error) {
	resp, err := c.callAxon(ctx, "/extract/themes", axonThemeRequest{Plot: plotSummary})
	if err != nil {
		return nil, fmt.Errorf("ExtractThemes: %w", err)
	}

	themes := make([]string, 0, len(resp.Themes))
	for _, t := range resp.Themes {
		themes = append(themes, t.Name)
	}
	return themes, nil
}

// ExtractMood calls Axon to extract emotional mood tags from a plot summary.
func (c *Client) ExtractMood(ctx context.Context, plotSummary string) ([]string, error) {
	resp, err := c.callAxon(ctx, "/extract/themes", axonThemeRequest{Plot: plotSummary})
	if err != nil {
		return nil, fmt.Errorf("ExtractMood: %w", err)
	}

	moods := make([]string, 0, len(resp.Moods))
	for _, m := range resp.Moods {
		moods = append(moods, m.Name)
	}
	return moods, nil
}

// ProcessMovieBatch enriches a batch of movies with AI-extracted themes and moods,
// then upserts the Theme and Mood nodes and relationships into Neo4j.
func (c *Client) ProcessMovieBatch(ctx context.Context, movies []models.Movie, repo *graph.MovieRepository) error {
	// Process in sub-batches to respect Axon rate limits
	for i := 0; i < len(movies); i += maxBatchSize {
		end := i + maxBatchSize
		if end > len(movies) {
			end = len(movies)
		}
		sub := movies[i:end]

		for _, movie := range sub {
			if movie.Overview == "" {
				continue
			}

			resp, err := c.callAxon(ctx, "/extract/themes", axonThemeRequest{Plot: movie.Overview})
			if err != nil {
				log.Warn().Err(err).Int64("tmdb_id", movie.TMDBID).Msg("axon enrichment failed, skipping")
				continue
			}

			// Upsert Theme nodes and HAS_THEME relationships
			for _, t := range resp.Themes {
				if err := repo.UpsertTheme(ctx, int(movie.TMDBID), t.Name, t.Score, "movie"); err != nil {
					log.Warn().Err(err).Str("theme", t.Name).Int64("tmdb_id", movie.TMDBID).Msg("upsert theme failed")
				}
			}

			// Upsert Mood nodes and HAS_MOOD relationships
			for _, m := range resp.Moods {
				if err := repo.UpsertMood(ctx, int(movie.TMDBID), m.Name, m.Score, "movie"); err != nil {
					log.Warn().Err(err).Str("mood", m.Name).Int64("tmdb_id", movie.TMDBID).Msg("upsert mood failed")
				}
			}

			// Brief pause to respect Axon rate limits
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(50 * time.Millisecond):
			}
		}
	}
	return nil
}

// callAxon sends a POST request to the given Axon endpoint and returns the parsed response.
func (c *Client) callAxon(ctx context.Context, path string, payload interface{}) (*axonThemeResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal axon payload: %w", err)
	}

	endpoint, err := url.JoinPath(c.axonURL, path)
	if err != nil {
		return nil, fmt.Errorf("build axon url: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build axon request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.axonAPIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.axonAPIKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("axon http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("axon returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}

	var result axonThemeResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode axon response: %w", err)
	}
	return &result, nil
}
