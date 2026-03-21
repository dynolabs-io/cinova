package youtube

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode"
)

const (
	searchURL   = "https://www.googleapis.com/youtube/v3/search"
	videosURL   = "https://www.googleapis.com/youtube/v3/videos"
	minDuration = 30  // seconds
	maxDuration = 300 // seconds (5 min)
)

type VerticalFinder struct {
	apiKey     string
	httpClient *http.Client
}

func NewVerticalFinder(apiKey string) *VerticalFinder {
	return &VerticalFinder{
		apiKey: apiKey,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// FindVerticalTrailer searches YouTube for a portrait trailer for the given movie.
// It tries multiple search queries in order of specificity and returns the first
// validated vertical video key, or empty string if none found.
//
// Strictness rules to avoid wrong mappings:
//   - Duration: 30s–5min only
//   - Aspect ratio: height > width (portrait)
//   - Title fuzzy match: candidate snippet title must contain the movie title
func (f *VerticalFinder) FindVerticalTrailer(title string, year int) (string, error) {
	if f.apiKey == "" {
		return "", fmt.Errorf("youtube api key not configured")
	}

	queries := []string{
		fmt.Sprintf(`"%s" vertical trailer`, title),
		fmt.Sprintf(`"%s" %d vertical trailer`, title, year),
		fmt.Sprintf(`"%s" official vertical trailer`, title),
		fmt.Sprintf(`"%s" trailer shorts`, title),
	}

	for _, q := range queries {
		candidates, err := f.search(q)
		if err != nil {
			continue
		}
		for _, id := range candidates {
			ok, err := f.validateVertical(id, title)
			if err != nil {
				continue
			}
			if ok {
				return id, nil
			}
		}
	}
	return "", nil
}

// search returns up to 5 video IDs for a YouTube search query.
func (f *VerticalFinder) search(query string) ([]string, error) {
	params := url.Values{}
	params.Set("part", "snippet")
	params.Set("q", query)
	params.Set("type", "video")
	params.Set("maxResults", "5")
	params.Set("videoDuration", "short")    // under 4 minutes
	params.Set("videoEmbeddable", "true")   // only embeddable videos
	params.Set("key", f.apiKey)

	resp, err := f.httpClient.Get(searchURL + "?" + params.Encode())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Items []struct {
			ID struct {
				VideoID string `json:"videoId"`
			} `json:"id"`
			Snippet struct {
				Title string `json:"title"`
			} `json:"snippet"`
		} `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	ids := make([]string, 0, len(result.Items))
	for _, item := range result.Items {
		ids = append(ids, item.ID.VideoID)
	}
	return ids, nil
}

// validateVertical checks that a YouTube video is:
//   - Portrait orientation (height > width)
//   - Duration 30s–5min
//   - Title contains the movie title (fuzzy match to avoid wrong mappings)
func (f *VerticalFinder) validateVertical(videoID, movieTitle string) (bool, error) {
	params := url.Values{}
	params.Set("part", "player,contentDetails,snippet,status")
	params.Set("id", videoID)
	params.Set("maxWidth", "3840")
	params.Set("key", f.apiKey)

	resp, err := f.httpClient.Get(videosURL + "?" + params.Encode())
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	var result struct {
		Items []struct {
			Snippet struct {
				Title string `json:"title"`
			} `json:"snippet"`
			Player struct {
				EmbedWidth  string `json:"embedWidth"`
				EmbedHeight string `json:"embedHeight"`
			} `json:"player"`
			ContentDetails struct {
				Duration string `json:"duration"` // ISO 8601 e.g. PT1M30S
			} `json:"contentDetails"`
			Status struct {
				Embeddable bool `json:"embeddable"`
			} `json:"status"`
		} `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, err
	}
	if len(result.Items) == 0 {
		return false, nil
	}

	item := result.Items[0]

	// Must be embeddable — reject immediately if studio has disabled embedding
	if !item.Status.Embeddable {
		return false, nil
	}

	// Title must contain the movie title (normalised, case-insensitive)
	if !titleMatches(item.Snippet.Title, movieTitle) {
		return false, nil
	}

	// Parse dimensions
	var w, h int
	fmt.Sscanf(item.Player.EmbedWidth, "%d", &w)
	fmt.Sscanf(item.Player.EmbedHeight, "%d", &h)
	if w == 0 || h == 0 {
		return false, nil
	}

	// Must be portrait
	if h <= w {
		return false, nil
	}

	// Duration check
	secs := parseDuration(item.ContentDetails.Duration)
	if secs < minDuration || secs > maxDuration {
		return false, nil
	}

	return true, nil
}

// titleMatches returns true if the candidate video title contains the movie title
// (case-insensitive, normalised, ignoring common noise words like "official", "trailer").
func titleMatches(videoTitle, movieTitle string) bool {
	norm := func(s string) string {
		s = strings.ToLower(s)
		// remove non-letter/digit/space
		var b strings.Builder
		for _, r := range s {
			if unicode.IsLetter(r) || unicode.IsDigit(r) || r == ' ' {
				b.WriteRune(r)
			}
		}
		return strings.TrimSpace(b.String())
	}
	return strings.Contains(norm(videoTitle), norm(movieTitle))
}

// parseDuration parses ISO 8601 duration string (e.g. "PT1M30S") into seconds.
func parseDuration(iso string) int {
	// Strip PT prefix
	iso = strings.TrimPrefix(iso, "PT")
	var total int
	var cur int
	for _, c := range iso {
		switch {
		case c >= '0' && c <= '9':
			cur = cur*10 + int(c-'0')
		case c == 'H':
			total += cur * 3600
			cur = 0
		case c == 'M':
			total += cur * 60
			cur = 0
		case c == 'S':
			total += cur
			cur = 0
		}
	}
	return total
}
