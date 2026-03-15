package graph

import (
	"github.com/neo4j/neo4j-go-driver/v5/neo4j/dbtype"

	"context"
	"fmt"

	"github.com/foundrylab-app/cinova/backend/internal/models"
)

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
