package models

import "time"

// ---- Media ----

// Movie represents a movie node in the graph database.
type Movie struct {
	TMDBID             int64      `json:"tmdb_id"`
	MediaType          string     `json:"media_type"`
	IMDbID             string     `json:"imdb_id,omitempty"`
	WikidataID         string     `json:"wikidata_id,omitempty"`
	Title              string     `json:"title"`
	OriginalTitle      string     `json:"original_title,omitempty"`
	Tagline            string     `json:"tagline,omitempty"`
	Overview           string     `json:"overview,omitempty"`
	ReleaseDate        string     `json:"release_date,omitempty"`
	Runtime            int        `json:"runtime,omitempty"`
	VoteAverage        float64    `json:"vote_average"`
	VoteCount          int64      `json:"vote_count"`
	Popularity         float64    `json:"popularity"`
	Budget             int64      `json:"budget,omitempty"`
	Revenue            int64      `json:"revenue,omitempty"`
	Certification      string     `json:"certification,omitempty"`
	TrailerYouTubeKey  string     `json:"trailer_youtube_key,omitempty"`
	PosterPath         string     `json:"poster_path,omitempty"`
	BackdropPath       string     `json:"backdrop_path,omitempty"`
	OriginalLanguage   string     `json:"original_language,omitempty"`
	SpokenLanguages    []string   `json:"spoken_languages,omitempty"`
	CollectionID       int64      `json:"collection_id,omitempty"`
	CollectionName     string     `json:"collection_name,omitempty"`
	Adult              bool       `json:"adult"`
	CinovaScore        float64    `json:"cinova_score,omitempty"`
	Genres             []Genre    `json:"genres,omitempty"`
	Keywords           []Keyword  `json:"keywords,omitempty"`
	Themes             []Theme    `json:"themes,omitempty"`
	Moods              []Mood     `json:"moods,omitempty"`
	Cast               []Person   `json:"cast,omitempty"`
	Directors          []Person   `json:"directors,omitempty"`
	Writers            []Person   `json:"writers,omitempty"`
	Producers          []Person   `json:"producers,omitempty"`
	Providers          []Provider `json:"providers,omitempty"`
	Awards             []Award    `json:"awards,omitempty"`
	PlotSummary        string     `json:"plot_summary,omitempty"`
	CinovaSynopsis     string     `json:"cinova_synopsis,omitempty"`
}

// TVShow represents a TV series node in the graph database.
type TVShow struct {
	TMDBID           int64      `json:"tmdb_id"`
	MediaType        string     `json:"media_type"`
	WikidataID       string     `json:"wikidata_id,omitempty"`
	Name             string     `json:"name"`
	OriginalName     string     `json:"original_name,omitempty"`
	Tagline          string     `json:"tagline,omitempty"`
	Overview         string     `json:"overview,omitempty"`
	FirstAirDate     string     `json:"first_air_date,omitempty"`
	LastAirDate      string     `json:"last_air_date,omitempty"`
	NumberOfSeasons  int        `json:"number_of_seasons,omitempty"`
	NumberOfEpisodes int        `json:"number_of_episodes,omitempty"`
	VoteAverage      float64    `json:"vote_average"`
	VoteCount        int64      `json:"vote_count"`
	Popularity       float64    `json:"popularity"`
	PosterPath       string     `json:"poster_path,omitempty"`
	BackdropPath     string     `json:"backdrop_path,omitempty"`
	OriginalLanguage string     `json:"original_language,omitempty"`
	Status           string     `json:"status,omitempty"`
	CinovaScore      float64    `json:"cinova_score,omitempty"`
	Genres           []Genre    `json:"genres,omitempty"`
	Keywords         []Keyword  `json:"keywords,omitempty"`
	Themes           []Theme    `json:"themes,omitempty"`
	Moods            []Mood     `json:"moods,omitempty"`
	Cast             []Person   `json:"cast,omitempty"`
	Creators         []Person   `json:"creators,omitempty"`
	Providers        []Provider `json:"providers,omitempty"`
	PlotSummary      string     `json:"plot_summary,omitempty"`
	CinovaSynopsis   string     `json:"cinova_synopsis,omitempty"`
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

// Keyword is a structured TMDB vocabulary tag (e.g. "vampire", "1930s", "southern gothic").
type Keyword struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// Award represents a film/TV award or nomination sourced from Wikidata (P166/P1411).
type Award struct {
	WikidataID    string `json:"wikidata_id,omitempty"`   // QID of the award item
	AwardName     string `json:"award_name"`
	CeremonyName  string `json:"ceremony_name,omitempty"`
	Year          int    `json:"year,omitempty"`
	RecipientName string `json:"recipient_name,omitempty"` // individual winner if applicable
	Category      string `json:"category,omitempty"`
	IsNomination  bool   `json:"is_nomination"`
}

// ScoringProfile stores a user's personalized scoring weights preset.
type ScoringProfile struct {
	Preset     string  `json:"preset"`     // "mainstream", "cinephile", "arthouse", "blockbuster", "award_season"
	Audience   float64 `json:"audience"`
	Critic     float64 `json:"critic"`
	Award      float64 `json:"award"`
	Prestige   float64 `json:"prestige"`
	Commercial float64 `json:"commercial"`
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
