package wikidata

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/foundrylab-app/cinova/backend/internal/models"
)

const (
	sparqlEndpoint = "https://query.wikidata.org/sparql"
	userAgent      = "Cinova/1.0 (https://cinova.app; cinova@foundrylab.app) Go/1.23"
)

// Pre-compiled regexes for stripWikiMarkup — compiled once at init, not per call.
var (
	reRefBlock    = regexp.MustCompile(`(?s)<ref[^>]*>.*?</ref>`)
	reRefSelf     = regexp.MustCompile(`<ref[^/]*/>`)
	reTemplate    = regexp.MustCompile(`\{\{[^{}]*\}\}`)
	reFileLink    = regexp.MustCompile(`\[\[(?:File|Image|Datei|Fichier):[^\]]*\]\]`)
	reLinkText    = regexp.MustCompile(`\[\[[^\]|]*\|([^\]]*)\]\]`)
	reLinkPlain   = regexp.MustCompile(`\[\[([^\]]*)\]\]`)
	reHTMLTag     = regexp.MustCompile(`<[^>]+>`)
	reMultiNL     = regexp.MustCompile(`\n{3,}`)
)

// Client is a SPARQL client for the Wikidata query service.
type Client struct {
	httpClient *http.Client
}

// NewClient creates a Wikidata SPARQL client.
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Influence represents a directed influence relationship between two people.
type Influence struct {
	WikidataID            string // the influenced person
	InfluencedByWikidataID string // their influencer
}

// WikidataPerson represents a person node from Wikidata.
type WikidataPerson struct {
	WikidataID string
	IMDbID     string
	Name       string
	BirthDate  string
	Occupation string
}

// GetDirectorInfluences returns pairs of {director, influencer} where the
// influencer is a film director known to have influenced the other.
func (c *Client) GetDirectorInfluences(ctx context.Context) ([]Influence, error) {
	sparql := `
SELECT ?director ?influencer WHERE {
  ?director wdt:P31 wd:Q5 ;
            wdt:P106 wd:Q2526255 ;
            wdt:P737 ?influencer .
  ?influencer wdt:P106 wd:Q2526255 .
}
LIMIT 10000
`
	rows, err := c.query(ctx, sparql)
	if err != nil {
		return nil, fmt.Errorf("GetDirectorInfluences: %w", err)
	}

	influences := make([]Influence, 0, len(rows))
	for _, row := range rows {
		dir := extractQID(row["director"])
		inf := extractQID(row["influencer"])
		if dir == "" || inf == "" {
			continue
		}
		influences = append(influences, Influence{
			WikidataID:             dir,
			InfluencedByWikidataID: inf,
		})
	}
	return influences, nil
}

// GetPersonByIMDbID looks up a Wikidata person by their IMDb person ID (nm*).
func (c *Client) GetPersonByIMDbID(ctx context.Context, imdbID string) (*WikidataPerson, error) {
	sparql := fmt.Sprintf(`
SELECT ?person ?personLabel ?birthDate ?occupationLabel WHERE {
  ?person wdt:P345 "%s" .
  OPTIONAL { ?person wdt:P569 ?birthDate . }
  OPTIONAL { ?person wdt:P106 ?occupation . }
  SERVICE wikibase:label { bd:serviceParam wikibase:language "en" . }
}
LIMIT 1
`, imdbID)

	rows, err := c.query(ctx, sparql)
	if err != nil {
		return nil, fmt.Errorf("GetPersonByIMDbID(%s): %w", imdbID, err)
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("person with IMDb ID %s not found in Wikidata", imdbID)
	}

	row := rows[0]
	person := &WikidataPerson{
		WikidataID: extractQID(row["person"]),
		IMDbID:     imdbID,
		Name:       extractLiteral(row["personLabel"]),
		Occupation: extractLiteral(row["occupationLabel"]),
	}

	if bd, ok := row["birthDate"]; ok {
		person.BirthDate = extractLiteral(bd)
	}

	return person, nil
}

// ---- SPARQL execution ----

// sparqlResponse is the W3C SPARQL JSON response format.
type sparqlResponse struct {
	Results struct {
		Bindings []map[string]sparqlValue `json:"bindings"`
	} `json:"results"`
}

type sparqlValue struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// query executes a SPARQL SELECT query and returns the bindings.
func (c *Client) query(ctx context.Context, sparql string) ([]map[string]sparqlValue, error) {
	params := url.Values{}
	params.Set("query", sparql)
	params.Set("format", "json")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		sparqlEndpoint+"?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("build sparql request: %w", err)
	}
	req.Header.Set("Accept", "application/sparql-results+json")
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sparql http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("sparql returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var result sparqlResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode sparql response: %w", err)
	}

	return result.Results.Bindings, nil
}

