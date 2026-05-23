package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/dynolabs-io/cinova/backend/internal/config"
)

// STTHandler handles speech-to-text transcription via Groq Whisper.
type STTHandler struct {
	cfg    *config.Config
	client *http.Client
}

// NewSTTHandler creates a new STTHandler.
func NewSTTHandler(cfg *config.Config) *STTHandler {
	return &STTHandler{
		cfg:    cfg,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// Transcribe accepts a multipart audio file and returns a JSON transcript.
// POST /api/v1/me/stt
func (h *STTHandler) Transcribe(w http.ResponseWriter, r *http.Request) {
	if h.cfg.GroqAPIKey == "" {
		http.Error(w, `{"error":"STT not configured"}`, http.StatusServiceUnavailable)
		return
	}

	// Parse multipart — limit to 25MB
	if err := r.ParseMultipartForm(25 << 20); err != nil {
		http.Error(w, `{"error":"bad request"}`, http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("audio")
	if err != nil {
		http.Error(w, `{"error":"audio field missing"}`, http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Read audio bytes
	audioBytes, err := io.ReadAll(io.LimitReader(file, 25<<20))
	if err != nil {
		http.Error(w, `{"error":"failed to read audio"}`, http.StatusInternalServerError)
		return
	}

	// Forward to Groq Whisper
	transcript, err := h.callGroqWhisper(audioBytes, header.Filename)
	if err != nil {
		log.Error().Err(err).Msg("stt: groq whisper failed")
		http.Error(w, `{"error":"transcription failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"transcript": transcript})
}

func (h *STTHandler) callGroqWhisper(audioBytes []byte, filename string) (string, error) {
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)

	// model field
	if err := mw.WriteField("model", "whisper-large-v3-turbo"); err != nil {
		return "", err
	}

	// determine content type from filename
	ext := "m4a"
	if parts := strings.Split(filename, "."); len(parts) > 1 {
		ext = strings.ToLower(parts[len(parts)-1])
	}
	mimeMap := map[string]string{
		"m4a": "audio/m4a", "mp4": "audio/mp4",
		"3gp": "audio/3gpp", "webm": "audio/webm",
		"wav": "audio/wav", "mp3": "audio/mpeg",
	}
	mime := mimeMap[ext]
	if mime == "" {
		mime = "audio/m4a"
	}

	// audio file field
	part, err := mw.CreateFormFile("file", filename)
	if err != nil {
		return "", err
	}
	if _, err := part.Write(audioBytes); err != nil {
		return "", err
	}
	mw.Close()

	req, err := http.NewRequest(http.MethodPost, "https://api.groq.com/openai/v1/audio/transcriptions", &body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+h.cfg.GroqAPIKey)

	resp, err := h.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("groq http: %w", err)
	}
	defer resp.Body.Close()

	respBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("groq returned %d: %s", resp.StatusCode, strings.TrimSpace(string(respBytes)))
	}

	var result struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(respBytes, &result); err != nil {
		return "", fmt.Errorf("parse groq response: %w", err)
	}
	return strings.TrimSpace(result.Text), nil
}
