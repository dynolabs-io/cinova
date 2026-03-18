package graph

import (
	"context"
	"fmt"
	"strings"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j/dbtype"

	"github.com/foundrylab-app/cinova/backend/internal/models"
)

// latinize converts Turkish (and common accented) characters to their ASCII
// equivalents so name searches work even when the model omits diacritics.
func latinize(s string) string {
	r := strings.ToLower(s)
	for _, pair := range [][2]string{
		{"ı", "i"}, {"ğ", "g"}, {"ü", "u"}, {"ş", "s"}, {"ö", "o"}, {"ç", "c"},
		{"â", "a"}, {"î", "i"}, {"û", "u"}, {"é", "e"}, {"è", "e"}, {"ê", "e"},
		{"à", "a"}, {"á", "a"}, {"ñ", "n"}, {"ô", "o"}, {"ò", "o"}, {"ó", "o"},
	} {
		r = strings.ReplaceAll(r, pair[0], pair[1])
	}
	return r
}

// latinizeAll applies latinize to every element of a slice.
func latinizeAll(ss []string) []string {
	out := make([]string, len(ss))
	for i, s := range ss {
		out[i] = latinize(s)
	}
	return out
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
		params["directors"] = latinizeAll(f.Directors)
		// Normalize stored name the same way so Turkish diacritics match ASCII queries
		conds = append(conds, "EXISTS { MATCH (d:Person)-[:DIRECTED]->(m) WHERE any(dn IN $directors WHERE replace(replace(replace(replace(replace(replace(toLower(d.name),'ı','i'),'ğ','g'),'ü','u'),'ş','s'),'ö','o'),'ç','c') CONTAINS dn) }")
	}
	if len(f.Actors) > 0 {
		params["actors"] = latinizeAll(f.Actors)
		conds = append(conds, "EXISTS { MATCH (a:Person)-[:ACTED_IN]->(m) WHERE any(an IN $actors WHERE replace(replace(replace(replace(replace(replace(toLower(a.name),'ı','i'),'ğ','g'),'ü','u'),'ş','s'),'ö','o'),'ç','c') CONTAINS an) }")
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

// GetReels returns trending movies optimised for the Discover reel feed.
// Results include backdrop_path, overview, genres, cinova_score, and streaming providers.
func (r *MovieRepository) GetReels(ctx context.Context, country string, limit int) ([]models.Movie, error) {
	records, err := r.driver.RunQuery(ctx, `
		MATCH (m:Movie)-[:AVAILABLE_ON {country: $country}]->(:Provider)
		WHERE m.backdrop_path IS NOT NULL AND m.backdrop_path <> ''
		  AND m.overview IS NOT NULL AND m.overview <> ''
		  AND m.cinova_score > 50
		WITH DISTINCT m
		ORDER BY m.cinova_score DESC, m.popularity DESC
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