// ---- Helpers ----

// extractQID extracts the Wikidata QID (e.g. "Q12345") from a URI value.
// Input: {"type":"uri","value":"http://www.wikidata.org/entity/Q12345"}
func extractQID(v sparqlValue) string {
	if v.Type != "uri" {
		return ""
	}
	parts := strings.Split(v.Value, "/entity/")
	if len(parts) != 2 {
		return ""
	}
	return parts[1]
}

// extractLiteral returns the string value for a literal or uri binding.
func extractLiteral(v sparqlValue) string {
	return v.Value
}

// InfluenceWithLabels represents a director-influencer pair with human-readable names.
type InfluenceWithLabels struct {
	DirectorName  string
	InfluencerName string
}

// ---- Per-movie enrichment ----

// MovieEnrichment holds Wikidata-sourced enrichment data for a single film.
type MovieEnrichment struct {
	WikidataID string
	RTScore    float64          // Rotten Tomatoes percentage / 100, or -1 if absent
	MetaScore  float64          // Metacritic score / 100, or -1 if absent
	IMDbScore  float64          // IMDb rating / 10, or -1 if absent
	Awards     []models.Award
}

// GetMovieEnrichment fetches critic scores and award data for a film by Wikidata QID.
// Runs two separate SPARQL queries to avoid timeout.
func (c *Client) GetMovieEnrichment(ctx context.Context, wikidataID string) (*MovieEnrichment, error) {
	result := &MovieEnrichment{
		WikidataID: wikidataID,
		RTScore:    -1,
		MetaScore:  -1,
		IMDbScore:  -1,
	}

	// Query 1: review scores (P444 = review score, P447 = reviewed by)
	scoreSPARQL := fmt.Sprintf(`
SELECT ?score ?reviewer ?reviewerLabel WHERE {
  wd:%s p:P444 ?stmt .
  ?stmt ps:P444 ?score .
  OPTIONAL { ?stmt pq:P447 ?reviewer . }
  SERVICE wikibase:label { bd:serviceParam wikibase:language "en" . }
}
LIMIT 20
`, wikidataID)
	scoreRows, err := c.query(ctx, scoreSPARQL)
	if err == nil {
		for _, row := range scoreRows {
			scoreStr := extractLiteral(row["score"])
			reviewer := strings.ToLower(extractLiteral(row["reviewerLabel"]))
			if rt, ok := parseRTScore(scoreStr); ok && strings.Contains(reviewer, "rotten tomatoes") {
				result.RTScore = rt
			} else if meta, ok := parseMetacriticScore(scoreStr); ok && strings.Contains(reviewer, "metacritic") {
				result.MetaScore = meta
			} else if imdb, ok := parseIMDbScore(scoreStr); ok && strings.Contains(reviewer, "imdb") {
				result.IMDbScore = imdb
			}
		}
	}

	// Query 2: awards won (P166)
	awardSPARQL := fmt.Sprintf(`
SELECT ?award ?awardLabel ?year ?recipient ?recipientLabel WHERE {
  wd:%s p:P166 ?stmt .
  ?stmt ps:P166 ?award .
  OPTIONAL { ?stmt pq:P585 ?year . }
  OPTIONAL { ?stmt pq:P1346 ?recipient . }
  SERVICE wikibase:label { bd:serviceParam wikibase:language "en" . }
}
LIMIT 50
`, wikidataID)
	awardRows, err := c.query(ctx, awardSPARQL)
	if err == nil {
		for _, row := range awardRows {
			qid := extractQID(row["award"])
			name := extractLiteral(row["awardLabel"])
			if qid == "" || name == "" {
				continue
			}
			a := models.Award{
				WikidataID:   qid,
				AwardName:    name,
				IsNomination: false,
			}
			if yr, ok := row["year"]; ok {
				a.Year = parseYear(extractLiteral(yr))
			}
			if rec, ok := row["recipientLabel"]; ok {
				a.RecipientName = extractLiteral(rec)
			}
			result.Awards = append(result.Awards, a)
		}
	}

	// Query 3: nominations (P1411)
	nomSPARQL := fmt.Sprintf(`
SELECT ?award ?awardLabel ?year WHERE {
  wd:%s p:P1411 ?stmt .
  ?stmt ps:P1411 ?award .
  OPTIONAL { ?stmt pq:P585 ?year . }
  SERVICE wikibase:label { bd:serviceParam wikibase:language "en" . }
}
LIMIT 50
`, wikidataID)
	nomRows, err := c.query(ctx, nomSPARQL)
	if err == nil {
		for _, row := range nomRows {
			qid := extractQID(row["award"])
			name := extractLiteral(row["awardLabel"])
			if qid == "" || name == "" {
				continue
			}
			a := models.Award{
				WikidataID:   qid,
				AwardName:    name,
				IsNomination: true,
			}
			if yr, ok := row["year"]; ok {
				a.Year = parseYear(extractLiteral(yr))
			}
			result.Awards = append(result.Awards, a)
		}
	}

	return result, nil
}

