package graph

import (
	"context"
	"fmt"

	"github.com/foundrylab-app/cinova/backend/internal/models"
)

// MovieRepository provides graph operations for movies, TV shows, people, and providers.
type MovieRepository struct {
	driver *Driver
}

// NewMovieRepository creates a MovieRepository backed by the given Driver.
func NewMovieRepository(driver *Driver) *MovieRepository {
	return &MovieRepository{driver: driver}
}

// GetMovie retrieves a full movie node with genres, themes, cast, and directors.
func (r *MovieRepository) GetMovie(ctx context.Context, tmdbID int) (*models.Movie, error) {
	records, err := r.driver.RunQuery(ctx, `
		MATCH (m:Movie {tmdb_id: $tmdb_id})
		OPTIONAL MATCH (m)-[:IN_GENRE]->(g:Genre)
		OPTIONAL MATCH (m)-[:HAS_THEME]->(t:Theme)
		OPTIONAL MATCH (m)-[:HAS_MOOD]->(mo:Mood)
		OPTIONAL MATCH (m)<-[act:ACTED_IN]-(p:Person)
		OPTIONAL MATCH (m)<-[dir:DIRECTED]-(d:Person)
		RETURN m,
		       collect(DISTINCT {id: g.id, name: g.name})              AS genres,
		       collect(DISTINCT {name: t.name, score: t.score})         AS themes,
		       collect(DISTINCT {name: mo.name, score: mo.score})       AS moods,
		       collect(DISTINCT {tmdb_id: p.tmdb_id, name: p.name,
		                          profile_path: p.profile_path,
		                          role: act.character, order: act.order}) AS cast,
		       collect(DISTINCT {tmdb_id: d.tmdb_id, name: d.name,
		                          profile_path: d.profile_path,
		                          job: dir.job})                         AS directors
	`, map[string]interface{}{"tmdb_id": tmdbID})
	if err != nil {
		return nil, fmt.Errorf("GetMovie query: %w", err)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("movie %d not found", tmdbID)
	}

	rec := records[0]
	node, _ := rec.Get("m")
	movie := movieNodeToModel(node)

	if v, ok := rec.Get("genres"); ok {
		movie.Genres = toGenres(v)
	}
	if v, ok := rec.Get("themes"); ok {
		movie.Themes = toThemes(v)
	}
	if v, ok := rec.Get("moods"); ok {
		movie.Moods = toMoods(v)
	}
	if v, ok := rec.Get("cast"); ok {
		movie.Cast = toPeople(v)
	}
	if v, ok := rec.Get("directors"); ok {
		movie.Directors = toPeople(v)
	}

	return movie, nil
}

// GetTVShow retrieves a full TV show node with genres, themes, cast, and creators.
func (r *MovieRepository) GetTVShow(ctx context.Context, tmdbID int) (*models.TVShow, error) {
	records, err := r.driver.RunQuery(ctx, `
		MATCH (t:TVShow {tmdb_id: $tmdb_id})
		OPTIONAL MATCH (t)-[:IN_GENRE]->(g:Genre)
		OPTIONAL MATCH (t)-[:HAS_THEME]->(th:Theme)
		OPTIONAL MATCH (t)-[:HAS_MOOD]->(mo:Mood)
		OPTIONAL MATCH (t)<-[act:ACTED_IN]-(p:Person)
		OPTIONAL MATCH (t)<-[cr:CREATED]-(c:Person)
		RETURN t,
		       collect(DISTINCT {id: g.id, name: g.name})               AS genres,
		       collect(DISTINCT {name: th.name, score: th.score})        AS themes,
		       collect(DISTINCT {name: mo.name, score: mo.score})        AS moods,
		       collect(DISTINCT {tmdb_id: p.tmdb_id, name: p.name,
		                          profile_path: p.profile_path,
		                          role: act.character, order: act.order}) AS cast,
		       collect(DISTINCT {tmdb_id: c.tmdb_id, name: c.name,
		                          profile_path: c.profile_path})          AS creators
	`, map[string]interface{}{"tmdb_id": tmdbID})
	if err != nil {
		return nil, fmt.Errorf("GetTVShow query: %w", err)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("tv show %d not found", tmdbID)
	}

	rec := records[0]
	node, _ := rec.Get("t")
	show := tvNodeToModel(node)

	if v, ok := rec.Get("genres"); ok {
		show.Genres = toGenres(v)
	}
	if v, ok := rec.Get("themes"); ok {
		show.Themes = toThemes(v)
	}
	if v, ok := rec.Get("moods"); ok {
		show.Moods = toMoods(v)
	}
	if v, ok := rec.Get("cast"); ok {
		show.Cast = toPeople(v)
	}
	if v, ok := rec.Get("creators"); ok {
		show.Creators = toPeople(v)
	}

	return show, nil
}

