package tmdb

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/dynolabs-io/cinova/backend/internal/models"
)

const (
	baseURL    = "https://api.themoviedb.org/3"
	exportsURL = "https://files.tmdb.org/p/exports"
	// TMDB rate limit: 40 requests per 10 seconds
	rateLimitDelay = 250 * time.Millisecond
)

// Client is a TMDB API v3 HTTP client.
type Client struct {
	apiKey     string
	httpClient *http.Client
}

// NewClient creates a TMDB client with the given API key.
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ---- TMDB response structs ----

// TMDBMovie is the full movie details response from TMDB.
type TMDBMovie struct {
	ID                  int                   `json:"id"`
	IMDbID              string                `json:"imdb_id"`
	Title               string                `json:"title"`
	OriginalTitle       string                `json:"original_title"`
	Tagline             string                `json:"tagline"`
	Overview            string                `json:"overview"`
	ReleaseDate         string                `json:"release_date"`
	Runtime             int                   `json:"runtime"`
	VoteAverage         float64               `json:"vote_average"`
	VoteCount           int64                 `json:"vote_count"`
	Popularity          float64               `json:"popularity"`
	Budget              int64                 `json:"budget"`
	Revenue             int64                 `json:"revenue"`
	PosterPath          string                `json:"poster_path"`
	BackdropPath        string                `json:"backdrop_path"`
	OriginalLanguage    string                `json:"original_language"`
	SpokenLanguages     []tmdbLanguage        `json:"spoken_languages"`
	BelongsToCollection *tmdbCollection       `json:"belongs_to_collection,omitempty"`
	Adult               bool                  `json:"adult"`
	Genres              []tmdbGenre           `json:"genres"`
	Credits             *tmdbCredits          `json:"credits,omitempty"`
	Keywords            *tmdbKeywords         `json:"keywords,omitempty"`
	ReleaseDates        *tmdbReleaseDatesRoot `json:"release_dates,omitempty"`
	Videos              *tmdbVideosRoot       `json:"videos,omitempty"`
	WatchProviders      *tmdbWatchRoot        `json:"watch/providers,omitempty"`
	Similar             *tmdbMoviePage        `json:"similar,omitempty"`
	ExternalIDs         *tmdbExternalIDs      `json:"external_ids,omitempty"`
}

// ToModel converts a TMDBMovie to a models.Movie.
func (t *TMDBMovie) ToModel() *models.Movie {
	wikidataID := ""
	if t.ExternalIDs != nil {
		wikidataID = t.ExternalIDs.WikidataID
	}

	m := &models.Movie{
		TMDBID:           int64(t.ID),
		IMDbID:           t.IMDbID,
		WikidataID:       wikidataID,
		Title:            t.Title,
		OriginalTitle:    t.OriginalTitle,
		Tagline:          t.Tagline,
		Overview:         t.Overview,
		ReleaseDate:      t.ReleaseDate,
		Runtime:          t.Runtime,
		VoteAverage:      t.VoteAverage,
		VoteCount:        t.VoteCount,
		Popularity:       t.Popularity,
		Budget:           t.Budget,
		Revenue:          t.Revenue,
		PosterPath:       t.PosterPath,
		BackdropPath:     t.BackdropPath,
		OriginalLanguage: t.OriginalLanguage,
		Adult:            t.Adult,
	}

	for _, g := range t.Genres {
		m.Genres = append(m.Genres, models.Genre{ID: int64(g.ID), Name: g.Name})
	}

	for _, l := range t.SpokenLanguages {
		if l.Name != "" {
			m.SpokenLanguages = append(m.SpokenLanguages, l.Name)
		}
	}

	if t.BelongsToCollection != nil {
		m.CollectionID = int64(t.BelongsToCollection.ID)
		m.CollectionName = t.BelongsToCollection.Name
	}

	if t.Keywords != nil {
		for _, kw := range t.Keywords.Keywords {
			m.Keywords = append(m.Keywords, models.Keyword{ID: int64(kw.ID), Name: kw.Name})
		}
	}

	// US theatrical certification
	if t.ReleaseDates != nil {
		for _, entry := range t.ReleaseDates.Results {
			if entry.Iso31661 != "US" {
				continue
			}
			for _, rd := range entry.ReleaseDates {
				if rd.Type == 3 && rd.Certification != "" { // 3 = Theatrical
					m.Certification = rd.Certification
					break
				}
			}
			break
		}
	}

	// First official YouTube trailer (fall back to teaser)
	if t.Videos != nil {
		var teaserKey string
		for _, v := range t.Videos.Results {
			if v.Site != "YouTube" {
				continue
			}
			if v.Type == "Trailer" && v.Official {
				m.TrailerYouTubeKey = v.Key
				break
			}
			if v.Type == "Teaser" && v.Official && teaserKey == "" {
				teaserKey = v.Key
			}
		}
		if m.TrailerYouTubeKey == "" {
			m.TrailerYouTubeKey = teaserKey
		}
	}

	if t.Credits != nil {
		for _, c := range t.Credits.Cast {
			if c.Order > 10 {
				continue // limit to top billed cast
			}
			m.Cast = append(m.Cast, models.Person{
				TMDBID:      int64(c.ID),
				Name:        c.Name,
				ProfilePath: c.ProfilePath,
				Role:        c.Character,
				Order:       c.Order,
			})
		}
		for _, cr := range t.Credits.Crew {
			person := models.Person{
				TMDBID:      int64(cr.ID),
				Name:        cr.Name,
				ProfilePath: cr.ProfilePath,
				Job:         cr.Job,
				Department:  cr.Department,
			}
			switch cr.Job {
			case "Director":
				m.Directors = append(m.Directors, person)
			case "Writer", "Screenplay", "Story", "Author":
				m.Writers = append(m.Writers, person)
			case "Producer", "Executive Producer":
				m.Producers = append(m.Producers, person)
			}
		}
	}

	return m
}

