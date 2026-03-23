package graph

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j/dbtype"

	"github.com/foundrylab-app/cinova/backend/internal/models"
)

// asciiName normalises a person name to lowercase ASCII for fuzzy matching.
//
// Strategy:
//  1. NFD-decompose: é→e+combining_acute, ü→u+combining_diaeresis, ç→c+combining_cedilla, etc.
//  2. Strip all Unicode combining/mark characters (the accent/diacritic glyphs).
//  3. Handle the few characters that do NOT decompose via NFD and need manual mapping:
//     - ı (U+0131, Turkish dotless-i) → i
//     - ß (U+00DF, German sharp-s)    → ss
//     - æ (U+00E6)                    → ae
//     - ø (U+00F8)                    → o
//     - å (U+00E5) decomposes, handled by step 2
//
// This covers virtually all Latin-script diacritics (Turkish, French, German, Spanish,
// Portuguese, Scandinavian, Eastern European, etc.) without a hand-picked character list.
var asciiNameTransformer = transform.Chain(norm.NFD, transform.RemoveFunc(unicode.IsMark))

func asciiName(s string) string {
	lower := strings.ToLower(s)
	result, _, _ := transform.String(asciiNameTransformer, lower)
	// Non-decomposable characters — manual pass
	result = strings.ReplaceAll(result, "ı", "i")  // Turkish dotless-i (U+0131)
	result = strings.ReplaceAll(result, "ß", "ss") // German sharp-s
	result = strings.ReplaceAll(result, "æ", "ae") // ae ligature
	result = strings.ReplaceAll(result, "ø", "o")  // Nordic o-slash
	return result
}

// asciiNames applies asciiName to every element of a slice.
func asciiNames(ss []string) []string {
	out := make([]string, len(ss))
	for i, s := range ss {
		out[i] = asciiName(s)
	}
	return out
}

// cypherDiacriticReplacements is the same set of mappings as asciiName() but expressed
// as Cypher replace() calls. Neo4j has no built-in NFD decomposition, so we chain replace()
// for each diacritic that appears in TMDB person names.
var cypherDiacriticReplacements = [][2]string{
	// Non-decomposable via NFD — must be first so subsequent passes see plain letters
	{"ı", "i"},   // Turkish dotless-i
	{"ß", "ss"},  // German sharp-s (expands to two chars)
	{"æ", "ae"},  // ae ligature
	{"ø", "o"},   // Nordic o-slash
	// Vowels with diacritics (all decompose via NFD in Go but not in Cypher)
	{"à", "a"}, {"á", "a"}, {"â", "a"}, {"ã", "a"}, {"ä", "a"}, {"å", "a"},
	{"è", "e"}, {"é", "e"}, {"ê", "e"}, {"ë", "e"},
	{"ì", "i"}, {"í", "i"}, {"î", "i"}, {"ï", "i"},
	{"ò", "o"}, {"ó", "o"}, {"ô", "o"}, {"õ", "o"}, {"ö", "o"}, {"ő", "o"},
	{"ù", "u"}, {"ú", "u"}, {"û", "u"}, {"ü", "u"}, {"ű", "u"},
	{"ý", "y"}, {"ÿ", "y"},
	// Consonants with diacritics
	{"ñ", "n"},
	{"ğ", "g"}, {"ş", "s"}, {"ç", "c"}, // Turkish
	{"ć", "c"}, {"č", "c"}, {"š", "s"}, {"ž", "z"}, // Slavic
	{"ř", "r"}, {"ě", "e"}, {"ď", "d"}, {"ť", "t"},
	{"ľ", "l"}, {"ĺ", "l"}, {"ń", "n"}, {"ź", "z"}, {"ż", "z"},
	// Long vowels (Baltic, Maori, etc.)
	{"ā", "a"}, {"ē", "e"}, {"ī", "i"}, {"ō", "o"}, {"ū", "u"},
}

// buildCypherAsciiName wraps a Cypher expression with chained replace() calls that
// normalise diacritics to ASCII — the Cypher equivalent of asciiName().
func buildCypherAsciiName(expr string) string {
	result := fmt.Sprintf("toLower(%s)", expr)
	for _, pair := range cypherDiacriticReplacements {
		result = fmt.Sprintf("replace(%s,'%s','%s')", result, pair[0], pair[1])
	}
	return result
}

// ChatFilters holds intent extracted by the AI from the conversation.
type ChatFilters struct {
	Genres     []string
	Themes     []string
	Moods      []string
	Providers  []string
	Directors  []string // director name substrings (case-insensitive)
	Actors     []string // actor name substrings (case-insensitive)
	Language   string   // ISO 639-1 language code e.g. "tr", "fr", "ja"
	MaxRuntime int
	MinYear    int
	MaxYear    int
	MinScore   float64
	ExcludeIDs []int64
}

