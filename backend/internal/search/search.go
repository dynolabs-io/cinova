package search

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

	"github.com/foundrylab-app/cinova/backend/internal/auth"
	"github.com/foundrylab-app/cinova/backend/internal/config"
	"github.com/foundrylab-app/cinova/backend/internal/graph"
	"github.com/foundrylab-app/cinova/backend/internal/models"
	"github.com/foundrylab-app/cinova/backend/internal/store"
)

const (
	defaultLimit = 20
	axonTimeout  = 60 * time.Second

	// Sonnet 4.6 for NL search — quality matters for user-facing queries
	searchModel = "claude-sonnet-4-6"

	nl2cypherSystemPrompt = `You are a Cypher query assistant for a movie and TV show graph database.

Schema:
- Nodes: Movie (title, overview, vote_average, cinova_score, release_year, runtime), TVShow (title, overview, vote_average, cinova_score, first_air_year), Genre (name), Theme (name), Mood (name), Provider (name, provider_name)
- Relationships: (Movie|TVShow)-[:IN_GENRE]->(Genre), (Movie|TVShow)-[:HAS_THEME]->(Theme), (Movie|TVShow)-[:HAS_MOOD]->(Mood), (Movie|TVShow)-[:AVAILABLE_ON {country}]->(Provider)

The query variable is 'n' (for Movie or TVShow nodes).
Provider filtering uses: EXISTS { MATCH (n)-[:AVAILABLE_ON {country: $country}]->(p:Provider) WHERE toLower(p.provider_name) CONTAINS '<name>' }

Return a JSON object with:
- "where_clause": a valid Cypher WHERE fragment (no MATCH/RETURN, just conditions on 'n' and optional EXISTS subqueries)
- "explanation": a 1-sentence human-readable description of what the query finds

If the query is just a title search, use: toLower(n.title) CONTAINS toLower('<title>')
For themes/moods, use EXISTS { MATCH (n)-[:HAS_THEME]->(t:Theme) WHERE toLower(t.name) CONTAINS '<theme>' }

Return ONLY valid JSON, no markdown, no code blocks.`
)

// Handler handles natural-language search requests.
type Handler struct {
	neo    *graph.Driver
	redis  *store.RedisStore
	cfg    *config.Config
	client *http.Client
}

// NewHandler creates a new search Handler.
func NewHandler(neo *graph.Driver, redis *store.RedisStore, cfg *config.Config) *Handler {
	return &Handler{
		neo:   neo,
		redis: redis,
		cfg:   cfg,
		client: &http.Client{
			Timeout: axonTimeout,
		},
	}
}

// nl2cypherResponse is the structured response from Axon's NL→Cypher translation.
type nl2cypherResponse struct {
	WhereClause string `json:"where_clause"`
	Explanation string `json:"explanation,omitempty"`
}

// chatMessage is a single message in an OpenAI-compatible chat request.
type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// chatRequest is the OpenAI-compatible /v1/chat/completions request body.
type chatRequest struct {
	Model     string        `json:"model"`
	Messages  []chatMessage `json:"messages"`
	MaxTokens int           `json:"max_tokens"`
}

