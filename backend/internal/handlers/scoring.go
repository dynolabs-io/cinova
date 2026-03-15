package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/rs/zerolog/log"

	"github.com/foundrylab-app/cinova/backend/internal/auth"
	"github.com/foundrylab-app/cinova/backend/internal/graph"
	"github.com/foundrylab-app/cinova/backend/internal/models"
	"github.com/foundrylab-app/cinova/backend/internal/scoring"
)

// ScoringHandler handles GET/PUT /api/v1/me/scoring-profile.
type ScoringHandler struct {
	repo *graph.MovieRepository
}

// NewScoringHandler creates a ScoringHandler.
func NewScoringHandler(repo *graph.MovieRepository) *ScoringHandler {
	return &ScoringHandler{repo: repo}
}

// GetScoringProfile handles GET /api/v1/me/scoring-profile
func (h *ScoringHandler) GetScoringProfile(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromCtx(r.Context())
	if userID == "" {
		writeError(w, "unauthorized", "registered users only", http.StatusUnauthorized)
		return
	}

	profile, err := h.repo.GetScoringProfile(r.Context(), userID)
	if err != nil {
		log.Error().Err(err).Str("user_id", userID).Msg("GetScoringProfile")
		writeError(w, "internal_error", "failed to fetch scoring profile", http.StatusInternalServerError)
		return
	}

	// Enrich with resolved weights from scoring package
	weights := scoring.FromProfile(profile)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"preset":     profile.Preset,
		"audience":   weights.Audience,
		"critic":     weights.Critic,
		"award":      weights.Award,
		"prestige":   weights.Prestige,
		"commercial": weights.Commercial,
		"presets":    presetDescriptions(),
	})
}

// SetScoringProfile handles PUT /api/v1/me/scoring-profile
func (h *ScoringHandler) SetScoringProfile(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromCtx(r.Context())
	if userID == "" || auth.IsAnonymousFromCtx(r.Context()) {
		writeError(w, "unauthorized", "registered users only", http.StatusUnauthorized)
		return
	}

	var req struct {
		Preset     string  `json:"preset"`
		Audience   float64 `json:"audience"`
		Critic     float64 `json:"critic"`
		Award      float64 `json:"award"`
		Prestige   float64 `json:"prestige"`
		Commercial float64 `json:"commercial"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "bad_request", "invalid request body", http.StatusBadRequest)
		return
	}

	// If preset-only request, resolve weights from preset
	profile := models.ScoringProfile{
		Preset:     req.Preset,
		Audience:   req.Audience,
		Critic:     req.Critic,
		Award:      req.Award,
		Prestige:   req.Prestige,
		Commercial: req.Commercial,
	}

	// If weights not provided (all zero), use preset defaults
	sum := profile.Audience + profile.Critic + profile.Award + profile.Prestige + profile.Commercial
	if sum < 0.01 && profile.Preset != "" {
		w := scoring.PresetWeights(profile.Preset)
		profile.Audience = w.Audience
		profile.Critic = w.Critic
		profile.Award = w.Award
		profile.Prestige = w.Prestige
		profile.Commercial = w.Commercial
	}

	if err := h.repo.UpsertScoringProfile(r.Context(), userID, profile); err != nil {
		log.Error().Err(err).Str("user_id", userID).Msg("SetScoringProfile")
		writeError(w, "internal_error", "failed to save scoring profile", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"preset":     profile.Preset,
		"audience":   profile.Audience,
		"critic":     profile.Critic,
		"award":      profile.Award,
		"prestige":   profile.Prestige,
		"commercial": profile.Commercial,
	})
}

type presetDescription struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Emoji       string  `json:"emoji"`
	Description string  `json:"description"`
	Audience    float64 `json:"audience"`
	Critic      float64 `json:"critic"`
	Award       float64 `json:"award"`
	Prestige    float64 `json:"prestige"`
	Commercial  float64 `json:"commercial"`
}

func presetDescriptions() []presetDescription {
	return []presetDescription{
		{
			ID: "mainstream", Name: "Mainstream", Emoji: "🎯",
			Description: "Audience ratings drive recommendations",
			Audience: 0.50, Critic: 0.20, Award: 0.15, Prestige: 0.10, Commercial: 0.05,
		},
		{
			ID: "cinephile", Name: "Cinephile", Emoji: "🎬",
			Description: "Critics and awards matter most",
			Audience: 0.25, Critic: 0.35, Award: 0.25, Prestige: 0.15, Commercial: 0.00,
		},
		{
			ID: "arthouse", Name: "Arthouse", Emoji: "🎨",
			Description: "Director influence and auteur cinema",
			Audience: 0.20, Critic: 0.30, Award: 0.20, Prestige: 0.25, Commercial: 0.05,
		},
		{
			ID: "blockbuster", Name: "Blockbuster", Emoji: "💥",
			Description: "Commercial success and spectacle",
			Audience: 0.45, Critic: 0.15, Award: 0.10, Prestige: 0.05, Commercial: 0.25,
		},
		{
			ID: "award_season", Name: "Award Season", Emoji: "🏆",
			Description: "Oscars, BAFTAs, and film festivals",
			Audience: 0.20, Critic: 0.25, Award: 0.45, Prestige: 0.10, Commercial: 0.00,
		},
	}
}
