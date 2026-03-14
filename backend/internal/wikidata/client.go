package wikidata

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	sparqlEndpoint = "https://query.wikidata.org/sparql"
	userAgent      = "Cinova/1.0 (https://cinova.app; cinova@foundrylab.app) Go/1.23"
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