// UserChatContext holds personalisation data injected into the chat prompt.
type UserChatContext struct {
	TopThemes   []string // top 5 themes from rated/saved history
	SavedTitles []string // last 5 saved movie titles
}

// GetChatCandidates fetches up to 20 movies matching the given chat filters.
// Candidates are returned with genres, themes, moods, providers, and synopsis
// so Claude has rich context for writing recommendations.
func (r *MovieRepository) GetChatCandidates(ctx context.Context, f ChatFilters, country string) ([]models.MovieSummary, error) {
	params := map[string]interface{}{
		"country":     country,
		"min_score":   f.MinScore,
		"exclude_ids": f.ExcludeIDs,
	}

	// Build optional WHERE conditions
	var conds []string
	conds = append(conds, "m.cinova_score >= $min_score")
	conds = append(conds, "NOT m.tmdb_id IN $exclude_ids")

	if len(f.Genres) > 0 {
		params["genres"] = f.Genres
		conds = append(conds, "EXISTS { MATCH (m)-[:IN_GENRE]->(g:Genre) WHERE g.name IN $genres }")
	}
	if len(f.Themes) > 0 {
		params["themes"] = f.Themes
		conds = append(conds, "EXISTS { MATCH (m)-[:HAS_THEME]->(t:Theme) WHERE any(th IN $themes WHERE toLower(t.name) CONTAINS toLower(th)) }")
	}
	if len(f.Moods) > 0 {
		params["moods"] = f.Moods
		conds = append(conds, "EXISTS { MATCH (m)-[:HAS_MOOD]->(mo:Mood) WHERE any(md IN $moods WHERE toLower(mo.name) CONTAINS toLower(md)) }")
	}
	if len(f.Providers) > 0 {
		params["providers"] = f.Providers
		conds = append(conds, "EXISTS { MATCH (m)-[:AVAILABLE_ON {country: $country}]->(p:Provider) WHERE any(pv IN $providers WHERE toLower(p.provider_name) CONTAINS toLower(pv)) }")
	}
	if len(f.Directors) > 0 {
		params["directors"] = asciiNames(f.Directors)
		conds = append(conds, fmt.Sprintf(
			"EXISTS { MATCH (d:Person)-[:DIRECTED]->(m) WHERE any(dn IN $directors WHERE %s CONTAINS dn) }",
			buildCypherAsciiName("d.name"),
		))
	}
	if len(f.Actors) > 0 {
		params["actors"] = asciiNames(f.Actors)
		conds = append(conds, fmt.Sprintf(
			"EXISTS { MATCH (a:Person)-[:ACTED_IN]->(m) WHERE any(an IN $actors WHERE %s CONTAINS an) }",
			buildCypherAsciiName("a.name"),
		))
	}
	if f.Language != "" {
		params["language"] = strings.ToLower(f.Language)
		conds = append(conds, "toLower(m.original_language) = $language")
	}
	if f.MaxRuntime > 0 {
		params["max_runtime"] = f.MaxRuntime
		conds = append(conds, "m.runtime > 0 AND m.runtime <= $max_runtime")
	}
	if f.MinYear > 0 {
		params["min_year"] = fmt.Sprintf("%d-01-01", f.MinYear)
		conds = append(conds, "m.release_date >= $min_year")
	}
	if f.MaxYear > 0 {
		params["max_year"] = fmt.Sprintf("%d-12-31", f.MaxYear)
		conds = append(conds, "m.release_date <= $max_year")
	}

	cypher := fmt.Sprintf(`
		MATCH (m:Movie)
		WHERE %s
		WITH m ORDER BY m.cinova_score DESC, m.popularity DESC LIMIT 20
		OPTIONAL MATCH (m)-[:IN_GENRE]->(g:Genre)
		OPTIONAL MATCH (m)-[:HAS_THEME]->(t:Theme)
		OPTIONAL MATCH (m)-[:HAS_MOOD]->(mo:Mood)
		OPTIONAL MATCH (m)-[avail:AVAILABLE_ON {country: $country}]->(prov:Provider)
		RETURN m,
		       collect(DISTINCT {id: g.id, name: g.name})             AS genres,
		       collect(DISTINCT t.name)                                AS themes,
		       collect(DISTINCT mo.name)                               AS moods,
		       collect(DISTINCT {provider_id: prov.provider_id,
		                          provider_name: prov.provider_name,
		                          logo_path: prov.logo_path,
		                          type: avail.type})                   AS providers
	`, strings.Join(conds, "\n  AND "))

	records, err := r.driver.RunQuery(ctx, cypher, params)
	if err != nil {
		return nil, fmt.Errorf("GetChatCandidates: %w", err)
	}

	candidates := make([]models.MovieSummary, 0, len(records))
	for _, rec := range records {
		node, _ := rec.Get("m")
		m := movieNodeToModel(node)
		if m == nil {
			continue
		}
		s := models.MovieSummary{
			TMDBID:         m.TMDBID,
			MediaType:      "movie",
			Title:          m.Title,
			PosterPath:     m.PosterPath,
			CinovaScore:    m.CinovaScore,
			Overview:       m.Overview,
			CinovaSynopsis: m.CinovaSynopsis,
		}
		if rd := m.ReleaseDate; len(rd) >= 4 {
			s.ReleaseYear = rd[:4]
		}
		if v, ok := rec.Get("genres"); ok {
			s.Genres = toGenres(v)
		}
		if v, ok := rec.Get("providers"); ok {
			s.Providers = toProviders(v)
		}
		candidates = append(candidates, s)
	}
	return candidates, nil
}

