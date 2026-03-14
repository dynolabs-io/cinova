package streaming

import "fmt"

// ProviderInfo holds static metadata for a known streaming service.
type ProviderInfo struct {
	ID       int
	Name     string
	LogoPath string // TMDB logo path (e.g. "/pbpMk2JmcoNnQwx5JGpXngfoWtp.jpg")
	Color    string // brand hex color
}

// KnownProviders is a map of TMDB provider ID → ProviderInfo for major services.
var KnownProviders = map[int]ProviderInfo{
	8: {
		ID:    8,
		Name:  "Netflix",
		Color: "#E50914",
	},
	9: {
		ID:    9,
		Name:  "Amazon Prime Video",
		Color: "#00A8E0",
	},
	337: {
		ID:    337,
		Name:  "Disney+",
		Color: "#113CCF",
	},
	350: {
		ID:    350,
		Name:  "Apple TV+",
		Color: "#000000",
	},
	1899: {
		ID:    1899,
		Name:  "Max",
		Color: "#002BE7",
	},
	15: {
		ID:    15,
		Name:  "Hulu",
		Color: "#1CE783",
	},
	387: {
		ID:    387,
		Name:  "Peacock",
		Color: "#000000",
	},
	531: {
		ID:    531,
		Name:  "Paramount+",
		Color: "#0064FF",
	},
	192: {
		ID:    192,
		Name:  "YouTube",
		Color: "#FF0000",
	},
}

// GetProviderInfo returns the ProviderInfo for a given TMDB provider ID.
// Returns nil if the provider is not in the known list.
func GetProviderInfo(providerID int) *ProviderInfo {
	if info, ok := KnownProviders[providerID]; ok {
		return &info
	}
	return nil
}

// GetJustWatchLink returns a JustWatch deep-link URL for a title.
// mediaType should be "movie" or "tv".
// The URL allows users to find where to watch across all services.
func GetJustWatchLink(tmdbID int, mediaType string) string {
	// JustWatch uses locale-based URLs; default to US English.
	switch mediaType {
	case "tv":
		return fmt.Sprintf("https://www.justwatch.com/us/tv-show/%d", tmdbID)
	default:
		return fmt.Sprintf("https://www.justwatch.com/us/movie/%d", tmdbID)
	}
}