// ProvidersForCountry extracts streaming providers for a country from the
// append_to_response=watch/providers data embedded in this movie response.
func (t *TMDBMovie) ProvidersForCountry(country string) []models.Provider {
	if t.WatchProviders == nil {
		return nil
	}
	return extractProviders(t.WatchProviders.Results, country)
}

// TMDBShow is the full TV show details response from TMDB.
type TMDBShow struct {
	ID               int              `json:"id"`
	Name             string           `json:"name"`
	OriginalName     string           `json:"original_name"`
	Tagline          string           `json:"tagline"`
	Overview         string           `json:"overview"`
	FirstAirDate     string           `json:"first_air_date"`
	LastAirDate      string           `json:"last_air_date"`
	NumberOfSeasons  int              `json:"number_of_seasons"`
	NumberOfEpisodes int              `json:"number_of_episodes"`
	VoteAverage      float64          `json:"vote_average"`
	VoteCount        int64            `json:"vote_count"`
	Popularity       float64          `json:"popularity"`
	PosterPath       string           `json:"poster_path"`
	BackdropPath     string           `json:"backdrop_path"`
	OriginalLanguage string           `json:"original_language"`
	Status           string           `json:"status"`
	Genres           []tmdbGenre      `json:"genres"`
	CreatedBy        []tmdbCreator    `json:"created_by"`
	Credits          *tmdbCredits     `json:"credits,omitempty"`
	Keywords         *tmdbTVKeywords  `json:"keywords,omitempty"`
	Videos           *tmdbVideosRoot  `json:"videos,omitempty"`
	WatchProviders   *tmdbWatchRoot   `json:"watch/providers,omitempty"`
	ExternalIDs      *tmdbExternalIDs `json:"external_ids,omitempty"`
}