// GetUserChatContext fetches lightweight personalisation signals for the chat prompt.
func (r *MovieRepository) GetUserChatContext(ctx context.Context, ownerID, ownerType string) (*UserChatContext, error) {
	uc := &UserChatContext{}

	// Top 5 themes from the user's rated/saved history
	themeCypher := fmt.Sprintf(`
		MATCH (o:%s {id: $uid})-[:RATED|:SAVED]->(m:Movie)-[:HAS_THEME]->(t:Theme)
		WITH t.name AS theme, count(*) AS cnt
		ORDER BY cnt DESC LIMIT 5
		RETURN theme
	`, ownerType)
	themeRecs, err := r.driver.RunQuery(ctx, themeCypher, map[string]interface{}{"uid": ownerID})
	if err == nil {
		for _, rec := range themeRecs {
			if v, ok := rec.Get("theme"); ok {
				uc.TopThemes = append(uc.TopThemes, strVal(v))
			}
		}
	}

	// Last 5 saved movie titles
	savedCypher := fmt.Sprintf(`
		MATCH (o:%s {id: $uid})-[s:SAVED]->(m:Movie)
		RETURN m.title AS title
		ORDER BY s.saved_at DESC LIMIT 5
	`, ownerType)
	savedRecs, err := r.driver.RunQuery(ctx, savedCypher, map[string]interface{}{"uid": ownerID})
	if err == nil {
		for _, rec := range savedRecs {
			if v, ok := rec.Get("title"); ok {
				uc.SavedTitles = append(uc.SavedTitles, strVal(v))
			}
		}
	}

	return uc, nil
}

// GetPopular returns popular movies in a country ordered by popularity, with pagination.
func (r *MovieRepository) GetPopular(ctx context.Context, country string, limit, offset int) ([]models.Movie, error) {
	records, err := r.driver.RunQuery(ctx, `
		MATCH (m:Movie)-[:AVAILABLE_ON {country: $country}]->(:Provider)
		WITH DISTINCT m
		ORDER BY m.popularity DESC, m.cinova_score DESC
		SKIP $offset
		LIMIT $limit
		OPTIONAL MATCH (m)-[:IN_GENRE]->(g:Genre)
		OPTIONAL MATCH (m)-[avail:AVAILABLE_ON {country: $country}]->(prov:Provider)
		RETURN m,
		       collect(DISTINCT {id: g.id, name: g.name})              AS genres,
		       collect(DISTINCT {provider_id: prov.provider_id,
		                          provider_name: prov.provider_name,
		                          logo_path: prov.logo_path,
		                          type: avail.type})                    AS providers
	`, map[string]interface{}{"country": country, "limit": limit, "offset": offset})
	if err != nil {
		return nil, fmt.Errorf("GetPopular query: %w", err)
	}

	movies := make([]models.Movie, 0, len(records))
	for _, rec := range records {
		node, _ := rec.Get("m")
		m := movieNodeToModel(node)
		if v, ok := rec.Get("genres"); ok {
			m.Genres = toGenres(v)
		}
		if v, ok := rec.Get("providers"); ok {
			m.Providers = toProviders(v)
		}
		movies = append(movies, *m)
	}
	return movies, nil
}