// chatResponse is the OpenAI-compatible /v1/chat/completions response.
type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// SearchHandler handles GET /api/v1/search?q=&country=
func (h *Handler) SearchHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	country := r.URL.Query().Get("country")
	if country == "" {
		country = "US"
	}

	if q == "" {
		writeError(w, "invalid_request", "query parameter 'q' is required", http.StatusBadRequest)
		return
	}

	sessionID := auth.SessionIDFromCtx(r.Context())

	// Check rate limit
	if sessionID != "" {
		allowed, remaining, err := h.redis.CheckRateLimit(r.Context(), sessionID)
		if err != nil {
			log.Warn().Err(err).Msg("rate limit check failed, allowing request")
		} else if !allowed {
			w.Header().Set("X-RateLimit-Remaining", "0")
			writeError(w, "rate_limited", "too many requests, please slow down", http.StatusTooManyRequests)
			return
		} else {
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
		}
	}

	// Check Redis cache
	cached, err := h.redis.GetCachedSearchResults(r.Context(), q, country)
	if err != nil {
		log.Warn().Err(err).Msg("search cache read failed")
	}
	if cached != nil {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cache", "HIT")
		w.WriteHeader(http.StatusOK)
		w.Write(cached)
		return
	}

	// Translate natural language query to Cypher WHERE clause via Axon
	whereClause, explanation, err := h.translateQuery(r.Context(), q, country)
	if err != nil {
		log.Error().Err(err).Str("query", q).Msg("axon translation failed")
		// Fall back to simple title search
		whereClause = fmt.Sprintf("toLower(n.title) CONTAINS toLower('%s')", escapeStr(q))
		explanation = ""
	} else {
		log.Info().Str("query", q).Str("where_clause", whereClause).Str("explanation", explanation).Msg("axon nl2cypher")
	}

	results, err := h.executeSearch(r.Context(), whereClause, country, defaultLimit)
	if err != nil {
		log.Error().Err(err).Str("query", q).Msg("search query failed")
		writeError(w, "internal_error", "search failed", http.StatusInternalServerError)
		return
	}

	// Attach match reason from Axon's explanation
	if explanation != "" {
		for i := range results {
			if results[i].MatchReason == "" {
				results[i].MatchReason = explanation
			}
		}
	}

	response := map[string]interface{}{
		"results": results,
		"query":   q,
		"country": country,
		"total":   len(results),
	}

	// Cache the results
	if err := h.redis.CacheSearchResults(r.Context(), q, country, response); err != nil {
		log.Warn().Err(err).Msg("failed to cache search results")
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Cache", "MISS")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Error().Err(err).Msg("write search response")
	}
}