// ToModel converts a TMDBShow to a models.TVShow.
func (t *TMDBShow) ToModel() *models.TVShow {
	tvWikidataID := ""
	if t.ExternalIDs != nil {
		tvWikidataID = t.ExternalIDs.WikidataID
	}

	show := &models.TVShow{
		TMDBID:           int64(t.ID),
		WikidataID:       tvWikidataID,
		Name:             t.Name,
		OriginalName:     t.OriginalName,
		Tagline:          t.Tagline,
		Overview:         t.Overview,
		FirstAirDate:     t.FirstAirDate,
		LastAirDate:      t.LastAirDate,
		NumberOfSeasons:  t.NumberOfSeasons,
		NumberOfEpisodes: t.NumberOfEpisodes,
		VoteAverage:      t.VoteAverage,
		VoteCount:        t.VoteCount,
		Popularity:       t.Popularity,
		PosterPath:       t.PosterPath,
		BackdropPath:     t.BackdropPath,
		OriginalLanguage: t.OriginalLanguage,
		Status:           t.Status,
	}

	for _, g := range t.Genres {
		show.Genres = append(show.Genres, models.Genre{ID: int64(g.ID), Name: g.Name})
	}

	if t.Keywords != nil {
		for _, kw := range t.Keywords.Results {
			show.Keywords = append(show.Keywords, models.Keyword{ID: int64(kw.ID), Name: kw.Name})
		}
	}

	// First official YouTube trailer (fall back to teaser) — same logic as movies
	if t.Videos != nil {
		var teaserKey string
		for _, v := range t.Videos.Results {
			if v.Site != "YouTube" {
				continue
			}
			if v.Type == "Trailer" && v.Official {
				show.TrailerYouTubeKey = v.Key
				break
			}
			if v.Type == "Teaser" && v.Official && teaserKey == "" {
				teaserKey = v.Key
			}
		}
		if show.TrailerYouTubeKey == "" {
			show.TrailerYouTubeKey = teaserKey
		}
	}

	for _, c := range t.CreatedBy {
		show.Creators = append(show.Creators, models.Person{
			TMDBID: int64(c.ID),
			Name:   c.Name,
		})
	}

	if t.Credits != nil {
		for _, c := range t.Credits.Cast {
			if c.Order > 10 {
				continue
			}
			show.Cast = append(show.Cast, models.Person{
				TMDBID:      int64(c.ID),
				Name:        c.Name,
				ProfilePath: c.ProfilePath,
				Role:        c.Character,
				Order:       c.Order,
			})
		}
	}

	return show
}

// ProvidersForCountry extracts streaming providers for a country from the
// append_to_response=watch/providers data embedded in this show response.
func (t *TMDBShow) ProvidersForCountry(country string) []models.Provider {
	if t.WatchProviders == nil {
		return nil
	}
	return extractProviders(t.WatchProviders.Results, country)
}

// extractProviders converts a tmdbWatchRoot results map entry to []models.Provider.
func extractProviders(results map[string]tmdbCountryProviders, country string) []models.Provider {
	cp, ok := results[country]
	if !ok {
		return nil
	}
	out := make([]models.Provider, 0, len(cp.Flatrate)+len(cp.Rent)+len(cp.Buy))
	for _, p := range cp.Flatrate {
		out = append(out, models.Provider{ProviderID: int64(p.ProviderID), ProviderName: p.ProviderName, LogoPath: p.LogoPath, DisplayPriority: p.DisplayPriority, Type: "flatrate", Country: country})
	}
	for _, p := range cp.Rent {
		out = append(out, models.Provider{ProviderID: int64(p.ProviderID), ProviderName: p.ProviderName, LogoPath: p.LogoPath, DisplayPriority: p.DisplayPriority, Type: "rent", Country: country})
	}
	for _, p := range cp.Buy {
		out = append(out, models.Provider{ProviderID: int64(p.ProviderID), ProviderName: p.ProviderName, LogoPath: p.LogoPath, DisplayPriority: p.DisplayPriority, Type: "buy", Country: country})
	}
	return out
}

// TMDBProvider represents a single streaming provider in TMDB's watch/providers response.
type TMDBProvider struct {
	ProviderID      int    `json:"provider_id"`
	ProviderName    string `json:"provider_name"`
	LogoPath        string `json:"logo_path"`
	DisplayPriority int    `json:"display_priority"`
	Type            string `json:"-"` // injected by caller: "flatrate", "rent", "buy"
}

// ---- Private TMDB structs ----

type tmdbGenre struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type tmdbCredits struct {
	Cast []tmdbCastMember `json:"cast"`
	Crew []tmdbCrewMember `json:"crew"`
}

type tmdbCastMember struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Character   string `json:"character"`
	ProfilePath string `json:"profile_path"`
	Order       int    `json:"order"`
}

type tmdbCrewMember struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Job         string `json:"job"`
	Department  string `json:"department"`
	ProfilePath string `json:"profile_path"`
}

type tmdbCreator struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type tmdbKeywords struct {
	Keywords []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"keywords"`
}

// tmdbTVKeywords handles TV show keywords (TMDB uses "results" key instead of "keywords").
type tmdbTVKeywords struct {
	Results []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"results"`
}

type tmdbCollection struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type tmdbExternalIDs struct {
	WikidataID string `json:"wikidata_id"`
	IMDbID     string `json:"imdb_id"`
}

type tmdbLanguage struct {
	Iso6391 string `json:"iso_639_1"`
	Name    string `json:"english_name"`
}

type tmdbReleaseDatesRoot struct {
	Results []tmdbReleaseDateEntry `json:"results"`
}

