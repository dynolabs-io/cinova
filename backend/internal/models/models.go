package models

import "time"

// ---- Media ----

// Movie represents a movie node in the graph database.
type Movie struct {
	TMDBID          int64     `json:"tmdb_id"`
	IMDbID          string    `json:"imdb_id,omitempty"`
	Title           string    `json:"title"`
	OriginalTitle   string    `json:"original_title,omitempty"`
	Overview        string    `json:"overview,omitempty"`
	ReleaseDate     string    `json:"release_date,omitempty"`
	Runtime         int       `json:"runtime,omitempty"`
	VoteAverage     float64   `json:"vote_average"`
	VoteCount       int64     `json:"vote_count"`
	Popularity      float64   `json:"popularity"`
	PosterPath      string    `json:"poster_path,omitempty"`
	BackdropPath    string    `json:"backdrop_path,omitempty"`
	OriginalLanguage string   `json:"original_language,omitempty"`
	Adult           bool      `json:"adult"`
	CinovaScore     float64   `json:"cinova_score,omitempty"`
	Genres          []Genre   `json:"genres,omitempty"`
	Themes          []Theme   `json:"themes,omitempty"`
	Moods           []Mood    `json:"moods,omitempty"`
	Cast            []Person  `json:"cast,omitempty"`
	Directors       []Person  `json:"directors,omitempty"`
	Providers       []Provider `json:"providers,omitempty"`
}

// TVShow represents a TV series node in the graph database.
type TVShow struct {
	TMDBID           int64     `json:"tmdb_id"`
	Name             string    `json:"name"`
	OriginalName     string    `json:"original_name,omitempty"`
	Overview         string    `json:"overview,omitempty"`
	FirstAirDate     string    `json:"first_air_date,omitempty"`
	LastAirDate      string    `json:"last_air_date,omitempty"`
	NumberOfSeasons  int       `json:"number_of_seasons,omitempty"`
	NumberOfEpisodes int       `json:"number_of_episodes,omitempty"`
	VoteAverage      float64   `json:"vote_average"`
	VoteCount        int64     `json:"vote_count"`
	Popularity       float64   `json:"popularity"`
	PosterPath       string    `json:"poster_path,omitempty"`
	BackdropPath     string    `json:"backdrop_path,omitempty"`
	OriginalLanguage string    `json:"original_language,omitempty"`
	Status           string    `json:"status,omitempty"`
	CinovaScore      float64   `json:"cinova_score,omitempty"`
	Genres           []Genre   `json:"genres,omitempty"`
	Themes           []Theme   `json:"themes,omitempty"`
	Moods            []Mood    `json:"moods,omitempty"`
	Cast             []Person  `json:"cast,omitempty"`
	Creators         []Person  `json:"creators,omitempty"`
	Providers        []Provider `json:"providers,omitempty"`
}

// Person represents an actor, director, or other crew member.
type Person struct {
	TMDBID      int64  `json:"tmdb_id"`
	WikidataID  string `json:"wikidata_id,omitempty"`
	Name        string `json:"name"`
	ProfilePath string `json:"profile_path,omitempty"`
	Role        string `json:"role,omitempty"`   // actor role / character name
	Department  string `json:"department,omitempty"`
	Job         string `json:"job,omitempty"`
	Order       int    `json:"order,omitempty"` // cast billing order
}

// Genre represents a movie/TV genre.
type Genre struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// Theme is an AI-extracted thematic tag.
type Theme struct {
	Name  string  `json:"name"`
	Score float64 `json:"score,omitempty"` // confidence
}

// Mood is an AI-extracted emotional tone.
type Mood struct {
	Name  string  `json:"name"`
	Score float64 `json:"score,omitempty"`
}

// Provider represents a streaming service.
type Provider struct {
	ProviderID   int64  `json:"provider_id"`
	ProviderName string `json:"provider_name"`
	LogoPath     string `json:"logo_path,omitempty"`
	DisplayPriority int  `json:"display_priority,omitempty"`
	Type         string `json:"type"` // "flatrate", "rent", "buy"
	DeepLink     string `json:"deep_link,omitempty"`
	Country      string `json:"country,omitempty"`
}

// SearchResult is a unified result item returned from search or recommendations.
type SearchResult struct {
	TMDBID      int64   `json:"tmdb_id"`
	MediaType   string  `json:"media_type"` // "movie" or "tv"
	Title       string  `json:"title"`
	PosterPath  string  `json:"poster_path,omitempty"`
	ReleaseYear string  `json:"release_year,omitempty"`
	VoteAverage float64 `json:"vote_average"`
	CinovaScore float64 `json:"cinova_score"`
	Overview    string  `json:"overview,omitempty"`
	Genres      []Genre `json:"genres,omitempty"`
	Providers   []Provider `json:"providers,omitempty"`
	MatchReason string  `json:"match_reason,omitempty"` // AI-generated explanation
}

// ---- Interactions ----

// InteractionType defines the kind of user interaction with a title.
type InteractionType string

const (
	InteractionRate    InteractionType = "RATED"
	InteractionSave    InteractionType = "SAVED"
	InteractionDismiss InteractionType = "DISMISSED"
	InteractionWatch   InteractionType = "WATCHED"
)

// Interaction records a user or session interaction with a title.
type Interaction struct {
	ID          string          `json:"id"`
	SessionID   string          `json:"session_id,omitempty"`
	UserID      string          `json:"user_id,omitempty"`
	TMDBID      int64           `json:"tmdb_id"`
	MediaType   string          `json:"media_type"`
	Type        InteractionType `json:"type"`
	Score       float64         `json:"score,omitempty"` // for RATED
	CreatedAt   time.Time       `json:"created_at"`
}

// ---- Auth / Users ----

// User is a registered Cinova account.
type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	DisplayName  string    `json:"display_name,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// AnonymousSession tracks a guest user session before signup.
type AnonymousSession struct {
	UUID      string    `json:"uuid"`
	DeviceID  string    `json:"device_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	MergedAt  *time.Time `json:"merged_at,omitempty"`
	UserID    *string   `json:"user_id,omitempty"`
}

// ---- Request / Response helpers ----

// RateRequest is the payload for POST /api/v1/me/rate.
type RateRequest struct {
	TMDBID    int64   `json:"tmdb_id"`
	MediaType string  `json:"media_type"`
	Score     float64 `json:"score"` // 0.0 – 10.0
}

// SaveRequest is the payload for POST /api/v1/me/save.
type SaveRequest struct {
	TMDBID    int64  `json:"tmdb_id"`
	MediaType string `json:"media_type"`
}

// DismissRequest is the payload for POST /api/v1/me/dismiss.
type DismissRequest struct {
	TMDBID    int64  `json:"tmdb_id"`
	MediaType string `json:"media_type"`
}

// SignupRequest is the payload for POST /api/v1/auth/signup.
type SignupRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name,omitempty"`
	SessionUUID string `json:"session_uuid,omitempty"` // anonymous session to merge
}

// LoginRequest is the payload for POST /api/v1/auth/login.
type LoginRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	SessionUUID string `json:"session_uuid,omitempty"` // anonymous session to merge
}

// TokenResponse is returned after successful auth.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"` // seconds
	UserID       string `json:"user_id,omitempty"`
	Anonymous    bool   `json:"anonymous"`
}

// AnonymousRequest is the payload for POST /api/v1/auth/anonymous.
type AnonymousRequest struct {
	DeviceID string `json:"device_id,omitempty"`
}

// RefreshRequest is the payload for POST /api/v1/auth/refresh.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// ErrorResponse is the standard API error envelope.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
	Code    int    `json:"code"`
}