// GetWikipediaPlot fetches the Plot section from Wikipedia for a film by Wikidata QID.
// Returns the raw section text (may contain light wiki markup — Sonnet handles it).
// Returns empty string if no Wikipedia article or Plot section is found.
func (c *Client) GetWikipediaPlot(ctx context.Context, wikidataID string) (string, error) {
	title, lang, err := c.getWikipediaSitelink(ctx, wikidataID)
	if err != nil || title == "" {
		return "", nil
	}

	sectionIdx, err := c.findPlotSectionIndex(ctx, title, lang)
	if err != nil || sectionIdx < 0 {
		// Fallback: use lead section (intro paragraph)
		sectionIdx = 0
	}

	text, err := c.getWikipediaSection(ctx, title, lang, sectionIdx)
	if err != nil {
		return "", nil
	}
	return stripWikiMarkup(text), nil
}

// ---- Wikipedia helpers ----

type wikiSitelinksResponse struct {
	Entities map[string]struct {
		Sitelinks map[string]struct {
			Site  string `json:"site"`
			Title string `json:"title"`
		} `json:"sitelinks"`
	} `json:"entities"`
}

// getWikipediaSitelink returns the best Wikipedia article title and language code for a QID.
// Prefers English; falls back to any available sitelink.
func (c *Client) getWikipediaSitelink(ctx context.Context, wikidataID string) (title, lang string, err error) {
	apiURL := "https://www.wikidata.org/w/api.php"
	params := url.Values{}
	params.Set("action", "wbgetentities")
	params.Set("ids", wikidataID)
	params.Set("props", "sitelinks")
	params.Set("format", "json")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL+"?"+params.Encode(), nil)
	if err != nil {
		return "", "", fmt.Errorf("build sitelinks request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("sitelinks http: %w", err)
	}
	defer resp.Body.Close()

	var result wikiSitelinksResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", fmt.Errorf("decode sitelinks: %w", err)
	}

	entity, ok := result.Entities[wikidataID]
	if !ok {
		return "", "", nil
	}

	// Prefer English
	if sl, ok := entity.Sitelinks["enwiki"]; ok {
		return sl.Title, "en", nil
	}
	// Fallback to first available Wikipedia sitelink
	for site, sl := range entity.Sitelinks {
		if strings.HasSuffix(site, "wiki") && !strings.Contains(site, "wikidata") {
			langCode := strings.TrimSuffix(site, "wiki")
			return sl.Title, langCode, nil
		}
	}
	return "", "", nil
}

type wikiSectionsResponse struct {
	Parse struct {
		Sections []struct {
			Index string `json:"index"`
			Line  string `json:"line"`
		} `json:"sections"`
	} `json:"parse"`
}

// findPlotSectionIndex returns the numeric section index for the Plot/Synopsis section.
// Returns -1 if not found.
func (c *Client) findPlotSectionIndex(ctx context.Context, title, lang string) (int, error) {
	apiURL := fmt.Sprintf("https://%s.wikipedia.org/w/api.php", lang)
	params := url.Values{}
	params.Set("action", "parse")
	params.Set("page", title)
	params.Set("prop", "sections")
	params.Set("format", "json")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL+"?"+params.Encode(), nil)
	if err != nil {
		return -1, err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return -1, err
	}
	defer resp.Body.Close()

	var result wikiSectionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return -1, err
	}

	for _, s := range result.Parse.Sections {
		lower := strings.ToLower(s.Line)
		if lower == "plot" || lower == "synopsis" || lower == "story" ||
			strings.HasPrefix(lower, "plot") || strings.HasPrefix(lower, "synopsis") {
			idx, err := strconv.Atoi(s.Index)
			if err == nil {
				return idx, nil
			}
		}
	}
	return -1, nil
}