type tmdbReleaseDateEntry struct {
	Iso31661    string           `json:"iso_3166_1"`
	ReleaseDates []tmdbReleaseDate `json:"release_dates"`
}

type tmdbReleaseDate struct {
	Certification string `json:"certification"`
	Type          int    `json:"type"` // 3 = Theatrical
}

type tmdbVideosRoot struct {
	Results []tmdbVideo `json:"results"`
}

type tmdbVideo struct {
	Key      string `json:"key"`
	Site     string `json:"site"`
	Type     string `json:"type"` // "Trailer", "Teaser", "Clip"
	Official bool   `json:"official"`
}

type tmdbWatchRoot struct {
	Results map[string]tmdbCountryProviders `json:"results"`
}

type tmdbCountryProviders struct {
	Flatrate []tmdbRawProvider `json:"flatrate"`
	Rent     []tmdbRawProvider `json:"rent"`
	Buy      []tmdbRawProvider `json:"buy"`
}

type tmdbRawProvider struct {
	ProviderID      int    `json:"provider_id"`
	ProviderName    string `json:"provider_name"`
	LogoPath        string `json:"logo_path"`
	DisplayPriority int    `json:"display_priority"`
}

type tmdbMoviePage struct {
	Results []struct {
		ID int `json:"id"`
	} `json:"results"`
}

type tmdbTrendingPage struct {
	Results []struct {
		ID        int    `json:"id"`
		MediaType string `json:"media_type"`
	} `json:"results"`
}

type tmdbBulkEntry struct {
	ID         int     `json:"id"`
	Original   string  `json:"original_title"`
	Popularity float64 `json:"popularity"`
	Adult      bool    `json:"adult"`
}

// ---- Public API methods ----

// GetMovieDetails fetches full movie details with credits, keywords, watch/providers, similar, release_dates, and videos.
func (c *Client) GetMovieDetails(ctx context.Context, id int) (*TMDBMovie, error) {
	url := fmt.Sprintf("%s/movie/%d?api_key=%s&append_to_response=credits,keywords,watch/providers,similar,release_dates,videos,external_ids", baseURL, id, c.apiKey)
	var result TMDBMovie
	if err := c.get(ctx, url, &result); err != nil {
		return nil, fmt.Errorf("GetMovieDetails(%d): %w", id, err)
	}
	return &result, nil
}

// GetTVDetails fetches full TV show details with credits, keywords, watch/providers, and videos.
func (c *Client) GetTVDetails(ctx context.Context, id int) (*TMDBShow, error) {
	url := fmt.Sprintf("%s/tv/%d?api_key=%s&append_to_response=credits,keywords,watch/providers,videos,external_ids", baseURL, id, c.apiKey)
	var result TMDBShow
	if err := c.get(ctx, url, &result); err != nil {
		return nil, fmt.Errorf("GetTVDetails(%d): %w", id, err)
	}
	return &result, nil
}

// GetTrendingMovies fetches the list of trending movie IDs for the given page.
func (c *Client) GetTrendingMovies(ctx context.Context, page int) ([]int, error) {
	url := fmt.Sprintf("%s/trending/movie/day?api_key=%s&page=%d", baseURL, c.apiKey, page)
	var result tmdbTrendingPage
	if err := c.get(ctx, url, &result); err != nil {
		return nil, fmt.Errorf("GetTrendingMovies(page=%d): %w", page, err)
	}
	ids := make([]int, 0, len(result.Results))
	for _, r := range result.Results {
		ids = append(ids, r.ID)
	}
	return ids, nil
}

// GetBulkMovieIDs downloads the TMDB daily export file and returns movie IDs
// with popularity >= minPopularity. Use minPopularity=0 for all IDs.
// The export file is a gzipped JSONL file at http://files.tmdb.org/p/exports/movie_ids_MM_DD_YYYY.json.gz
func (c *Client) GetBulkMovieIDs(ctx context.Context, minPopularity float64) ([]int, error) {
	// Build URL for yesterday's export (today's may not be ready yet)
	yesterday := time.Now().UTC().AddDate(0, 0, -1)
	exportURL := fmt.Sprintf("%s/movie_ids_%02d_%02d_%04d.json.gz",
		exportsURL, yesterday.Month(), yesterday.Day(), yesterday.Year())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, exportURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build bulk request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("bulk export download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bulk export returned status %d", resp.StatusCode)
	}

	gzr, err := gzip.NewReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("gzip reader: %w", err)
	}
	defer gzr.Close()

	var ids []int
	decoder := json.NewDecoder(gzr)
	for {
		var entry tmdbBulkEntry
		if err := decoder.Decode(&entry); err != nil {
			if err == io.EOF {
				break
			}
			// Skip malformed lines
			continue
		}
		if entry.ID > 0 && !entry.Adult && (minPopularity <= 0 || entry.Popularity >= minPopularity) {
			ids = append(ids, entry.ID)
		}
	}

	return ids, nil
}