// translateQuery calls Axon's /v1/chat/completions to convert a natural language
// query into a Cypher WHERE clause.
func (h *Handler) translateQuery(ctx context.Context, query, country string) (whereClause, explanation string, err error) {
	userContent := fmt.Sprintf("Query: %q\nCountry: %s\n\nReturn a JSON object with where_clause and explanation.", query, country)

	reqBody := chatRequest{
		Model: searchModel,
		Messages: []chatMessage{
			{Role: "system", Content: nl2cypherSystemPrompt},
			{Role: "user", Content: userContent},
		},
		MaxTokens: 512,
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return "", "", fmt.Errorf("marshal chat request: %w", err)
	}

	endpoint, err := url.JoinPath(h.cfg.AxonURL, "/v1/chat/completions")
	if err != nil {
		return "", "", fmt.Errorf("build axon url: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return "", "", fmt.Errorf("build axon request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if h.cfg.AxonAPIKey != "" {
		req.Header.Set("Authorization", "Bearer "+h.cfg.AxonAPIKey)
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("axon http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return "", "", fmt.Errorf("axon returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}

	var chatResp chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", "", fmt.Errorf("decode chat response: %w", err)
	}
	if len(chatResp.Choices) == 0 {
		return "", "", fmt.Errorf("empty choices in chat response")
	}

	content := chatResp.Choices[0].Message.Content

	// Strip markdown code blocks if model added them
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var nl2cResp nl2cypherResponse
	if err := json.Unmarshal([]byte(content), &nl2cResp); err != nil {
		return "", "", fmt.Errorf("parse nl2cypher response: %w (raw: %.200s)", err, content)
	}
	if nl2cResp.WhereClause == "" {
		return "", "", fmt.Errorf("empty where_clause in response")
	}

	return nl2cResp.WhereClause, nl2cResp.Explanation, nil
}

// executeSearch runs the translated Cypher WHERE clause against Neo4j and
// returns ranked SearchResult items.
func (h *Handler) executeSearch(ctx context.Context, whereClause, country string, limit int) ([]models.SearchResult, error) {
	cypher := fmt.Sprintf(`
		MATCH (n)
		WHERE (n:Movie OR n:TVShow) AND (%s)
		OPTIONAL MATCH (n)-[:IN_GENRE]->(g:Genre)
		OPTIONAL MATCH (n)-[:AVAILABLE_ON {country: $country}]->(prov:Provider)
		WITH n,
		     collect(DISTINCT {id: g.id, name: g.name})              AS genres,
		     collect(DISTINCT {provider_id: prov.provider_id,
		                        provider_name: prov.provider_name,
		                        logo_path: prov.logo_path,
		                        type: prov.type})                     AS providers
		RETURN n, genres, providers
		ORDER BY n.cinova_score DESC, n.popularity DESC
		LIMIT $limit
	`, whereClause)

	records, err := h.neo.RunQuery(ctx, cypher, map[string]interface{}{
		"country": country,
		"limit":   limit,
	})
	if err != nil {
		return nil, fmt.Errorf("executeSearch: %w", err)
	}

	results := make([]models.SearchResult, 0, len(records))
	for _, rec := range records {
		node, ok := rec.Get("n")
		if !ok {
			continue
		}

		// Determine if Movie or TVShow from node labels
		type labeller interface {
			Labels() []string
			Props() map[string]interface{}
		}
		ln, isLabeller := node.(labeller)
		if !isLabeller {
			continue
		}

		props := ln.Props()
		mediaType := "movie"
		title := ""
		for _, label := range ln.Labels() {
			if label == "TVShow" {
				mediaType = "tv"
				break
			}
		}

		if mediaType == "tv" {
			title = strVal(props["name"])
		} else {
			title = strVal(props["title"])
		}

		releaseYear := ""
		if mediaType == "movie" {
			date := strVal(props["release_date"])
			if len(date) >= 4 {
				releaseYear = date[:4]
			}
		} else {
			date := strVal(props["first_air_date"])
			if len(date) >= 4 {
				releaseYear = date[:4]
			}
		}

		sr := models.SearchResult{
			TMDBID:      int64Val(props["tmdb_id"]),
			MediaType:   mediaType,
			Title:       title,
			PosterPath:  strVal(props["poster_path"]),
			ReleaseYear: releaseYear,
			VoteAverage: float64Val(props["vote_average"]),
			CinovaScore: float64Val(props["cinova_score"]),
			Overview:    strVal(props["overview"]),
		}

		if v, ok := rec.Get("genres"); ok {
			sr.Genres = toGenres(v)
		}
		if v, ok := rec.Get("providers"); ok {
			sr.Providers = toProviders(v)
		}

		results = append(results, sr)
	}

	return results, nil
}

// ---- Helpers ----

// escapeStr escapes single quotes in a string for safe inline Cypher embedding.
func escapeStr(s string) string {
	result := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '\'' {
			result = append(result, '\'', '\'')
		} else {
			result = append(result, s[i])
		}
	}
	return string(result)
}

func writeError(w http.ResponseWriter, errCode, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(models.ErrorResponse{
		Error:   errCode,
		Message: message,
		Code:    status,
	})
}

// toGenres, toProviders, strVal, int64Val, float64Val re-use the same logic as graph package
// but are local to avoid cross-package import cycles.

func strVal(v interface{}) string {
	if v == nil {
		return ""
	}
	s, _ := v.(string)
	return s
}

func int64Val(v interface{}) int64 {
	if v == nil {
		return 0
	}
	switch t := v.(type) {
	case int64:
		return t
	case float64:
		return int64(t)
	}
	return 0
}

func float64Val(v interface{}) float64 {
	if v == nil {
		return 0
	}
	switch t := v.(type) {
	case float64:
		return t
	case int64:
		return float64(t)
	}
	return 0
}

func toGenres(v interface{}) []models.Genre {
	list, _ := v.([]interface{})
	genres := make([]models.Genre, 0, len(list))
	for _, item := range list {
		m, _ := item.(map[string]interface{})
		if m == nil || m["id"] == nil {
			continue
		}
		genres = append(genres, models.Genre{
			ID:   int64Val(m["id"]),
			Name: strVal(m["name"]),
		})
	}
	return genres
}

func toProviders(v interface{}) []models.Provider {
	list, _ := v.([]interface{})
	providers := make([]models.Provider, 0, len(list))
	for _, item := range list {
		m, _ := item.(map[string]interface{})
		if m == nil || m["provider_id"] == nil {
			continue
		}
		providers = append(providers, models.Provider{
			ProviderID:   int64Val(m["provider_id"]),
			ProviderName: strVal(m["provider_name"]),
			LogoPath:     strVal(m["logo_path"]),
			Type:         strVal(m["type"]),
		})
	}
	return providers
}
