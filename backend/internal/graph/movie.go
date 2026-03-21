package graph

import (
	"context"
	"fmt"
	"strings"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j/dbtype"

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

// GetMovie retrieves a full movie node with genres, keywords, themes, cast, directors, and awards.
func (r *MovieRepository) GetMovie(ctx context.Context, tmdbID int) (*models.Movie, error) {
	records, err := r.driver.RunQuery(ctx, `
		MATCH (m:Movie {tmdb_id: $tmdb_id})
		CALL { WITH m OPTIONAL MATCH (m)-[:IN_GENRE]->(g:Genre)
		       RETURN collect(DISTINCT {id: g.id, name: g.name}) AS genres }
		CALL { WITH m OPTIONAL MATCH (m)-[:HAS_KEYWORD]->(k:Keyword)
		       RETURN collect(DISTINCT {id: k.id, name: k.name}) AS keywords }
		CALL { WITH m OPTIONAL MATCH (m)-[:HAS_THEME]->(t:Theme)
		       RETURN collect(DISTINCT {name: t.name, score: t.score}) AS themes }
		CALL { WITH m OPTIONAL MATCH (m)-[:HAS_MOOD]->(mo:Mood)
		       RETURN collect(DISTINCT {name: mo.name, score: mo.score}) AS moods }
		CALL { WITH m OPTIONAL MATCH (m)<-[act:ACTED_IN]-(p:Person)
		       RETURN collect(DISTINCT {tmdb_id: p.tmdb_id, name: p.name,
		                                profile_path: p.profile_path,
		                                role: act.character, order: act.order}) AS cast }
		CALL { WITH m OPTIONAL MATCH (m)<-[dir:DIRECTED]-(d:Person)
		       RETURN collect(DISTINCT {tmdb_id: d.tmdb_id, name: d.name,
		                                profile_path: d.profile_path,
		                                job: dir.job}) AS directors }
		CALL { WITH m OPTIONAL MATCH (m)-[won:HAS_WON]->(aw:Award)
		       RETURN collect(DISTINCT {wikidata_id: aw.wikidata_id, award_name: aw.award_name,
		                                ceremony_name: aw.ceremony_name, year: won.year,
		                                recipient_name: won.recipient_name,
		                                is_nomination: false}) AS wins }
		CALL { WITH m OPTIONAL MATCH (m)-[nom:HAS_NOMINATION]->(na:Award)
		       RETURN collect(DISTINCT {wikidata_id: na.wikidata_id, award_name: na.award_name,
		                                ceremony_name: na.ceremony_name, year: nom.year,
		                                is_nomination: true}) AS nominations }
		RETURN m, genres, keywords, themes, moods, cast, directors, wins, nominations
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
	if v, ok := rec.Get("keywords"); ok {
		movie.Keywords = toKeywords(v)
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
	if v, ok := rec.Get("wins"); ok {
		movie.Awards = append(movie.Awards, toAwards(v)...)
	}
	if v, ok := rec.Get("nominations"); ok {
		movie.Awards = append(movie.Awards, toAwards(v)...)
	}

	return movie, nil
}

// GetTVShow retrieves a full TV show node with genres, keywords, themes, cast, and creators.
func (r *MovieRepository) GetTVShow(ctx context.Context, tmdbID int) (*models.TVShow, error) {
	records, err := r.driver.RunQuery(ctx, `
		MATCH (t:TVShow {tmdb_id: $tmdb_id})
		CALL { WITH t OPTIONAL MATCH (t)-[:IN_GENRE]->(g:Genre)
		       RETURN collect(DISTINCT {id: g.id, name: g.name}) AS genres }
		CALL { WITH t OPTIONAL MATCH (t)-[:HAS_KEYWORD]->(k:Keyword)
		       RETURN collect(DISTINCT {id: k.id, name: k.name}) AS keywords }
		CALL { WITH t OPTIONAL MATCH (t)-[:HAS_THEME]->(th:Theme)
		       RETURN collect(DISTINCT {name: th.name, score: th.score}) AS themes }
		CALL { WITH t OPTIONAL MATCH (t)-[:HAS_MOOD]->(mo:Mood)
		       RETURN collect(DISTINCT {name: mo.name, score: mo.score}) AS moods }
		CALL { WITH t OPTIONAL MATCH (t)<-[act:ACTED_IN]-(p:Person)
		       RETURN collect(DISTINCT {tmdb_id: p.tmdb_id, name: p.name,
		                                profile_path: p.profile_path,
		                                role: act.character, order: act.order}) AS cast }
		CALL { WITH t OPTIONAL MATCH (t)<-[cr:CREATED]-(c:Person)
		       RETURN collect(DISTINCT {tmdb_id: c.tmdb_id, name: c.name,
		                                profile_path: c.profile_path}) AS creators }
		RETURN t, genres, keywords, themes, moods, cast, creators
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
	if v, ok := rec.Get("keywords"); ok {
		show.Keywords = toKeywords(v)
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
	// Type priority: flatrate > free > rent > buy — used to pick the best when
	// the same provider appears under multiple availability types.
	typePriority := map[string]int{"flatrate": 1, "free": 2, "rent": 3, "buy": 4}

	records, err := r.driver.RunQuery(ctx, `
		MATCH (n:Movie)-[avail:AVAILABLE_ON {country: $country}]->(prov:Provider)
		WHERE n.tmdb_id = $tmdb_id
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

	// Deduplicate by provider_id: same provider can appear for multiple
	// availability types (flatrate + rent). Keep the highest-priority type.
	seen := make(map[int64]int) // provider_id → index in providers slice
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
		if idx, exists := seen[p.ProviderID]; exists {
			// Replace if this type has higher priority
			if typePriority[p.Type] < typePriority[providers[idx].Type] {
				providers[idx].Type = p.Type
			}
		} else {
			seen[p.ProviderID] = len(providers)
			providers = append(providers, p)
		}
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
		OPTIONAL MATCH (m)-[avail:AVAILABLE_ON {country: $country}]->(prov:Provider)
		RETURN m,
		       collect(DISTINCT {id: g.id, name: g.name})              AS genres,
		       collect(DISTINCT {provider_id: prov.provider_id,
		                          provider_name: prov.provider_name,
		                          logo_path: prov.logo_path,
		                          type: avail.type})                     AS providers
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

// UpsertMovie creates or updates a Movie node and its related Genre/Person/Keyword relationships.
func (r *MovieRepository) UpsertMovie(ctx context.Context, movie *models.Movie) error {
	err := r.driver.RunWriteUnit(ctx, `
		MERGE (m:Movie {tmdb_id: $tmdb_id})
		SET m.imdb_id              = $imdb_id,
		    m.wikidata_id          = $wikidata_id,
		    m.title                = $title,
		    m.original_title       = $original_title,
		    m.tagline              = $tagline,
		    m.overview             = $overview,
		    m.release_date         = $release_date,
		    m.runtime              = $runtime,
		    m.vote_average         = $vote_average,
		    m.vote_count           = $vote_count,
		    m.popularity           = $popularity,
		    m.budget               = $budget,
		    m.revenue              = $revenue,
		    m.certification        = $certification,
		    m.trailer_youtube_key  = $trailer_youtube_key,
		    m.poster_path          = $poster_path,
		    m.backdrop_path        = $backdrop_path,
		    m.original_language    = $original_language,
		    m.spoken_languages     = $spoken_languages,
		    m.collection_id        = $collection_id,
		    m.collection_name      = $collection_name,
		    m.adult                = $adult,
		    m.cinova_score         = $cinova_score,
		    m.plot_summary         = CASE WHEN $plot_summary <> '' THEN $plot_summary ELSE coalesce(m.plot_summary, '') END,
		    m.cinova_synopsis      = CASE WHEN $cinova_synopsis <> '' THEN $cinova_synopsis ELSE coalesce(m.cinova_synopsis, '') END,
		    m.cinova_editorial     = CASE WHEN $cinova_editorial <> '' THEN $cinova_editorial ELSE coalesce(m.cinova_editorial, '') END
	`, map[string]interface{}{
		"tmdb_id":             movie.TMDBID,
		"imdb_id":             movie.IMDbID,
		"wikidata_id":         movie.WikidataID,
		"title":               movie.Title,
		"original_title":      movie.OriginalTitle,
		"tagline":             movie.Tagline,
		"overview":            movie.Overview,
		"release_date":        movie.ReleaseDate,
		"runtime":             movie.Runtime,
		"vote_average":        movie.VoteAverage,
		"vote_count":          movie.VoteCount,
		"popularity":          movie.Popularity,
		"budget":              movie.Budget,
		"revenue":             movie.Revenue,
		"certification":       movie.Certification,
		"trailer_youtube_key": movie.TrailerYouTubeKey,
		"poster_path":         movie.PosterPath,
		"backdrop_path":       movie.BackdropPath,
		"original_language":   movie.OriginalLanguage,
		"spoken_languages":    movie.SpokenLanguages,
		"collection_id":       movie.CollectionID,
		"collection_name":     movie.CollectionName,
		"adult":               movie.Adult,
		"cinova_score":        movie.CinovaScore,
		"plot_summary":        movie.PlotSummary,
		"cinova_synopsis":     movie.CinovaSynopsis,
		"cinova_editorial":    movie.CinovaEditorial,
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

	// Upsert keywords (UNWIND for efficiency — one round trip)
	if len(movie.Keywords) > 0 {
		kwMaps := make([]map[string]interface{}, len(movie.Keywords))
		for i, kw := range movie.Keywords {
			kwMaps[i] = map[string]interface{}{"id": kw.ID, "name": kw.Name}
		}
		if err := r.driver.RunWriteUnit(ctx, `
			UNWIND $keywords AS kw
			MERGE (k:Keyword {id: kw.id}) SET k.name = kw.name
			WITH k
			MATCH (m:Movie {tmdb_id: $tmdb_id})
			MERGE (m)-[:HAS_KEYWORD]->(k)
		`, map[string]interface{}{"keywords": kwMaps, "tmdb_id": movie.TMDBID}); err != nil {
			return fmt.Errorf("UpsertMovie keywords: %w", err)
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

	// Upsert writers
	for _, w := range movie.Writers {
		if err := r.driver.RunWriteUnit(ctx, `
			MERGE (p:Person {tmdb_id: $tmdb_id})
			SET p.name = $name, p.profile_path = $profile_path
			WITH p
			MATCH (m:Movie {tmdb_id: $movie_tmdb_id})
			MERGE (p)-[wr:WROTE {movie_tmdb_id: $movie_tmdb_id}]->(m)
			SET wr.job = $job
		`, map[string]interface{}{
			"tmdb_id":       w.TMDBID,
			"name":          w.Name,
			"profile_path":  w.ProfilePath,
			"movie_tmdb_id": movie.TMDBID,
			"job":           w.Job,
		}); err != nil {
			return fmt.Errorf("UpsertMovie writer %d: %w", w.TMDBID, err)
		}
	}

	// Upsert producers
	for _, pr := range movie.Producers {
		if err := r.driver.RunWriteUnit(ctx, `
			MERGE (p:Person {tmdb_id: $tmdb_id})
			SET p.name = $name, p.profile_path = $profile_path
			WITH p
			MATCH (m:Movie {tmdb_id: $movie_tmdb_id})
			MERGE (p)-[prod:PRODUCED {movie_tmdb_id: $movie_tmdb_id}]->(m)
			SET prod.job = $job
		`, map[string]interface{}{
			"tmdb_id":       pr.TMDBID,
			"name":          pr.Name,
			"profile_path":  pr.ProfilePath,
			"movie_tmdb_id": movie.TMDBID,
			"job":           pr.Job,
		}); err != nil {
			return fmt.Errorf("UpsertMovie producer %d: %w", pr.TMDBID, err)
		}
	}

	// Upsert streaming providers
	if len(movie.Providers) > 0 {
		// Group by country — usually all providers share the same country in ingestion
		byCountry := make(map[string][]models.Provider)
		for _, p := range movie.Providers {
			byCountry[p.Country] = append(byCountry[p.Country], p)
		}
		for country, provs := range byCountry {
			if err := r.UpsertProvider(ctx, int(movie.TMDBID), provs, country); err != nil {
				return fmt.Errorf("UpsertMovie providers: %w", err)
			}
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
	m := &models.Movie{MediaType: "movie"}
	node, ok := v.(dbtype.Node)
	if !ok {
		return m
	}
	props := node.Props
	m.TMDBID = int64Val(props["tmdb_id"])
	m.IMDbID = strVal(props["imdb_id"])
	m.WikidataID = strVal(props["wikidata_id"])
	m.Title = strVal(props["title"])
	m.OriginalTitle = strVal(props["original_title"])
	m.Tagline = strVal(props["tagline"])
	m.Overview = strVal(props["overview"])
	m.ReleaseDate = strVal(props["release_date"])
	m.Runtime = int(int64Val(props["runtime"]))
	m.VoteAverage = float64Val(props["vote_average"])
	m.VoteCount = int64Val(props["vote_count"])
	m.Popularity = float64Val(props["popularity"])
	m.Budget = int64Val(props["budget"])
	m.Revenue = int64Val(props["revenue"])
	m.Certification = strVal(props["certification"])
	m.TrailerYouTubeKey = strVal(props["trailer_youtube_key"])
	m.PosterPath = strVal(props["poster_path"])
	m.BackdropPath = strVal(props["backdrop_path"])
	m.OriginalLanguage = strVal(props["original_language"])
	m.CollectionID = int64Val(props["collection_id"])
	m.CollectionName = strVal(props["collection_name"])
	m.Adult = boolVal(props["adult"])
	m.CinovaScore = float64Val(props["cinova_score"])
	m.PlotSummary = strVal(props["plot_summary"])
	m.CinovaSynopsis = strVal(props["cinova_synopsis"])
	m.CinovaEditorial = strVal(props["cinova_editorial"])
	if v := strVal(props["vertical_trailer_youtube_key"]); v != "" && v != "NOT_FOUND" {
		m.VerticalTrailerYouTubeKey = v
	}
	if sl, ok := props["spoken_languages"]; ok && sl != nil {
		if list, ok := sl.([]interface{}); ok {
			for _, item := range list {
				if s, ok := item.(string); ok {
					m.SpokenLanguages = append(m.SpokenLanguages, s)
				}
			}
		}
	}
	return m
}

func tvNodeToModel(v interface{}) *models.TVShow {
	t := &models.TVShow{MediaType: "tv"}
	node, ok := v.(dbtype.Node)
	if !ok {
		return t
	}
	props := node.Props
	t.TMDBID = int64Val(props["tmdb_id"])
	t.WikidataID = strVal(props["wikidata_id"])
	t.Name = strVal(props["name"])
	t.OriginalName = strVal(props["original_name"])
	t.Tagline = strVal(props["tagline"])
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
	t.TrailerYouTubeKey = strVal(props["trailer_youtube_key"])
	t.PlotSummary = strVal(props["plot_summary"])
	t.CinovaSynopsis = strVal(props["cinova_synopsis"])
	t.CinovaEditorial = strVal(props["cinova_editorial"])
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

// toKeywords converts a Neo4j list of keyword maps into []models.Keyword.
func toKeywords(v interface{}) []models.Keyword {
	list, _ := v.([]interface{})
	keywords := make([]models.Keyword, 0, len(list))
	for _, item := range list {
		m, _ := item.(map[string]interface{})
		if m == nil || m["id"] == nil {
			continue
		}
		keywords = append(keywords, models.Keyword{
			ID:   int64Val(m["id"]),
			Name: strVal(m["name"]),
		})
	}
	return keywords
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

// toAwards converts a Neo4j list of award maps into []models.Award.
func toAwards(v interface{}) []models.Award {
	list, _ := v.([]interface{})
	awards := make([]models.Award, 0, len(list))
	for _, item := range list {
		m, _ := item.(map[string]interface{})
		if m == nil || m["wikidata_id"] == nil || strVal(m["wikidata_id"]) == "" {
			continue
		}
		awards = append(awards, models.Award{
			WikidataID:    strVal(m["wikidata_id"]),
			AwardName:     strVal(m["award_name"]),
			CeremonyName:  strVal(m["ceremony_name"]),
			Year:          int(int64Val(m["year"])),
			RecipientName: strVal(m["recipient_name"]),
			IsNomination:  boolVal(m["is_nomination"]),
		})
	}
	return awards
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
			Type:         strVal(m["type"]),
		})
	}
	return providers
}

// UpsertTVShow creates or updates a TVShow node and its relationships.
func (r *MovieRepository) UpsertTVShow(ctx context.Context, show *models.TVShow) error {
	err := r.driver.RunWriteUnit(ctx, `
		MERGE (s:TVShow {tmdb_id: $tmdb_id})
		SET s.name               = $name,
		    s.original_name      = $original_name,
		    s.wikidata_id        = $wikidata_id,
		    s.tagline            = $tagline,
		    s.overview           = $overview,
		    s.first_air_date     = $first_air_date,
		    s.last_air_date      = $last_air_date,
		    s.number_of_seasons  = $number_of_seasons,
		    s.number_of_episodes = $number_of_episodes,
		    s.vote_average       = $vote_average,
		    s.vote_count         = $vote_count,
		    s.popularity         = $popularity,
		    s.poster_path        = $poster_path,
		    s.backdrop_path      = $backdrop_path,
		    s.original_language  = $original_language,
		    s.status             = $status,
		    s.cinova_score       = $cinova_score,
		    s.plot_summary       = CASE WHEN $plot_summary <> '' THEN $plot_summary ELSE coalesce(s.plot_summary, '') END,
		    s.trailer_youtube_key = CASE WHEN $trailer_youtube_key <> '' THEN $trailer_youtube_key ELSE coalesce(s.trailer_youtube_key, '') END,
		    s.cinova_synopsis    = CASE WHEN $cinova_synopsis <> '' THEN $cinova_synopsis ELSE coalesce(s.cinova_synopsis, '') END,
		    s.cinova_editorial   = CASE WHEN $cinova_editorial <> '' THEN $cinova_editorial ELSE coalesce(s.cinova_editorial, '') END
	`, map[string]interface{}{
		"tmdb_id":            show.TMDBID,
		"name":               show.Name,
		"original_name":      show.OriginalName,
		"wikidata_id":        show.WikidataID,
		"tagline":            show.Tagline,
		"overview":           show.Overview,
		"first_air_date":     show.FirstAirDate,
		"last_air_date":      show.LastAirDate,
		"number_of_seasons":  show.NumberOfSeasons,
		"number_of_episodes": show.NumberOfEpisodes,
		"vote_average":       show.VoteAverage,
		"vote_count":         show.VoteCount,
		"popularity":         show.Popularity,
		"poster_path":        show.PosterPath,
		"backdrop_path":      show.BackdropPath,
		"original_language":  show.OriginalLanguage,
		"status":             show.Status,
		"cinova_score":        show.CinovaScore,
		"trailer_youtube_key": show.TrailerYouTubeKey,
		"plot_summary":        show.PlotSummary,
		"cinova_synopsis":     show.CinovaSynopsis,
		"cinova_editorial":    show.CinovaEditorial,
	})
	if err != nil {
		return fmt.Errorf("UpsertTVShow: %w", err)
	}

	for _, g := range show.Genres {
		if err := r.driver.RunWriteUnit(ctx, `
			MERGE (g:Genre {id: $id}) SET g.name = $name
			WITH g
			MATCH (s:TVShow {tmdb_id: $tmdb_id})
			MERGE (s)-[:IN_GENRE]->(g)
		`, map[string]interface{}{"id": g.ID, "name": g.Name, "tmdb_id": show.TMDBID}); err != nil {
			return fmt.Errorf("UpsertTVShow genre %d: %w", g.ID, err)
		}
	}

	for _, p := range show.Cast {
		if err := r.driver.RunWriteUnit(ctx, `
			MERGE (p:Person {tmdb_id: $tmdb_id})
			SET p.name = $name, p.profile_path = $profile_path
			WITH p
			MATCH (s:TVShow {tmdb_id: $show_tmdb_id})
			MERGE (p)-[a:ACTED_IN {show_tmdb_id: $show_tmdb_id}]->(s)
			SET a.character = $role, a.order = $order
		`, map[string]interface{}{
			"tmdb_id":      p.TMDBID,
			"name":         p.Name,
			"profile_path": p.ProfilePath,
			"show_tmdb_id": show.TMDBID,
			"role":         p.Role,
			"order":        p.Order,
		}); err != nil {
			return fmt.Errorf("UpsertTVShow cast person %d: %w", p.TMDBID, err)
		}
	}

	for _, d := range show.Creators {
		if err := r.driver.RunWriteUnit(ctx, `
			MERGE (p:Person {tmdb_id: $tmdb_id})
			SET p.name = $name, p.profile_path = $profile_path
			WITH p
			MATCH (s:TVShow {tmdb_id: $show_tmdb_id})
			MERGE (p)-[dir:DIRECTED {show_tmdb_id: $show_tmdb_id}]->(s)
			SET dir.job = $job
		`, map[string]interface{}{
			"tmdb_id":      d.TMDBID,
			"name":         d.Name,
			"profile_path": d.ProfilePath,
			"show_tmdb_id": show.TMDBID,
			"job":          d.Job,
		}); err != nil {
			return fmt.Errorf("UpsertTVShow director %d: %w", d.TMDBID, err)
		}
	}

	// Upsert providers grouped by country (UpsertProvider handles Movie and TVShow nodes)
	if len(show.Providers) > 0 {
		byCountry := make(map[string][]models.Provider)
		for _, p := range show.Providers {
			byCountry[p.Country] = append(byCountry[p.Country], p)
		}
		for country, provs := range byCountry {
			if err := r.UpsertProvider(ctx, int(show.TMDBID), provs, country); err != nil {
				return fmt.Errorf("UpsertTVShow providers: %w", err)
			}
		}
	}

	return nil
}

// UpsertMovieAward creates or updates an Award node and attaches a HAS_WON (or HAS_NOMINATION)
// relationship from the Movie to the Award. Uses wikidata_id as the unique key.
func (r *MovieRepository) UpsertMovieAward(ctx context.Context, tmdbID int64, award models.Award) error {
	relType := "HAS_WON"
	if award.IsNomination {
		relType = "HAS_NOMINATION"
	}
	cypher := `
		MERGE (a:Award {wikidata_id: $wikidata_id})
		SET a.award_name    = $award_name,
		    a.ceremony_name = $ceremony_name,
		    a.year          = $year,
		    a.category      = $category
		WITH a
		MATCH (m:Movie {tmdb_id: $tmdb_id})
		MERGE (m)-[r:` + relType + ` {wikidata_id: $wikidata_id}]->(a)
		SET r.recipient_name = $recipient_name,
		    r.year           = $year
	`
	return r.driver.RunWriteUnit(ctx, cypher, map[string]interface{}{
		"wikidata_id":    award.WikidataID,
		"award_name":     award.AwardName,
		"ceremony_name":  award.CeremonyName,
		"year":           award.Year,
		"category":       award.Category,
		"recipient_name": award.RecipientName,
		"tmdb_id":        tmdbID,
	})
}

// UpdateMovieEnrichmentText writes Claude-generated cinova_synopsis and cinova_editorial
// back to a Movie node. plot_summary is written during Phase 1 ingest and is NOT touched here.
func (r *MovieRepository) UpdateMovieEnrichmentText(ctx context.Context, tmdbID int64, cinovaSynopsis, cinovaEditorial string) error {
	return r.driver.RunWriteUnit(ctx, `
		MATCH (m:Movie {tmdb_id: $tmdb_id})
		SET m.cinova_synopsis  = $cinova_synopsis,
		    m.cinova_editorial = $cinova_editorial
	`, map[string]interface{}{
		"tmdb_id":          tmdbID,
		"cinova_synopsis":  cinovaSynopsis,
		"cinova_editorial": cinovaEditorial,
	})
}

// UpdateTVShowEnrichmentText writes Claude-generated cinova_synopsis and cinova_editorial
// back to a TVShow node.
func (r *MovieRepository) UpdateTVShowEnrichmentText(ctx context.Context, tmdbID int64, cinovaSynopsis, cinovaEditorial string) error {
	return r.driver.RunWriteUnit(ctx, `
		MATCH (s:TVShow {tmdb_id: $tmdb_id})
		SET s.cinova_synopsis  = $cinova_synopsis,
		    s.cinova_editorial = $cinova_editorial
	`, map[string]interface{}{
		"tmdb_id":          tmdbID,
		"cinova_synopsis":  cinovaSynopsis,
		"cinova_editorial": cinovaEditorial,
	})
}

// UpsertInfluenceRelationship creates an INFLUENCED_BY relationship between two Person nodes by name.
func (r *MovieRepository) UpsertInfluenceRelationship(ctx context.Context, directorName, influencerName string) error {
	return r.driver.RunWriteUnit(ctx, `
		MERGE (d:Person {name: $director_name})
		MERGE (i:Person {name: $influencer_name})
		MERGE (d)-[r:INFLUENCED_BY]->(i)
		SET r.source = 'wikidata'
	`, map[string]interface{}{
		"director_name":  directorName,
		"influencer_name": influencerName,
	})
}

// UpsertTVShowAward creates or updates an Award node and attaches a HAS_WON or
// HAS_NOMINATION relationship from a TVShow node.
func (r *MovieRepository) UpsertTVShowAward(ctx context.Context, tmdbID int64, award models.Award) error {
	relType := "HAS_WON"
	if award.IsNomination {
		relType = "HAS_NOMINATION"
	}
	cypher := `
		MERGE (a:Award {wikidata_id: $wikidata_id})
		SET a.award_name    = $award_name,
		    a.ceremony_name = $ceremony_name,
		    a.year          = $year,
		    a.category      = $category
		WITH a
		MATCH (s:TVShow {tmdb_id: $tmdb_id})
		MERGE (s)-[r:` + relType + ` {wikidata_id: $wikidata_id}]->(a)
		SET r.recipient_name = $recipient_name,
		    r.year           = $year
	`
	return r.driver.RunWriteUnit(ctx, cypher, map[string]interface{}{
		"wikidata_id":    award.WikidataID,
		"award_name":     award.AwardName,
		"ceremony_name":  award.CeremonyName,
		"year":           award.Year,
		"category":       award.Category,
		"recipient_name": award.RecipientName,
		"tmdb_id":        tmdbID,
	})
}

// UpsertScoringProfile stores a user's scoring preset on the User node in Neo4j.
func (r *MovieRepository) UpsertScoringProfile(ctx context.Context, userID string, profile models.ScoringProfile) error {
	return r.driver.RunWriteUnit(ctx, `
		MERGE (u:User {id: $user_id})
		SET u.scoring_preset      = $preset,
		    u.scoring_audience    = $audience,
		    u.scoring_critic      = $critic,
		    u.scoring_award       = $award,
		    u.scoring_prestige    = $prestige,
		    u.scoring_commercial  = $commercial
	`, map[string]interface{}{
		"user_id":    userID,
		"preset":     profile.Preset,
		"audience":   profile.Audience,
		"critic":     profile.Critic,
		"award":      profile.Award,
		"prestige":   profile.Prestige,
		"commercial": profile.Commercial,
	})
}

// GetScoringProfile returns a user's stored scoring profile, or the default if not set.
func (r *MovieRepository) GetScoringProfile(ctx context.Context, userID string) (models.ScoringProfile, error) {
	records, err := r.driver.RunQuery(ctx, `
		MATCH (u:User {id: $user_id})
		RETURN u.scoring_preset     AS preset,
		       u.scoring_audience   AS audience,
		       u.scoring_critic     AS critic,
		       u.scoring_award      AS award,
		       u.scoring_prestige   AS prestige,
		       u.scoring_commercial AS commercial
	`, map[string]interface{}{"user_id": userID})
	if err != nil || len(records) == 0 {
		return models.ScoringProfile{Preset: "mainstream"}, nil
	}
	rec := records[0]
	p := models.ScoringProfile{}
	if v, ok := rec.Get("preset"); ok {
		p.Preset = strVal(v)
	}
	if v, ok := rec.Get("audience"); ok {
		p.Audience = float64Val(v)
	}
	if v, ok := rec.Get("critic"); ok {
		p.Critic = float64Val(v)
	}
	if v, ok := rec.Get("award"); ok {
		p.Award = float64Val(v)
	}
	if v, ok := rec.Get("prestige"); ok {
		p.Prestige = float64Val(v)
	}
	if v, ok := rec.Get("commercial"); ok {
		p.Commercial = float64Val(v)
	}
	if p.Preset == "" {
		p.Preset = "mainstream"
	}
	return p, nil
}

// ComputeAndUpdatePageRank runs a simplified PageRank-inspired computation using
// Cypher without the GDS plugin. Uses the INFLUENCED_BY + DIRECTED graph to
// compute a normalised prestige score per Movie/TVShow node.
//
// Strategy:
//   - Score a movie/show by summing the "importance" of its directors
//   - A director's importance = 1 + count of INFLUENCED_BY relationships they appear in
//   - Normalise to [0, 1] range across all movies
//   - Write cinova_score using ComputeCinovaScore(vote_average, vote_count, normalisedPrestige)
func (r *MovieRepository) ComputeAndUpdatePageRank(ctx context.Context) error {
	// Step 1: compute director prestige (in-degree on INFLUENCED_BY)
	_, err := r.driver.RunWrite(ctx, `
		MATCH (p:Person)
		OPTIONAL MATCH (other:Person)-[:INFLUENCED_BY]->(p)
		WITH p, count(other) AS influence_count
		SET p.prestige = toFloat(1 + influence_count)
	`, nil)
	if err != nil {
		return fmt.Errorf("ComputePageRank prestige: %w", err)
	}

	// Step 2: compute raw graph signal per Movie as sum of director prestige
	_, err = r.driver.RunWrite(ctx, `
		MATCH (m:Movie)
		OPTIONAL MATCH (d:Person)-[:DIRECTED]->(m)
		WITH m, coalesce(sum(d.prestige), 0.0) AS raw_signal
		SET m.raw_graph_signal = raw_signal
	`, nil)
	if err != nil {
		return fmt.Errorf("ComputePageRank movie signal: %w", err)
	}

	// Also for TV shows
	_, err = r.driver.RunWrite(ctx, `
		MATCH (s:TVShow)
		OPTIONAL MATCH (d:Person)-[:DIRECTED]->(s)
		WITH s, coalesce(sum(d.prestige), 0.0) AS raw_signal
		SET s.raw_graph_signal = raw_signal
	`, nil)
	if err != nil {
		return fmt.Errorf("ComputePageRank tvshow signal: %w", err)
	}

	// Step 3: get max raw signal for normalisation
	records, err := r.driver.RunQuery(ctx, `
		MATCH (n) WHERE n:Movie OR n:TVShow
		RETURN max(n.raw_graph_signal) AS max_signal
	`, nil)
	if err != nil || len(records) == 0 {
		return fmt.Errorf("ComputePageRank max query: %w", err)
	}

	maxSignalVal, _ := records[0].Get("max_signal")
	maxSignal := float64Val(maxSignalVal)
	if maxSignal <= 0 {
		// No influence data yet — skip prestige update
		return nil
	}

	// Step 4: rewrite cinova_score using actual award graph data + prestige signal.
	// award_sig rules:
	//   - wikidata_id not set (not yet checked): neutral 0.50
	//   - wikidata checked, wins found: 0.20 + wins*0.05 capped at 0.85
	//   - wikidata checked, nominations only: 0.15 + noms*0.03 capped at 0.60
	//   - wikidata checked, no awards found: 0.30 (slightly below neutral — known no awards)
	_, err = r.driver.RunWrite(ctx, `
		MATCH (n) WHERE n:Movie OR n:TVShow
		WITH n,
		     CASE WHEN $max_signal > 0 THEN n.raw_graph_signal / $max_signal ELSE 0.0 END AS norm_prestige,
		     (toFloat(n.vote_count) * n.vote_average + 1000.0 * 6.5) / (toFloat(n.vote_count) + 1000.0) / 10.0 AS audience,
		     size([(n)-[:HAS_WON]->(:Award) | 1]) AS wins,
		     size([(n)-[:HAS_NOMINATION]->(:Award) | 1]) AS noms
		WITH n, norm_prestige, audience, wins, noms,
		     CASE
		       WHEN (n.wikidata_id IS NULL OR n.wikidata_id = '') THEN 0.50
		       WHEN wins > 0 THEN least(0.20 + toFloat(wins) * 0.05, 0.85)
		       WHEN noms > 0 THEN least(0.15 + toFloat(noms) * 0.03, 0.60)
		       ELSE 0.30
		     END AS award_sig
		SET n.cinova_score = round((audience * 0.65 + award_sig * 0.20 + norm_prestige * 0.10 + 0.5 * 0.05) * 100, 1)
	`, map[string]interface{}{"max_signal": maxSignal})
	if err != nil {
		return fmt.Errorf("ComputePageRank score update: %w", err)
	}

	return nil
}

// ScoreRecomputeItem holds the raw signals needed to recompute a title's CinovaScore
// using the Go scoring package (proper AwardTier string-matching, etc.).
type ScoreRecomputeItem struct {
	TMDBID         int64
	VoteAverage    float64
	VoteCount      int
	RawGraphSignal float64
	WikidataID     string
	Budget         int64
	Revenue        int64
	Awards         []models.Award
}

// GetMaxRawGraphSignal returns the maximum raw_graph_signal across all Movie and TVShow nodes.
func (r *MovieRepository) GetMaxRawGraphSignal(ctx context.Context) (float64, error) {
	records, err := r.driver.RunQuery(ctx, `
		MATCH (n) WHERE n:Movie OR n:TVShow
		RETURN max(n.raw_graph_signal) AS max_signal
	`, nil)
	if err != nil || len(records) == 0 {
		return 0, fmt.Errorf("GetMaxRawGraphSignal: %w", err)
	}
	v, _ := records[0].Get("max_signal")
	return float64Val(v), nil
}

// GetMoviesForScoreRecompute returns a paginated batch of movies with their award data
// for Go-side score recomputation using the full scoring package.
func (r *MovieRepository) GetMoviesForScoreRecompute(ctx context.Context, skip, limit int) ([]ScoreRecomputeItem, error) {
	records, err := r.driver.RunQuery(ctx, `
		MATCH (m:Movie)
		WITH m ORDER BY m.tmdb_id ASC SKIP $skip LIMIT $limit
		OPTIONAL MATCH (m)-[rel:HAS_WON|HAS_NOMINATION]->(aw:Award)
		WITH m, collect({
		     award_name: aw.award_name,
		     ceremony_name: aw.ceremony_name,
		     year: rel.year,
		     recipient_name: rel.recipient_name,
		     is_nomination: (type(rel) = 'HAS_NOMINATION'),
		     wikidata_id: aw.wikidata_id
		}) AS awards
		RETURN m.tmdb_id          AS tmdb_id,
		       m.vote_average     AS vote_average,
		       m.vote_count       AS vote_count,
		       m.raw_graph_signal AS raw_graph_signal,
		       m.wikidata_id      AS wikidata_id,
		       m.budget           AS budget,
		       m.revenue          AS revenue,
		       awards
	`, map[string]interface{}{"skip": skip, "limit": limit})
	if err != nil {
		return nil, fmt.Errorf("GetMoviesForScoreRecompute: %w", err)
	}
	items := make([]ScoreRecomputeItem, 0, len(records))
	for _, rec := range records {
		item := ScoreRecomputeItem{}
		if v, ok := rec.Get("tmdb_id"); ok {
			item.TMDBID = int64Val(v)
		}
		if v, ok := rec.Get("vote_average"); ok {
			item.VoteAverage = float64Val(v)
		}
		if v, ok := rec.Get("vote_count"); ok {
			item.VoteCount = int(int64Val(v))
		}
		if v, ok := rec.Get("raw_graph_signal"); ok {
			item.RawGraphSignal = float64Val(v)
		}
		if v, ok := rec.Get("wikidata_id"); ok {
			item.WikidataID = strVal(v)
		}
		if v, ok := rec.Get("budget"); ok {
			item.Budget = int64Val(v)
		}
		if v, ok := rec.Get("revenue"); ok {
			item.Revenue = int64Val(v)
		}
		if v, ok := rec.Get("awards"); ok {
			item.Awards = toAwards(v)
		}
		items = append(items, item)
	}
	return items, nil
}

// GetTVShowsForScoreRecompute returns a paginated batch of TV shows for score recomputation.
func (r *MovieRepository) GetTVShowsForScoreRecompute(ctx context.Context, skip, limit int) ([]ScoreRecomputeItem, error) {
	records, err := r.driver.RunQuery(ctx, `
		MATCH (s:TVShow)
		WITH s ORDER BY s.tmdb_id ASC SKIP $skip LIMIT $limit
		OPTIONAL MATCH (s)-[rel:HAS_WON|HAS_NOMINATION]->(aw:Award)
		WITH s, collect({
		     award_name: aw.award_name,
		     ceremony_name: aw.ceremony_name,
		     year: rel.year,
		     recipient_name: rel.recipient_name,
		     is_nomination: (type(rel) = 'HAS_NOMINATION'),
		     wikidata_id: aw.wikidata_id
		}) AS awards
		RETURN s.tmdb_id          AS tmdb_id,
		       s.vote_average     AS vote_average,
		       s.vote_count       AS vote_count,
		       s.raw_graph_signal AS raw_graph_signal,
		       s.wikidata_id      AS wikidata_id,
		       awards
	`, map[string]interface{}{"skip": skip, "limit": limit})
	if err != nil {
		return nil, fmt.Errorf("GetTVShowsForScoreRecompute: %w", err)
	}
	items := make([]ScoreRecomputeItem, 0, len(records))
	for _, rec := range records {
		item := ScoreRecomputeItem{}
		if v, ok := rec.Get("tmdb_id"); ok {
			item.TMDBID = int64Val(v)
		}
		if v, ok := rec.Get("vote_average"); ok {
			item.VoteAverage = float64Val(v)
		}
		if v, ok := rec.Get("vote_count"); ok {
			item.VoteCount = int(int64Val(v))
		}
		if v, ok := rec.Get("raw_graph_signal"); ok {
			item.RawGraphSignal = float64Val(v)
		}
		if v, ok := rec.Get("wikidata_id"); ok {
			item.WikidataID = strVal(v)
		}
		if v, ok := rec.Get("awards"); ok {
			item.Awards = toAwards(v)
		}
		items = append(items, item)
	}
	return items, nil
}

// UpdateNodeScore writes a new cinova_score to a Movie or TVShow node identified by tmdb_id.
func (r *MovieRepository) UpdateNodeScore(ctx context.Context, label string, tmdbID int64, score float64) error {
	cypher := fmt.Sprintf(`MATCH (n:%s {tmdb_id: $tmdb_id}) SET n.cinova_score = $score`, label)
	return r.driver.RunWriteUnit(ctx, cypher, map[string]interface{}{
		"tmdb_id": tmdbID,
		"score":   score,
	})
}

// GetMoviesWithoutSynopsis returns up to limit Movie nodes that lack a cinova_synopsis,
// have a non-empty overview (so they can be enriched), ordered by popularity desc.
func (r *MovieRepository) GetMoviesWithoutSynopsis(ctx context.Context, limit int) ([]models.Movie, error) {
	records, err := r.driver.RunQuery(ctx, `
		MATCH (m:Movie)
		WHERE (m.cinova_synopsis IS NULL OR m.cinova_synopsis = '')
		  AND m.overview IS NOT NULL AND m.overview <> ''
		RETURN m
		ORDER BY m.popularity DESC
		LIMIT $limit
	`, map[string]interface{}{"limit": limit})
	if err != nil {
		return nil, fmt.Errorf("GetMoviesWithoutSynopsis: %w", err)
	}
	movies := make([]models.Movie, 0, len(records))
	for _, rec := range records {
		raw, _ := rec.Get("m")
		if m := movieNodeToModel(raw); m != nil {
			movies = append(movies, *m)
		}
	}
	return movies, nil
}

// QualityReport is returned by SampleQuality and summarises field completeness.
type QualityReport struct {
	Sampled        int
	MissingTitle   int
	MissingOverview int
	ZeroCinovaScore int
	MissingProviders int
	MissingPlot     int // has wikidata_id but no plot_summary
	MissingSynopsis  int
	MissingEditorial int
	MissingThemes   int
	MissingMoods    int
	// Computed
	PassRate float64 // 0–1
}

// SampleQuality pulls the N most-recently-upserted Movie nodes and checks field
// completeness. It counts failures per field and computes an overall pass rate.
// A "failure" is any field that should be populated but is empty/zero.
func (r *MovieRepository) SampleQuality(ctx context.Context, n int) (*QualityReport, error) {
	records, err := r.driver.RunQuery(ctx, `
		MATCH (m:Movie)
		WITH m ORDER BY m.tmdb_id DESC LIMIT $n
		OPTIONAL MATCH (m)-[:HAS_THEME]->(th:Theme)
		OPTIONAL MATCH (m)-[:HAS_MOOD]->(mo:Mood)
		OPTIONAL MATCH (m)-[:AVAILABLE_ON {country: 'US'}]->(prov:Provider)
		RETURN m,
		       count(DISTINCT th) AS theme_count,
		       count(DISTINCT mo) AS mood_count,
		       count(DISTINCT prov) AS provider_count
	`, map[string]interface{}{"n": n})
	if err != nil {
		return nil, fmt.Errorf("SampleQuality: %w", err)
	}

	rep := &QualityReport{Sampled: len(records)}
	failures := 0
	checks := 0

	for _, rec := range records {
		raw, _ := rec.Get("m")
		m := movieNodeToModel(raw)
		themeCount := int(int64Val(func() interface{} { v, _ := rec.Get("theme_count"); return v }()))
		moodCount := int(int64Val(func() interface{} { v, _ := rec.Get("mood_count"); return v }()))
		providerCount := int(int64Val(func() interface{} { v, _ := rec.Get("provider_count"); return v }()))

		checks += 8 // fields checked per movie

		if m.Title == "" { rep.MissingTitle++; failures++ }
		if m.Overview == "" { rep.MissingOverview++; failures++ }
		if m.CinovaScore == 0 { rep.ZeroCinovaScore++; failures++ }
		if providerCount == 0 { rep.MissingProviders++; failures++ }
		if m.WikidataID != "" && m.PlotSummary == "" { rep.MissingPlot++; failures++ }
		if m.CinovaSynopsis == "" { rep.MissingSynopsis++; failures++ }
		if m.CinovaEditorial == "" { rep.MissingEditorial++; failures++ }
		if themeCount == 0 { rep.MissingThemes++; failures++ }
		if moodCount == 0 { rep.MissingMoods++; failures++ }
	}

	if checks > 0 {
		rep.PassRate = 1.0 - float64(failures)/float64(checks)
	}
	return rep, nil
}

// AssessmentReport holds quality metrics for fully-enriched nodes.
// It only covers nodes where cinova_synopsis is populated (phase 2 complete).
type AssessmentReport struct {
	// Counts
	TotalEnriched   int
	Sampled         int

	// Phase-1 fields (should be ~100% for enriched nodes)
	MissingTitle     int
	MissingOverview  int
	ZeroCinovaScore  int
	MissingProviders int
	MissingPlot      int // has wikidata_id but no plot_summary

	// Phase-2 fields (these are the filter condition, so synopsis always 100%)
	MissingEditorial int
	MissingThemes    int
	MissingMoods     int

	// Content quality
	EditorialTooShort int // < 150 words
	EditorialTooLong  int // > 250 words
	SynopsisTooShort  int // < 3 sentences

	// Per-field pass rates (0–1)
	PassRatePhase1   float64
	PassRatePhase2   float64
	PassRateContent  float64
	PassRateOverall  float64
}

// AssessEnrichedMovies samples up to n fully-enriched Movie nodes (cinova_synopsis populated)
// and returns a detailed quality report covering phase-1 completeness, phase-2 completeness,
// and content quality (editorial word count, synopsis sentence count).
func (r *MovieRepository) AssessEnrichedMovies(ctx context.Context, n int) (*AssessmentReport, error) {
	// Step 1: total count of enriched movies
	countRecs, err := r.driver.RunQuery(ctx, `
		MATCH (m:Movie)
		WHERE m.cinova_synopsis IS NOT NULL AND m.cinova_synopsis <> ''
		RETURN count(m) AS total
	`, nil)
	if err != nil {
		return nil, fmt.Errorf("AssessEnrichedMovies count: %w", err)
	}
	rep := &AssessmentReport{}
	if len(countRecs) > 0 {
		rep.TotalEnriched = int(int64Val(func() interface{} { v, _ := countRecs[0].Get("total"); return v }()))
	}
	if rep.TotalEnriched == 0 {
		return rep, nil
	}

	// Step 2: sample enriched nodes (random order via rand())
	records, err := r.driver.RunQuery(ctx, `
		MATCH (m:Movie)
		WHERE m.cinova_synopsis IS NOT NULL AND m.cinova_synopsis <> ''
		WITH m ORDER BY rand() LIMIT $n
		OPTIONAL MATCH (m)-[:HAS_THEME]->(th:Theme)
		OPTIONAL MATCH (m)-[:HAS_MOOD]->(mo:Mood)
		OPTIONAL MATCH (m)-[:AVAILABLE_ON {country: 'US'}]->(prov:Provider)
		RETURN m,
		       count(DISTINCT th) AS theme_count,
		       count(DISTINCT mo) AS mood_count,
		       count(DISTINCT prov) AS provider_count
	`, map[string]interface{}{"n": n})
	if err != nil {
		return nil, fmt.Errorf("AssessEnrichedMovies sample: %w", err)
	}

	rep.Sampled = len(records)
	p1Checks, p1Failures := 0, 0
	p2Checks, p2Failures := 0, 0
	cqChecks, cqFailures := 0, 0

	for _, rec := range records {
		raw, _ := rec.Get("m")
		m := movieNodeToModel(raw)
		themeCount   := int(int64Val(func() interface{} { v, _ := rec.Get("theme_count"); return v }()))
		moodCount    := int(int64Val(func() interface{} { v, _ := rec.Get("mood_count"); return v }()))
		providerCount := int(int64Val(func() interface{} { v, _ := rec.Get("provider_count"); return v }()))

		// Phase-1 checks: basic metadata
		p1Checks += 4
		if m.Title == ""       { rep.MissingTitle++;    p1Failures++ }
		if m.Overview == ""    { rep.MissingOverview++; p1Failures++ }
		if m.CinovaScore == 0  { rep.ZeroCinovaScore++; p1Failures++ }
		if providerCount == 0  { rep.MissingProviders++; p1Failures++ }
		if m.WikidataID != "" && m.PlotSummary == "" {
			rep.MissingPlot++; p1Failures++; p1Checks++
		}

		// Phase-2 checks: enrichment fields
		p2Checks += 3
		if m.CinovaEditorial == "" { rep.MissingEditorial++; p2Failures++ }
		if themeCount == 0         { rep.MissingThemes++;    p2Failures++ }
		if moodCount == 0          { rep.MissingMoods++;     p2Failures++ }

		// Content quality: editorial word count (target 150-250 words)
		if m.CinovaEditorial != "" {
			cqChecks++
			wordCount := len(strings.Fields(m.CinovaEditorial))
			if wordCount < 150 { rep.EditorialTooShort++; cqFailures++ }
			if wordCount > 250 { rep.EditorialTooLong++;  cqFailures++ }
		}
		// Content quality: synopsis sentence count (target 3-4 sentences)
		if m.CinovaSynopsis != "" {
			cqChecks++
			sentenceCount := len(strings.Split(strings.TrimSpace(m.CinovaSynopsis), "."))
			if sentenceCount < 3 { rep.SynopsisTooShort++; cqFailures++ }
		}
	}

	if p1Checks > 0 { rep.PassRatePhase1 = 1.0 - float64(p1Failures)/float64(p1Checks) }
	if p2Checks > 0 { rep.PassRatePhase2 = 1.0 - float64(p2Failures)/float64(p2Checks) }
	if cqChecks > 0 { rep.PassRateContent = 1.0 - float64(cqFailures)/float64(cqChecks) }
	totalC := p1Checks + p2Checks + cqChecks
	totalF := p1Failures + p2Failures + cqFailures
	if totalC > 0 { rep.PassRateOverall = 1.0 - float64(totalF)/float64(totalC) }

	return rep, nil
}

// AssessEnrichedTVShows samples up to n fully-enriched TVShow nodes and returns
// the same quality report structure as AssessEnrichedMovies.
func (r *MovieRepository) AssessEnrichedTVShows(ctx context.Context, n int) (*AssessmentReport, error) {
	countRecs, err := r.driver.RunQuery(ctx, `
		MATCH (s:TVShow)
		WHERE s.cinova_synopsis IS NOT NULL AND s.cinova_synopsis <> ''
		RETURN count(s) AS total
	`, nil)
	if err != nil {
		return nil, fmt.Errorf("AssessEnrichedTVShows count: %w", err)
	}
	rep := &AssessmentReport{}
	if len(countRecs) > 0 {
		rep.TotalEnriched = int(int64Val(func() interface{} { v, _ := countRecs[0].Get("total"); return v }()))
	}
	if rep.TotalEnriched == 0 {
		return rep, nil
	}

	records, err := r.driver.RunQuery(ctx, `
		MATCH (s:TVShow)
		WHERE s.cinova_synopsis IS NOT NULL AND s.cinova_synopsis <> ''
		WITH s ORDER BY rand() LIMIT $n
		OPTIONAL MATCH (s)-[:HAS_THEME]->(th:Theme)
		OPTIONAL MATCH (s)-[:HAS_MOOD]->(mo:Mood)
		OPTIONAL MATCH (s)-[:AVAILABLE_ON {country: 'US'}]->(prov:Provider)
		RETURN s,
		       count(DISTINCT th) AS theme_count,
		       count(DISTINCT mo) AS mood_count,
		       count(DISTINCT prov) AS provider_count
	`, map[string]interface{}{"n": n})
	if err != nil {
		return nil, fmt.Errorf("AssessEnrichedTVShows sample: %w", err)
	}

	rep.Sampled = len(records)
	p1Checks, p1Failures := 0, 0
	p2Checks, p2Failures := 0, 0
	cqChecks, cqFailures := 0, 0

	for _, rec := range records {
		raw, _ := rec.Get("s")
		show := tvNodeToModel(raw)
		themeCount    := int(int64Val(func() interface{} { v, _ := rec.Get("theme_count"); return v }()))
		moodCount     := int(int64Val(func() interface{} { v, _ := rec.Get("mood_count"); return v }()))
		providerCount := int(int64Val(func() interface{} { v, _ := rec.Get("provider_count"); return v }()))

		p1Checks += 4
		if show.Name == ""       { rep.MissingTitle++;    p1Failures++ }
		if show.Overview == ""   { rep.MissingOverview++; p1Failures++ }
		if show.CinovaScore == 0 { rep.ZeroCinovaScore++; p1Failures++ }
		if providerCount == 0    { rep.MissingProviders++; p1Failures++ }
		if show.WikidataID != "" && show.PlotSummary == "" {
			rep.MissingPlot++; p1Failures++; p1Checks++
		}

		p2Checks += 3
		if show.CinovaEditorial == "" { rep.MissingEditorial++; p2Failures++ }
		if themeCount == 0            { rep.MissingThemes++;    p2Failures++ }
		if moodCount == 0             { rep.MissingMoods++;     p2Failures++ }

		if show.CinovaEditorial != "" {
			cqChecks++
			wordCount := len(strings.Fields(show.CinovaEditorial))
			if wordCount < 150 { rep.EditorialTooShort++; cqFailures++ }
			if wordCount > 250 { rep.EditorialTooLong++;  cqFailures++ }
		}
		if show.CinovaSynopsis != "" {
			cqChecks++
			sentenceCount := len(strings.Split(strings.TrimSpace(show.CinovaSynopsis), "."))
			if sentenceCount < 3 { rep.SynopsisTooShort++; cqFailures++ }
		}
	}

	if p1Checks > 0 { rep.PassRatePhase1 = 1.0 - float64(p1Failures)/float64(p1Checks) }
	if p2Checks > 0 { rep.PassRatePhase2 = 1.0 - float64(p2Failures)/float64(p2Checks) }
	if cqChecks > 0 { rep.PassRateContent = 1.0 - float64(cqFailures)/float64(cqChecks) }
	totalC := p1Checks + p2Checks + cqChecks
	totalF := p1Failures + p2Failures + cqFailures
	if totalC > 0 { rep.PassRateOverall = 1.0 - float64(totalF)/float64(totalC) }

	return rep, nil
}

// GetMoviesWithoutVerticalTrailer returns movies that have a trailer_youtube_key
// but no vertical_trailer_youtube_key, ordered by release_year DESC then popularity DESC.
// This prioritises recent movies since vertical trailers are far more common post-2019.
func (r *MovieRepository) GetMoviesWithoutVerticalTrailer(ctx context.Context, limit int) ([]models.Movie, error) {
	records, err := r.driver.RunQuery(ctx, `
		MATCH (m:Movie)
		WHERE m.trailer_youtube_key IS NOT NULL
		  AND m.trailer_youtube_key <> ''
		  AND (m.vertical_trailer_youtube_key IS NULL OR m.vertical_trailer_youtube_key = '')
		RETURN m
		ORDER BY m.release_year DESC, m.popularity DESC
		LIMIT $limit
	`, map[string]interface{}{"limit": limit})
	if err != nil {
		return nil, fmt.Errorf("GetMoviesWithoutVerticalTrailer: %w", err)
	}
	movies := make([]models.Movie, 0, len(records))
	for _, rec := range records {
		raw, _ := rec.Get("m")
		if m := movieNodeToModel(raw); m != nil {
			movies = append(movies, *m)
		}
	}
	return movies, nil
}

// SetVerticalTrailerKey stores a vertical_trailer_youtube_key on a Movie node.
func (r *MovieRepository) SetVerticalTrailerKey(ctx context.Context, tmdbID int, verticalKey string) error {
	return r.driver.RunWriteUnit(ctx, `
		MATCH (m:Movie {tmdb_id: $tmdb_id})
		SET m.vertical_trailer_youtube_key = $key
	`, map[string]interface{}{"tmdb_id": tmdbID, "key": verticalKey})
}

// GetMoviesWithoutPlot returns up to limit Movie nodes that have a wikidata_id
// but no plot_summary (excluding the __no_plot__ sentinel), ordered by popularity desc.
func (r *MovieRepository) GetMoviesWithoutPlot(ctx context.Context, limit int) ([]models.Movie, error) {
	records, err := r.driver.RunQuery(ctx, `
		MATCH (m:Movie)
		WHERE m.wikidata_id IS NOT NULL AND m.wikidata_id <> ''
		  AND (m.plot_summary IS NULL OR m.plot_summary = '')
		RETURN m
		ORDER BY m.popularity DESC
		LIMIT $limit
	`, map[string]interface{}{"limit": limit})
	if err != nil {
		return nil, fmt.Errorf("GetMoviesWithoutPlot: %w", err)
	}
	movies := make([]models.Movie, 0, len(records))
	for _, rec := range records {
		raw, _ := rec.Get("m")
		if m := movieNodeToModel(raw); m != nil {
			movies = append(movies, *m)
		}
	}
	return movies, nil
}

// UpdateMoviePlotSummary writes a Wikipedia plot summary back to a Movie node.
func (r *MovieRepository) UpdateMoviePlotSummary(ctx context.Context, tmdbID int64, plotSummary string) error {
	return r.driver.RunWriteUnit(ctx, `
		MATCH (m:Movie {tmdb_id: $tmdb_id})
		SET m.plot_summary = $plot_summary
	`, map[string]interface{}{
		"tmdb_id":      tmdbID,
		"plot_summary": plotSummary,
	})
}

// GetTVShowsWithoutSynopsis returns up to limit TVShow nodes that lack a cinova_synopsis.
func (r *MovieRepository) GetTVShowsWithoutSynopsis(ctx context.Context, limit int) ([]models.TVShow, error) {
	records, err := r.driver.RunQuery(ctx, `
		MATCH (s:TVShow)
		WHERE (s.cinova_synopsis IS NULL OR s.cinova_synopsis = '')
		  AND s.overview IS NOT NULL AND s.overview <> ''
		RETURN s
		ORDER BY s.popularity DESC
		LIMIT $limit
	`, map[string]interface{}{"limit": limit})
	if err != nil {
		return nil, fmt.Errorf("GetTVShowsWithoutSynopsis: %w", err)
	}
	shows := make([]models.TVShow, 0, len(records))
	for _, rec := range records {
		raw, _ := rec.Get("s")
		if s := tvNodeToModel(raw); s != nil && s.TMDBID > 0 {
			shows = append(shows, *s)
		}
	}
	return shows, nil
}