// GetWatchProviders returns streaming providers for a title by media type ("movie" or "tv").
// Returns a map of country code → []TMDBProvider with Type set to the offer type.
func (c *Client) GetWatchProviders(ctx context.Context, id int, mediaType string) (map[string][]TMDBProvider, error) {
	url := fmt.Sprintf("%s/%s/%d/watch/providers?api_key=%s", baseURL, mediaType, id, c.apiKey)
	var root tmdbWatchRoot
	if err := c.get(ctx, url, &root); err != nil {
		return nil, fmt.Errorf("GetWatchProviders(%s/%d): %w", mediaType, id, err)
	}

	result := make(map[string][]TMDBProvider, len(root.Results))
	for countryCode, cp := range root.Results {
		var providers []TMDBProvider
		for _, p := range cp.Flatrate {
			providers = append(providers, TMDBProvider{
				ProviderID:      p.ProviderID,
				ProviderName:    p.ProviderName,
				LogoPath:        p.LogoPath,
				DisplayPriority: p.DisplayPriority,
				Type:            "flatrate",
			})
		}
		for _, p := range cp.Rent {
			providers = append(providers, TMDBProvider{
				ProviderID:      p.ProviderID,
				ProviderName:    p.ProviderName,
				LogoPath:        p.LogoPath,
				DisplayPriority: p.DisplayPriority,
				Type:            "rent",
			})
		}
		for _, p := range cp.Buy {
			providers = append(providers, TMDBProvider{
				ProviderID:      p.ProviderID,
				ProviderName:    p.ProviderName,
				LogoPath:        p.LogoPath,
				DisplayPriority: p.DisplayPriority,
				Type:            "buy",
			})
		}
		result[countryCode] = providers
	}

	return result, nil
}

// ---- Internal helpers ----

func (c *Client) get(ctx context.Context, url string, dst interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http get: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("not found (404)")
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return fmt.Errorf("rate limited (429)")
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

// GetTVShowDetails is an alias for GetTVDetails for consistent naming.
func (c *Client) GetTVShowDetails(ctx context.Context, id int) (*TMDBShow, error) {
	return c.GetTVDetails(ctx, id)
}

// GetTrendingTVShows fetches trending TV show IDs for the given page.
func (c *Client) GetTrendingTVShows(ctx context.Context, page int) ([]int, error) {
	url := fmt.Sprintf("%s/trending/tv/day?api_key=%s&page=%d", baseURL, c.apiKey, page)
	var result tmdbTrendingPage
	if err := c.get(ctx, url, &result); err != nil {
		return nil, fmt.Errorf("GetTrendingTVShows(page=%d): %w", page, err)
	}
	ids := make([]int, 0, len(result.Results))
	for _, r := range result.Results {
		ids = append(ids, r.ID)
	}
	return ids, nil
}

// GetBulkTVShowIDs downloads the TMDB daily TV show export and returns show IDs
// with popularity >= minPopularity. Use minPopularity=0 for all IDs.
func (c *Client) GetBulkTVShowIDs(ctx context.Context, minPopularity float64) ([]int, error) {
	yesterday := time.Now().UTC().AddDate(0, 0, -1)
	exportURL := fmt.Sprintf("%s/tv_series_ids_%02d_%02d_%04d.json.gz",
		exportsURL, yesterday.Month(), yesterday.Day(), yesterday.Year())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, exportURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build TV bulk request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("TV bulk export download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TV bulk export returned status %d", resp.StatusCode)
	}

	gzr, err := gzip.NewReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("gzip reader: %w", err)
	}
	defer gzr.Close()

	var ids []int
	decoder := json.NewDecoder(gzr)
	for {
		var entry tmdbBulkEntry
		if err := decoder.Decode(&entry); err != nil {
			if err == io.EOF {
				break
			}
			continue
		}
		if entry.ID > 0 && (minPopularity <= 0 || entry.Popularity >= minPopularity) {
			ids = append(ids, entry.ID)
		}
	}

	return ids, nil
}
