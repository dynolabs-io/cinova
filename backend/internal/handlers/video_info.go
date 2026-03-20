package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/rs/zerolog/log"
)

// VideoInfoHandler serves YouTube video aspect ratio data.
type VideoInfoHandler struct {
	youtubeAPIKey string
	httpClient    *http.Client
}

func NewVideoInfoHandler(youtubeAPIKey string) *VideoInfoHandler {
	return &VideoInfoHandler{
		youtubeAPIKey: youtubeAPIKey,
		httpClient:    &http.Client{Timeout: 5 * time.Second},
	}
}

type VideoInfoResponse struct {
	YoutubeKey  string  `json:"youtube_key"`
	AspectRatio float64 `json:"aspect_ratio"` // width / height, e.g. 1.778 for 16:9
	Width       int     `json:"width"`
	Height      int     `json:"height"`
}

// GetVideoInfo handles GET /api/v1/video-info?youtube_key=XXXX
// Returns the video's native aspect ratio by querying YouTube Data API v3.
func (h *VideoInfoHandler) GetVideoInfo(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("youtube_key")
	if key == "" {
		writeError(w, "bad_request", "youtube_key is required", http.StatusBadRequest)
		return
	}

	if h.youtubeAPIKey == "" {
		writeError(w, "not_configured", "YouTube API key not configured", http.StatusServiceUnavailable)
		return
	}

	url := fmt.Sprintf(
		"https://www.googleapis.com/youtube/v3/videos?id=%s&part=player&maxWidth=3840&key=%s",
		key, h.youtubeAPIKey,
	)

	resp, err := h.httpClient.Get(url)
	if err != nil {
		log.Error().Err(err).Str("youtube_key", key).Msg("YouTube API request failed")
		writeError(w, "upstream_error", "failed to reach YouTube API", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	var ytResp struct {
		Items []struct {
			Player struct {
				EmbedWidth  string `json:"embedWidth"`
				EmbedHeight string `json:"embedHeight"`
			} `json:"player"`
		} `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&ytResp); err != nil {
		log.Error().Err(err).Str("youtube_key", key).Msg("failed to decode YouTube API response")
		writeError(w, "upstream_error", "invalid response from YouTube API", http.StatusBadGateway)
		return
	}

	if len(ytResp.Items) == 0 {
		writeError(w, "not_found", "video not found", http.StatusNotFound)
		return
	}

	w64, _ := strconv.Atoi(ytResp.Items[0].Player.EmbedWidth)
	h64, _ := strconv.Atoi(ytResp.Items[0].Player.EmbedHeight)
	if w64 == 0 || h64 == 0 {
		writeError(w, "upstream_error", "YouTube returned zero dimensions", http.StatusBadGateway)
		return
	}

	writeJSON(w, http.StatusOK, VideoInfoResponse{
		YoutubeKey:  key,
		AspectRatio: float64(w64) / float64(h64),
		Width:       w64,
		Height:      h64,
	})
}