// GetStreamingProviders returns streaming providers for a title in a given country.
func (r *MovieRepository) GetStreamingProviders(ctx context.Context, tmdbID int, country string) ([]models.Provider, error) {
	records, err := r.driver.RunQuery(ctx, `
		MATCH (n)-[avail:AVAILABLE_ON {country: $country}]->(prov:Provider)
		WHERE (n:Movie OR n:TVShow) AND n.tmdb_id = $tmdb_id
		RETURN prov.provider_id   AS provider_id,
		       prov.provider_name AS provider_name,
		       prov.logo_path     AS logo_path,
		       avail.type         AS type,
		       avail.country      AS country
		ORDER BY prov.display_priority ASC
	`, map[string]interface{}{"tmdb_id": tmdbID, "country": country})
	if err != nil {
		return nil, fmt.Errorf("GetStreamingProviders query: %w", err)
	}

	providers := make([]models.Provider, 0, len(records))
	for _, rec := range records {
		p := models.Provider{}
		if v, ok := rec.Get("provider_id"); ok {
			p.ProviderID = int64Val(v)
		}
		if v, ok := rec.Get("provider_name"); ok {
			p.ProviderName = strVal(v)
		}
		if v, ok := rec.Get("logo_path"); ok {
			p.LogoPath = strVal(v)
		}
		if v, ok := rec.Get("type"); ok {
			p.Type = strVal(v)
		}
		if v, ok := rec.Get("country"); ok {
			p.Country = strVal(v)
		}
		providers = append(providers, p)
	}
	return providers, nil
}