type wikiTextResponse struct {
	Parse struct {
		Wikitext struct {
			Content string `json:"*"`
		} `json:"wikitext"`
	} `json:"parse"`
}

// getWikipediaSection fetches the wikitext for a specific section of a Wikipedia article.
func (c *Client) getWikipediaSection(ctx context.Context, title, lang string, sectionIdx int) (string, error) {
	apiURL := fmt.Sprintf("https://%s.wikipedia.org/w/api.php", lang)
	params := url.Values{}
	params.Set("action", "parse")
	params.Set("page", title)
	params.Set("prop", "wikitext")
	params.Set("section", strconv.Itoa(sectionIdx))
	params.Set("format", "json")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL+"?"+params.Encode(), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result wikiTextResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	return result.Parse.Wikitext.Content, nil
}

// stripWikiMarkup removes the noisiest wiki markup so the text is cleaner for Sonnet.
// Removes {{templates}}, [[File:...]], ref tags, and reduces [[link|text]] → text.
func stripWikiMarkup(s string) string {
	s = reRefBlock.ReplaceAllString(s, "")
	s = reRefSelf.ReplaceAllString(s, "")
	// Remove {{templates}} iteratively for nesting, with infinite-loop guard.
	// If a pass makes no progress (malformed markup), stop to avoid spinning.
	for i := 0; i < 20 && strings.Contains(s, "{{"); i++ {
		next := reTemplate.ReplaceAllString(s, "")
		if next == s {
			break // no innermost template matched; avoid infinite loop
		}
		s = next
	}
	s = reFileLink.ReplaceAllString(s, "")
	s = reLinkText.ReplaceAllString(s, "$1")
	s = reLinkPlain.ReplaceAllString(s, "$1")
	s = reHTMLTag.ReplaceAllString(s, "")
	s = reMultiNL.ReplaceAllString(s, "\n\n")
	return strings.TrimSpace(s)
}

// ---- score parsing helpers ----

// parseRTScore parses "83%" → 0.83
func parseRTScore(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	if strings.HasSuffix(s, "%") {
		v, err := strconv.ParseFloat(strings.TrimSuffix(s, "%"), 64)
		if err == nil && v >= 0 && v <= 100 {
			return v / 100.0, true
		}
	}
	return 0, false
}

// parseMetacriticScore parses "73/100" → 0.73
func parseMetacriticScore(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	parts := strings.Split(s, "/")
	if len(parts) == 2 && strings.TrimSpace(parts[1]) == "100" {
		v, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
		if err == nil && v >= 0 && v <= 100 {
			return v / 100.0, true
		}
	}
	return 0, false
}

// parseIMDbScore parses "8.7/10" → 0.87
func parseIMDbScore(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	parts := strings.Split(s, "/")
	if len(parts) == 2 && strings.TrimSpace(parts[1]) == "10" {
		v, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
		if err == nil && v >= 0 && v <= 10 {
			return v / 10.0, true
		}
	}
	return 0, false
}

// parseYear extracts a 4-digit year from a Wikidata datetime string like "2001-01-01T00:00:00Z".
func parseYear(s string) int {
	if len(s) >= 4 {
		y, err := strconv.Atoi(s[:4])
		if err == nil && y > 1880 && y < 2100 {
			return y
		}
	}
	return 0
}

// GetDirectorInfluencesWithLabels returns director influence pairs with human-readable names.
// Fetches up to 10k pairs. Used during full ingestion to wire INFLUENCED_BY edges.
func (c *Client) GetDirectorInfluencesWithLabels(ctx context.Context) ([]InfluenceWithLabels, error) {
	sparql := `
SELECT ?directorLabel ?influencerLabel WHERE {
  ?director wdt:P31 wd:Q5 ;
            wdt:P106 wd:Q2526255 ;
            wdt:P737 ?influencer .
  ?influencer wdt:P106 wd:Q2526255 .
  SERVICE wikibase:label { bd:serviceParam wikibase:language "en" . }
}
LIMIT 10000
`
	rows, err := c.query(ctx, sparql)
	if err != nil {
		return nil, fmt.Errorf("GetDirectorInfluencesWithLabels: %w", err)
	}

	results := make([]InfluenceWithLabels, 0, len(rows))
	for _, row := range rows {
		dirName := extractLiteral(row["directorLabel"])
		infName := extractLiteral(row["influencerLabel"])
		if dirName == "" || infName == "" {
			continue
		}
		results = append(results, InfluenceWithLabels{
			DirectorName:   dirName,
			InfluencerName: infName,
		})
	}
	return results, nil
}
