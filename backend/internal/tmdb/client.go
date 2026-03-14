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

	"github.com/foundrylab-app/cinova/backend/internal/models"
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
	ID               int            `json:"id"`
	IMDbID           string         `json:"imdb_id"`
	Title            string         `json:"title"`
	OriginalTitle    string         `json:"original_title"`
	Overview         string         `json:"overview"`
	ReleaseDate      string         `json:"release_date"`
	Runtime          int            `json:"runtime"`
	VoteAverage      float64        `json:"vote_average"`
	VoteCount        int64          `json:"vote_count"`
	Popularity       float64        `json:"popularity"`
	PosterPath       string         `json:"poster_path"`
	BackdropPath     string         `json:"backdrop_path"`
	OriginalLanguage string         `json:"original_language"`
	Adult            bool           `json:"adult"`
	Genres           []tmdbGenre    `json:"genres"`
	Credits          *tmdbCredits   `json:"credits,omitempty"`
	Keywords         *tmdbKeywords  `json:"keywords,omitempty"`
	WatchProviders   *tmdbWatchRoot `json:"watch/providers,omitempty"`
	Similar          *tmdbMoviePage `json:"similar,omitempty"`
}

// ToModel converts a TMDBMovie to a models.Movie.
func (t *TMDBMovie) ToModel() *models.Movie {
	m := &models.Movie{
		TMDBID:           int64(t.ID),
		IMDbID:           t.IMDbID,
		Title:            t.Title,
		OriginalTitle:    t.OriginalTitle,
		Overview:         t.Overview,
		ReleaseDate:      t.ReleaseDate,
		Runtime:          t.Runtime,
		VoteAverage:      t.VoteAverage,
		VoteCount:        t.VoteCount,
		Popularity:       t.Popularity,
		PosterPath:       t.PosterPath,
		BackdropPath:     t.BackdropPath,
		OriginalLanguage: t.OriginalLanguage,
		Adult:            t.Adult,
	}

	for _, g := range t.Genres {
		m.Genres = append(m.Genres, models.Genre{ID: int64(g.ID), Name: g.Name})
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
			if cr.Job == "Director" {
				m.Directors = append(m.Directors, models.Person{
					TMDBID:      int64(cr.ID),
					Name:        cr.Name,
					ProfilePath: cr.ProfilePath,
					Job:         cr.Job,
					Department:  cr.Department,
				})
			}
		}
	}

	return m
}

// TMDBShow is the full TV show details response from TMDB.
type TMDBShow struct {
	ID               int            `json:"id"`
	Name             string         `json:"name"`
	OriginalName     string         `json:"original_name"`
	Overview         string         `json:"overview"`
	FirstAirDate     string         `json:"first_air_date"`
	LastAirDate      string         `json:"last_air_date"`
	NumberOfSeasons  int            `json:"number_of_seasons"`
	NumberOfEpisodes int            `json:"number_of_episodes"`
	VoteAverage      float64        `json:"vote_average"`
	VoteCount        int64          `json:"vote_count"`
	Popularity       float64        `json:"popularity"`
	PosterPath       string         `json:"poster_path"`
	BackdropPath     string         `json:"backdrop_path"`
	OriginalLanguage string         `json:"original_language"`
	Status           string         `json:"status"`
	Genres           []tmdbGenre    `json:"genres"`
	CreatedBy        []tmdbCreator  `json:"created_by"`
	Credits          *tmdbCredits   `json:"credits,omitempty"`
	WatchProviders   *tmdbWatchRoot `json:"watch/providers,omitempty"`
}

// ToModel converts a TMDBShow to a models.TVShow.
func (t *TMDBShow) ToModel() *models.TVShow {
	show := &models.TVShow{
		TMDBID:           int64(t.ID),
		Name:             t.Name,
		OriginalName:     t.OriginalName,
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
	ID       int    `json:"id"`
	Original string `json:"original_title"`
}

// ---- Public API methods ----

// GetMovieDetails fetches full movie details with credits, keywords, watch/providers, and similar.
func (c *Client) GetMovieDetails(ctx context.Context, id int) (*TMDBMovie, error) {
	url := fmt.Sprintf("%s/movie/%d?api_key=%s&append_to_response=credits,keywords,watch/providers,similar", baseURL, id, c.apiKey)
	var result TMDBMovie
	if err := c.get(ctx, url, &result); err != nil {
		return nil, fmt.Errorf("GetMovieDetails(%d): %w", id, err)
	}
	return &result, nil
}

// GetTVDetails fetches full TV show details with credits, keywords, and watch/providers.
func (c *Client) GetTVDetails(ctx context.Context, id int) (*TMDBShow, error) {
	url := fmt.Sprintf("%s/tv/%d?api_key=%s&append_to_response=credits,keywords,watch/providers", baseURL, id, c.apiKey)
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

// GetBulkMovieIDs downloads the TMDB daily export file and returns all movie IDs.
// The export file is a gzipped JSONL file at http://files.tmdb.org/p/exports/movie_ids_MM_DD_YYYY.json.gz
func (c *Client) GetBulkMovieIDs(ctx context.Context) ([]int, error) {
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
		if entry.ID > 0 {
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