// GetTrending returns trending movies in a country ordered by Cinova score.
func (r *MovieRepository) GetTrending(ctx context.Context, country string, limit int) ([]models.Movie, error) {
	records, err := r.driver.RunQuery(ctx, `
		MATCH (m:Movie)-[:AVAILABLE_ON {country: $country}]->(:Provider)
		WITH DISTINCT m
		ORDER BY m.cinova_score DESC, m.popularity DESC
		LIMIT $limit
		OPTIONAL MATCH (m)-[:IN_GENRE]->(g:Genre)
		OPTIONAL MATCH (m)-[:AVAILABLE_ON {country: $country}]->(prov:Provider)
		RETURN m,
		       collect(DISTINCT {id: g.id, name: g.name})              AS genres,
		       collect(DISTINCT {provider_id: prov.provider_id,
		                          provider_name: prov.provider_name,
		                          logo_path: prov.logo_path})            AS providers
	`, map[string]interface{}{"country": country, "limit": limit})
	if err != nil {
		return nil, fmt.Errorf("GetTrending query: %w", err)
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

// UpsertMovie creates or updates a Movie node and its related Genre/Person relationships.
func (r *MovieRepository) UpsertMovie(ctx context.Context, movie *models.Movie) error {
	err := r.driver.RunWriteUnit(ctx, `
		MERGE (m:Movie {tmdb_id: $tmdb_id})
		SET m.imdb_id           = $imdb_id,
		    m.title             = $title,
		    m.original_title    = $original_title,
		    m.overview          = $overview,
		    m.release_date      = $release_date,
		    m.runtime           = $runtime,
		    m.vote_average      = $vote_average,
		    m.vote_count        = $vote_count,
		    m.popularity        = $popularity,
		    m.poster_path       = $poster_path,
		    m.backdrop_path     = $backdrop_path,
		    m.original_language = $original_language,
		    m.adult             = $adult,
		    m.cinova_score      = $cinova_score
	`, map[string]interface{}{
		"tmdb_id":           movie.TMDBID,
		"imdb_id":           movie.IMDbID,
		"title":             movie.Title,
		"original_title":    movie.OriginalTitle,
		"overview":          movie.Overview,
		"release_date":      movie.ReleaseDate,
		"runtime":           movie.Runtime,
		"vote_average":      movie.VoteAverage,
		"vote_count":        movie.VoteCount,
		"popularity":        movie.Popularity,
		"poster_path":       movie.PosterPath,
		"backdrop_path":     movie.BackdropPath,
		"original_language": movie.OriginalLanguage,
		"adult":             movie.Adult,
		"cinova_score":      movie.CinovaScore,
	})
	if err != nil {
		return fmt.Errorf("UpsertMovie: %w", err)
	}

	// Upsert genres
	for _, g := range movie.Genres {
		if err := r.driver.RunWriteUnit(ctx, `
			MERGE (g:Genre {id: $id}) SET g.name = $name
			WITH g
			MATCH (m:Movie {tmdb_id: $tmdb_id})
			MERGE (m)-[:IN_GENRE]->(g)
		`, map[string]interface{}{"id": g.ID, "name": g.Name, "tmdb_id": movie.TMDBID}); err != nil {
			return fmt.Errorf("UpsertMovie genre %d: %w", g.ID, err)
		}
	}

	// Upsert cast
	for _, p := range movie.Cast {
		if err := r.driver.RunWriteUnit(ctx, `
			MERGE (p:Person {tmdb_id: $tmdb_id})
			SET p.name = $name, p.profile_path = $profile_path
			WITH p
			MATCH (m:Movie {tmdb_id: $movie_tmdb_id})
			MERGE (p)-[a:ACTED_IN {movie_tmdb_id: $movie_tmdb_id}]->(m)
			SET a.character = $role, a.order = $order
		`, map[string]interface{}{
			"tmdb_id":       p.TMDBID,
			"name":          p.Name,
			"profile_path":  p.ProfilePath,
			"movie_tmdb_id": movie.TMDBID,
			"role":          p.Role,
			"order":         p.Order,
		}); err != nil {
			return fmt.Errorf("UpsertMovie cast person %d: %w", p.TMDBID, err)
		}
	}

	// Upsert directors
	for _, d := range movie.Directors {
		if err := r.driver.RunWriteUnit(ctx, `
			MERGE (p:Person {tmdb_id: $tmdb_id})
			SET p.name = $name, p.profile_path = $profile_path
			WITH p
			MATCH (m:Movie {tmdb_id: $movie_tmdb_id})
			MERGE (p)-[dir:DIRECTED {movie_tmdb_id: $movie_tmdb_id}]->(m)
			SET dir.job = $job
		`, map[string]interface{}{
			"tmdb_id":       d.TMDBID,
			"name":          d.Name,
			"profile_path":  d.ProfilePath,
			"movie_tmdb_id": movie.TMDBID,
			"job":           d.Job,
		}); err != nil {
			return fmt.Errorf("UpsertMovie director %d: %w", d.TMDBID, err)
		}
	}

	return nil
}

// UpsertPerson creates or updates a Person node.
func (r *MovieRepository) UpsertPerson(ctx context.Context, person *models.Person) error {
	return r.driver.RunWriteUnit(ctx, `
		MERGE (p:Person {tmdb_id: $tmdb_id})
		SET p.name         = $name,
		    p.profile_path = $profile_path,
		    p.wikidata_id  = $wikidata_id,
		    p.department   = $department,
		    p.job          = $job
	`, map[string]interface{}{
		"tmdb_id":      person.TMDBID,
		"name":         person.Name,
		"profile_path": person.ProfilePath,
		"wikidata_id":  person.WikidataID,
		"department":   person.Department,
		"job":          person.Job,
	})
}

// UpsertProvider creates or updates streaming provider availability for a title.
func (r *MovieRepository) UpsertProvider(ctx context.Context, tmdbID int, providers []models.Provider, country string) error {
	// Remove stale provider edges for this country before re-adding
	if err := r.driver.RunWriteUnit(ctx, `
		MATCH (n)-[a:AVAILABLE_ON {country: $country}]->(:Provider)
		WHERE (n:Movie OR n:TVShow) AND n.tmdb_id = $tmdb_id
		DELETE a
	`, map[string]interface{}{"tmdb_id": tmdbID, "country": country}); err != nil {
		return fmt.Errorf("UpsertProvider remove stale: %w", err)
	}

	for _, p := range providers {
		if err := r.driver.RunWriteUnit(ctx, `
			MERGE (prov:Provider {provider_id: $provider_id})
			SET prov.provider_name    = $provider_name,
			    prov.logo_path        = $logo_path,
			    prov.display_priority = $display_priority
			WITH prov
			MATCH (n)
			WHERE (n:Movie OR n:TVShow) AND n.tmdb_id = $tmdb_id
			MERGE (n)-[a:AVAILABLE_ON {country: $country, type: $type}]->(prov)
		`, map[string]interface{}{
			"provider_id":      p.ProviderID,
			"provider_name":    p.ProviderName,
			"logo_path":        p.LogoPath,
			"display_priority": p.DisplayPriority,
			"tmdb_id":          tmdbID,
			"country":          country,
			"type":             p.Type,
		}); err != nil {
			return fmt.Errorf("UpsertProvider %d: %w", p.ProviderID, err)
		}
	}
	return nil
}

// UpsertTheme creates or updates a Theme node and links it to a Movie or TVShow.
// mediaType should be "movie" or "tv".
func (r *MovieRepository) UpsertTheme(ctx context.Context, tmdbID int, name string, score float64, mediaType string) error {
	nodeLabel := "Movie"
	if mediaType == "tv" {
		nodeLabel = "TVShow"
	}
	cypher := fmt.Sprintf(`
		MERGE (t:Theme {name: $name})
		SET t.score = $score
		WITH t
		MATCH (n:%s {tmdb_id: $tmdb_id})
		MERGE (n)-[:HAS_THEME]->(t)
	`, nodeLabel)
	return r.driver.RunWriteUnit(ctx, cypher, map[string]interface{}{
		"name":    name,
		"score":   score,
		"tmdb_id": tmdbID,
	})
}

// UpsertMood creates or updates a Mood node and links it to a Movie or TVShow.
// mediaType should be "movie" or "tv".
func (r *MovieRepository) UpsertMood(ctx context.Context, tmdbID int, name string, score float64, mediaType string) error {
	nodeLabel := "Movie"
	if mediaType == "tv" {
		nodeLabel = "TVShow"
	}
	cypher := fmt.Sprintf(`
		MERGE (m:Mood {name: $name})
		SET m.score = $score
		WITH m
		MATCH (n:%s {tmdb_id: $tmdb_id})
		MERGE (n)-[:HAS_MOOD]->(m)
	`, nodeLabel)
	return r.driver.RunWriteUnit(ctx, cypher, map[string]interface{}{
		"name":    name,
		"score":   score,
		"tmdb_id": tmdbID,
	})
}

// ---- Node conversion helpers ----

func movieNodeToModel(v interface{}) *models.Movie {
	m := &models.Movie{}
	node, ok := v.(interface {
		Props() map[string]interface{}
	})
	if !ok {
		return m
	}
	props := node.Props()
	m.TMDBID = int64Val(props["tmdb_id"])
	m.IMDbID = strVal(props["imdb_id"])
	m.Title = strVal(props["title"])
	m.OriginalTitle = strVal(props["original_title"])
	m.Overview = strVal(props["overview"])
	m.ReleaseDate = strVal(props["release_date"])
	m.Runtime = int(int64Val(props["runtime"]))
	m.VoteAverage = float64Val(props["vote_average"])
	m.VoteCount = int64Val(props["vote_count"])
	m.Popularity = float64Val(props["popularity"])
	m.PosterPath = strVal(props["poster_path"])
	m.BackdropPath = strVal(props["backdrop_path"])
	m.OriginalLanguage = strVal(props["original_language"])
	m.Adult = boolVal(props["adult"])
	m.CinovaScore = float64Val(props["cinova_score"])
	return m
}

func tvNodeToModel(v interface{}) *models.TVShow {
	t := &models.TVShow{}
	node, ok := v.(interface {
		Props() map[string]interface{}
	})
	if !ok {
		return t
	}
	props := node.Props()
	t.TMDBID = int64Val(props["tmdb_id"])
	t.Name = strVal(props["name"])
	t.OriginalName = strVal(props["original_name"])
	t.Overview = strVal(props["overview"])
	t.FirstAirDate = strVal(props["first_air_date"])
	t.LastAirDate = strVal(props["last_air_date"])
	t.NumberOfSeasons = int(int64Val(props["number_of_seasons"]))
	t.NumberOfEpisodes = int(int64Val(props["number_of_episodes"]))
	t.VoteAverage = float64Val(props["vote_average"])
	t.VoteCount = int64Val(props["vote_count"])
	t.Popularity = float64Val(props["popularity"])
	t.PosterPath = strVal(props["poster_path"])
	t.BackdropPath = strVal(props["backdrop_path"])
	t.OriginalLanguage = strVal(props["original_language"])
	t.Status = strVal(props["status"])
	t.CinovaScore = float64Val(props["cinova_score"])
	return t
}

// toGenres converts a Neo4j list of genre maps into []models.Genre.
func toGenres(v interface{}) []models.Genre {
	list, _ := v.([]interface{})
	genres := make([]models.Genre, 0, len(list))
	for _, item := range list {
		m, _ := item.(map[string]interface{})
		if m == nil {
			continue
		}
		if m["id"] == nil {
			continue
		}
		genres = append(genres, models.Genre{
			ID:   int64Val(m["id"]),
			Name: strVal(m["name"]),
		})
	}
	return genres
}

// toThemes converts a Neo4j list of theme maps into []models.Theme.
func toThemes(v interface{}) []models.Theme {
	list, _ := v.([]interface{})
	themes := make([]models.Theme, 0, len(list))
	for _, item := range list {
		m, _ := item.(map[string]interface{})
		if m == nil || m["name"] == nil {
			continue
		}
		themes = append(themes, models.Theme{
			Name:  strVal(m["name"]),
			Score: float64Val(m["score"]),
		})
	}
	return themes
}

// toMoods converts a Neo4j list of mood maps into []models.Mood.
func toMoods(v interface{}) []models.Mood {
	list, _ := v.([]interface{})
	moods := make([]models.Mood, 0, len(list))
	for _, item := range list {
		m, _ := item.(map[string]interface{})
		if m == nil || m["name"] == nil {
			continue
		}
		moods = append(moods, models.Mood{
			Name:  strVal(m["name"]),
			Score: float64Val(m["score"]),
		})
	}
	return moods
}

// toPeople converts a Neo4j list of person maps into []models.Person.
func toPeople(v interface{}) []models.Person {
	list, _ := v.([]interface{})
	people := make([]models.Person, 0, len(list))
	for _, item := range list {
		m, _ := item.(map[string]interface{})
		if m == nil || m["tmdb_id"] == nil {
			continue
		}
		people = append(people, models.Person{
			TMDBID:      int64Val(m["tmdb_id"]),
			Name:        strVal(m["name"]),
			ProfilePath: strVal(m["profile_path"]),
			Role:        strVal(m["role"]),
			Job:         strVal(m["job"]),
			Order:       int(int64Val(m["order"])),
		})
	}
	return people
}

// toProviders converts a Neo4j list of provider maps into []models.Provider.
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
		})
	}
	return providers
}