// GetReels returns movies for the full-screen vertical reel feed.
// ONLY returns movies with a verified embeddable vertical (9:16) trailer.
// Applies age bias: effective_score = cinova_score × exp(−0.15 × age_years)
func (r *MovieRepository) GetReels(ctx context.Context, country string, limit int) ([]models.Movie, error) {
	records, err := r.driver.RunQuery(ctx, `
		MATCH (m:Movie)
		WHERE m.vertical_trailer_youtube_key IS NOT NULL
		  AND m.vertical_trailer_youtube_key <> ''
		  AND m.vertical_trailer_youtube_key <> 'NOT_FOUND'
		WITH DISTINCT m,
		     m.cinova_score * exp(-0.15 * toFloat(date().year - toInteger(substring(coalesce(m.release_date, '2000-01-01'), 0, 4)))) * (0.4 + rand() * 0.6) AS effective_score
		ORDER BY effective_score DESC
		LIMIT $limit
		OPTIONAL MATCH (m)-[:IN_GENRE]->(g:Genre)
		OPTIONAL MATCH (m)-[avail:AVAILABLE_ON {country: $country}]->(prov:Provider)
		RETURN m,
		       collect(DISTINCT {id: g.id, name: g.name})              AS genres,
		       collect(DISTINCT {provider_id: prov.provider_id,
		                          provider_name: prov.provider_name,
		                          logo_path: prov.logo_path,
		                          type: avail.type})                    AS providers
	`, map[string]interface{}{"country": country, "limit": limit})
	if err != nil {
		return nil, fmt.Errorf("GetReels query: %w", err)
	}

	movies := make([]models.Movie, 0, len(records))
	for _, rec := range records {
		node, _ := rec.Get("m")
		m := movieNodeToModel(node)
		if v, ok := rec.Get("genres"); ok {
			m.Genres = toGenres(v)
		}
		if v, ok := rec.Get("providers"); ok {
			m.Providers = toProviders(v)
		}
		movies = append(movies, *m)
	}
	return movies, nil
}

// GetDiscoverMosaic returns quality movies for the Discover mosaic grid.
// Includes both trailer_youtube_key and vertical_trailer_youtube_key so the
// mobile client can render video tiles at their native aspect ratio.
// Applies age bias and random jitter for feed freshness. Supports pagination.
func (r *MovieRepository) GetDiscoverMosaic(ctx context.Context, country string, limit, offset int) ([]models.Movie, error) {
	records, err := r.driver.RunQuery(ctx, `
		MATCH (m:Movie)-[:AVAILABLE_ON {country: $country}]->(:Provider)
		WHERE m.poster_path IS NOT NULL AND m.poster_path <> ''
		  AND m.cinova_score > 50
		WITH DISTINCT m,
		     m.cinova_score * exp(-0.15 * toFloat(date().year - toInteger(substring(coalesce(m.release_date, '2000-01-01'), 0, 4)))) * (0.4 + rand() * 0.6) AS effective_score
		ORDER BY effective_score DESC
		SKIP $offset
		LIMIT $limit
		OPTIONAL MATCH (m)-[:IN_GENRE]->(g:Genre)
		OPTIONAL MATCH (m)-[avail:AVAILABLE_ON {country: $country}]->(prov:Provider)
		RETURN m,
		       collect(DISTINCT {id: g.id, name: g.name})              AS genres,
		       collect(DISTINCT {provider_id: prov.provider_id,
		                          provider_name: prov.provider_name,
		                          logo_path: prov.logo_path,
		                          type: avail.type})                    AS providers
	`, map[string]interface{}{"country": country, "limit": limit, "offset": offset})
	if err != nil {
		return nil, fmt.Errorf("GetDiscoverMosaic query: %w", err)
	}

	movies := make([]models.Movie, 0, len(records))
	for _, rec := range records {
		node, _ := rec.Get("m")
		m := movieNodeToModel(node)
		if v, ok := rec.Get("genres"); ok {
			m.Genres = toGenres(v)
		}
		if v, ok := rec.Get("providers"); ok {
			m.Providers = toProviders(v)
		}
		movies = append(movies, *m)
	}
	return movies, nil
}

