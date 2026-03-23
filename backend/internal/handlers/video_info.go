package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
	"unicode"

	"github.com/go-chi/chi/v5"
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

// ServeEmbed handles GET /api/v1/embed/{key}
// Returns an HTML page with a full-viewport YouTube iframe.
// Serving from a real HTTPS origin (api.cinova.openova.io) allows YouTube embedding
// without Error 153/152 — identical to what the catalog page does on Safari.
func (h *VideoInfoHandler) ServeEmbed(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	for _, c := range key {
		if !unicode.IsLetter(c) && !unicode.IsDigit(c) && c != '-' && c != '_' {
			http.Error(w, "invalid key", http.StatusBadRequest)
			return
		}
	}
	mute := "0"
	if r.URL.Query().Get("mute") == "1" {
		mute = "1"
	}
	controls := "1"
	if r.URL.Query().Get("controls") == "0" {
		controls = "0"
	}
	// autoplay=0 → player initialises and buffers but does not call playVideo().
	// The React Native WebView injects playVideo()/pauseVideo() via JS when the
	// item becomes active/inactive, enabling instant play on swipe (preload pattern).
	autoplay := r.URL.Query().Get("autoplay") != "0"

	onReady := `if(window.ReactNativeWebView){window.ReactNativeWebView.postMessage(JSON.stringify({type:'playerReady'}));}`
	if autoplay {
		onReady += `e.target.playVideo();`
	}
	// onStateChange: notify RN with current video key, loop on end
	onStateChange := `if(e.data===1){` +
		`if(window.ReactNativeWebView){window.ReactNativeWebView.postMessage(JSON.stringify({type:'playerPlaying',videoKey:currentKey}));}}` +
		`else if(e.data===0){e.target.seekTo(0);e.target.playVideo();}`

	html := `<!DOCTYPE html><html><head><meta name="viewport" content="width=device-width,initial-scale=1,maximum-scale=1">` +
		`<style>*{margin:0;padding:0;box-sizing:border-box;}` +
		`html,body{width:100%;height:100%;background:transparent;overflow:hidden;}` +
		`#p,#p iframe{position:absolute;top:0;left:0;width:100%!important;height:100%!important;border:none;}` +
		`</style></head><body><div id="p"></div>` +
		`<script>` +
		`var player;var currentKey='` + key + `';` +
		`var s=document.createElement('script');s.src='https://www.youtube.com/iframe_api';document.head.appendChild(s);` +
		`function pauseAll(){if(player&&player.pauseVideo){player.pauseVideo();}}` +
		`function playAll(){if(player&&player.playVideo){player.playVideo();}}` +
		`function switchVideo(key){currentKey=key;if(player&&player.loadVideoById){player.loadVideoById(key);}}` +
		`function onYouTubeIframeAPIReady(){` +
		`player=new YT.Player('p',{videoId:'` + key + `',` +
		`playerVars:{autoplay:1,mute:` + mute + `,controls:` + controls + `,rel:0,modestbranding:1,playsinline:1},` +
		`events:{onReady:function(e){` + onReady + `},onStateChange:function(e){` + onStateChange + `}}});}` +
		`</script></body></html>`
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, html)
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