// GetRecommendations returns personalised movie recommendations for a user or session.
// Falls back gracefully — callers should fall back to trending when the result is empty.
func (r *MovieRepository) GetRecommendations(ctx context.Context, ownerID, ownerType, country string, limit int) ([]models.Movie, error) {
	cypher := fmt.Sprintf(`
		MATCH (u:%s {id: $uid})-[:RATED {score: 1.0}]->(liked:Movie)
		MATCH (liked)-[:IN_GENRE]->(g:Genre)<-[:IN_GENRE]-(rec:Movie)
		WHERE NOT (u)-[:RATED|:DISMISSED]->(rec)
		  AND rec.cinova_score > 60
		  AND (rec)-[:AVAILABLE_ON {country: $country}]->(:Provider)
		WITH DISTINCT rec
		ORDER BY rec.cinova_score DESC
		LIMIT $limit
		OPTIONAL MATCH (rec)-[:IN_GENRE]->(g2:Genre)
		OPTIONAL MATCH (rec)-[avail:AVAILABLE_ON {country: $country}]->(prov:Provider)
		RETURN rec AS m,
		       collect(DISTINCT {id: g2.id, name: g2.name})            AS genres,
		       collect(DISTINCT {provider_id: prov.provider_id,
		                          provider_name: prov.provider_name,
		                          logo_path: prov.logo_path,
		                          type: avail.type})                    AS providers
	`, ownerType)

	records, err := r.driver.RunQuery(ctx, cypher, map[string]interface{}{
		"uid":     ownerID,
		"country": country,
		"limit":   limit,
	})
	if err != nil {
		return nil, fmt.Errorf("GetRecommendations query: %w", err)
	}

	movies := make([]models.Movie, 0, len(records))
	for _, rec := range records {
		node, _ := rec.Get("m")
		m := movieNodeToModel(node)
		if v, ok := rec.Get("genres"); ok {
			m.Genres = toGenres(v)
		}
		if v, ok := rec.Get("providers"); ok {
			m.Providers = toProviders(v)
		}
		movies = append(movies, *m)
	}
	return movies, nil
}

// personWithFilmography is the response shape for GetPerson.
type personWithFilmography struct {
	Person      models.Person          `json:"person"`
	Filmography []filmographyEntry     `json:"filmography"`
}

type filmographyEntry struct {
	TMDBID      int64   `json:"tmdb_id"`
	Title       string  `json:"title"`
	PosterPath  string  `json:"poster_path,omitempty"`
	CinovaScore float64 `json:"cinova_score,omitempty"`
	Year        string  `json:"year,omitempty"`
	Role        string  `json:"role"` // "ACTED_IN" or "DIRECTED"
}

// GetPerson retrieves a person node and their filmography.
func (r *MovieRepository) GetPerson(ctx context.Context, tmdbID int) (*personWithFilmography, error) {
	records, err := r.driver.RunQuery(ctx, `
		MATCH (p:Person {tmdb_id: $id})
		OPTIONAL MATCH (p)-[rel:ACTED_IN|DIRECTED]->(m:Movie)
		WITH p,
		     collect({tmdb_id: m.tmdb_id, title: m.title, poster_path: m.poster_path,
		               cinova_score: m.cinova_score, year: m.release_year,
		               role: type(rel)}) AS filmography
		RETURN p, filmography
	`, map[string]interface{}{"id": tmdbID})
	if err != nil {
		return nil, fmt.Errorf("GetPerson query: %w", err)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("person %d not found", tmdbID)
	}

	rec := records[0]
	node, _ := rec.Get("p")

	person := models.Person{}
	if n, ok := node.(dbtype.Node); ok {
		props := n.Props
		person.TMDBID = int64Val(props["tmdb_id"])
		person.Name = strVal(props["name"])
		person.ProfilePath = strVal(props["profile_path"])
		person.WikidataID = strVal(props["wikidata_id"])
		person.Department = strVal(props["department"])
		person.Job = strVal(props["job"])
	}

	filmography := make([]filmographyEntry, 0)
	if v, ok := rec.Get("filmography"); ok {
		list, _ := v.([]interface{})
		for _, item := range list {
			m, _ := item.(map[string]interface{})
			if m == nil || m["tmdb_id"] == nil {
				continue
			}
			filmography = append(filmography, filmographyEntry{
				TMDBID:      int64Val(m["tmdb_id"]),
				Title:       strVal(m["title"]),
				PosterPath:  strVal(m["poster_path"]),
				CinovaScore: float64Val(m["cinova_score"]),
				Year:        strVal(m["year"]),
				Role:        strVal(m["role"]),
			})
		}
	}

	return &personWithFilmography{
		Person:      person,
		Filmography: filmography,
	}, nil
}

// UnsaveTitle removes a SAVED relationship from an owner to a title.
func (r *MovieRepository) UnsaveTitle(ctx context.Context, ownerID string, ownerType string, tmdbID int) error {
	cypher := fmt.Sprintf(`
		MATCH (owner:%s {id: $owner_id})-[rel:SAVED]->(n)
		WHERE (n:Movie OR n:TVShow) AND n.tmdb_id = $tmdb_id
		DELETE rel
	`, ownerType)

	return r.driver.RunWriteUnit(ctx, cypher, map[string]interface{}{
		"owner_id": ownerID,
		"tmdb_id":  tmdbID,
	})
}
