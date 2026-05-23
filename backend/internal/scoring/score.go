package scoring

import (
	"math"
	"strings"

	"github.com/dynolabs-io/cinova/backend/internal/models"
)

const (
	// minVotes is the minimum votes required for the Bayesian prior to have low weight.
	minVotes = 1000

	// meanRating is the global mean used as the Bayesian prior (approximated from TMDB).
	meanRating = 6.5

	// maxScore is the output range upper bound.
	maxScore = 100.0
)

// ScoringWeights controls the relative importance of each signal.
// All weights should sum to 1.0.
type ScoringWeights struct {
	Audience   float64 // TMDB audience Bayesian rating signal
	Critic     float64 // RT% + Metacritic aggregation
	Award      float64 // awards & nominations prestige
	Prestige   float64 // director graph PageRank signal
	Commercial float64 // log(revenue/budget) ROI signal
}

// DefaultWeights returns the neutral balanced preset.
func DefaultWeights() ScoringWeights {
	return ScoringWeights{
		Audience:   0.40,
		Critic:     0.25,
		Award:      0.20,
		Prestige:   0.10,
		Commercial: 0.05,
	}
}

// Named preset profiles for per-user personalisation.
var (
	// WeightsMainstream — popular films, audience-driven.
	WeightsMainstream = ScoringWeights{0.50, 0.20, 0.15, 0.10, 0.05}
	// WeightsCinephile — critics and awards matter more.
	WeightsCinephile = ScoringWeights{0.25, 0.35, 0.25, 0.15, 0.00}
	// WeightsArthouse — auteur prestige, low commercial weight.
	WeightsArthouse = ScoringWeights{0.20, 0.30, 0.20, 0.25, 0.05}
	// WeightsBlockbuster — commercial success and popularity.
	WeightsBlockbuster = ScoringWeights{0.45, 0.15, 0.10, 0.05, 0.25}
	// WeightsAwardSeason — Oscar/BAFTA oriented.
	WeightsAwardSeason = ScoringWeights{0.20, 0.25, 0.45, 0.10, 0.00}
)

// PresetWeights returns named preset weights. Falls back to DefaultWeights.
func PresetWeights(preset string) ScoringWeights {
	switch strings.ToLower(preset) {
	case "mainstream":
		return WeightsMainstream
	case "cinephile":
		return WeightsCinephile
	case "arthouse":
		return WeightsArthouse
	case "blockbuster":
		return WeightsBlockbuster
	case "award_season", "awardsseason":
		return WeightsAwardSeason
	default:
		return DefaultWeights()
	}
}

// FromProfile converts a models.ScoringProfile to ScoringWeights.
// If the profile weights sum to ~0 (unset), returns the preset by name.
func FromProfile(p models.ScoringProfile) ScoringWeights {
	sum := p.Audience + p.Critic + p.Award + p.Prestige + p.Commercial
	if sum < 0.01 {
		return PresetWeights(p.Preset)
	}
	return ScoringWeights{
		Audience:   p.Audience,
		Critic:     p.Critic,
		Award:      p.Award,
		Prestige:   p.Prestige,
		Commercial: p.Commercial,
	}
}

// ScoreParams holds the raw signal inputs for one title.
type ScoreParams struct {
	VoteAverage   float64 // TMDB vote average 0–10
	VoteCount     int
	CriticScore   float64 // 0–1 avg of RT%/100 + Metacritic/100; set to -1 if unavailable
	AwardScore    float64 // 0–1 computed from Wikidata awards; 0.5 = neutral/unknown
	GraphPrestige float64 // 0–1 normalised director influence PageRank
	Budget        int64   // 0 = unknown
	Revenue       int64   // 0 = unknown
}

// ComputeFullScore returns a 0–100 score using configurable per-user weights.
//
// Signal breakdown:
//
//	audience   = Bayesian(voteAverage, voteCount) / 10
//	critic     = avg(RT%/100, metacritic/100)  [fallback: audience signal]
//	award      = tier-weighted award/nomination score [0.5 = unknown]
//	prestige   = normalised director PageRank
//	commercial = log10(revenue/budget) normalised to [0,1] [0.5 = unknown]
func ComputeFullScore(p ScoreParams, w ScoringWeights) float64 {
	// Audience: Bayesian weighted average
	v := float64(p.VoteCount)
	m := float64(minVotes)
	bayesian := (v*p.VoteAverage + m*meanRating) / (v + m)
	audienceSig := clamp01(bayesian / 10.0)

	// Critic: use critic score if provided, otherwise fall back to audience signal
	criticSig := audienceSig
	if p.CriticScore >= 0 {
		criticSig = clamp01(p.CriticScore)
	}

	// Award: already 0–1; 0.5 = neutral unknown
	awardSig := clamp01(p.AwardScore)

	// Prestige: already 0–1
	prestigeSig := clamp01(p.GraphPrestige)

	// Commercial: log10(revenue/budget) scaled to [0,1]
	// log10(0.01)=-2 → 0.0, log10(1)=0 → 0.5, log10(100)=2 → 1.0
	commercialSig := 0.5 // neutral when unknown
	if p.Budget > 0 && p.Revenue > 0 {
		roi := math.Log10(float64(p.Revenue) / float64(p.Budget))
		commercialSig = clamp01((roi + 2.0) / 4.0)
	}

	combined := audienceSig*w.Audience +
		criticSig*w.Critic +
		awardSig*w.Award +
		prestigeSig*w.Prestige +
		commercialSig*w.Commercial

	return clamp(combined*maxScore, 0, maxScore)
}

// ComputeCinovaScore is the legacy simplified formula used during initial ingestion
// before Wikidata critic/award data is available.
//
//	score = ((bayesian/10)*0.8 + graphPrestige*0.2) * 100
func ComputeCinovaScore(voteAverage float64, voteCount int, graphPrestige float64) float64 {
	return ComputeFullScore(ScoreParams{
		VoteAverage:   voteAverage,
		VoteCount:     voteCount,
		CriticScore:   -1,
		AwardScore:    0.5,
		GraphPrestige: clamp01(graphPrestige),
	}, ScoringWeights{
		Audience:   0.80,
		Critic:     0.00,
		Award:      0.00,
		Prestige:   0.20,
		Commercial: 0.00,
	})
}

// AwardTier returns the prestige weight [0,1] for a single named award.
// isNomination=true gives a lower tier than a win.
func AwardTier(awardName string, isNomination bool) float64 {
	type tier struct {
		sub     string
		winTier float64
		nomTier float64
	}
	tiers := []tier{
		{"academy award for best picture", 1.00, 0.60},
		{"academy award", 0.80, 0.45},
		{"oscar", 0.80, 0.45},
		{"bafta award for best film", 0.70, 0.45},
		{"bafta", 0.55, 0.30},
		{"palme d'or", 0.75, 0.40},
		{"golden lion", 0.70, 0.40},
		{"golden bear", 0.70, 0.40},
		{"grand jury prize", 0.45, 0.25},
		{"sundance", 0.35, 0.20},
		{"golden globe", 0.50, 0.30},
		{"screen actors guild", 0.40, 0.25},
		{"critics' choice", 0.35, 0.20},
		{"saturn award", 0.25, 0.15},
	}
	lower := strings.ToLower(awardName)
	for _, t := range tiers {
		if strings.Contains(lower, t.sub) {
			if isNomination {
				return t.nomTier
			}
			return t.winTier
		}
	}
	if isNomination {
		return 0.10
	}
	return 0.20
}

// ComputeAwardScore converts a list of Wikidata awards into a 0–1 signal.
// Returns 0.5 (neutral) when the slice is empty — meaning "data unavailable",
// not "no awards ever won".
func ComputeAwardScore(awards []models.Award) float64 {
	if len(awards) == 0 {
		return 0.5
	}
	var maxTier float64
	var winCount int
	for _, a := range awards {
		t := AwardTier(a.AwardName, a.IsNomination)
		if t > maxTier {
			maxTier = t
		}
		if !a.IsNomination {
			winCount++
		}
	}
	// Blend: prestige of best award (60%) + breadth bonus for multiple wins (40% cap)
	breadth := math.Min(float64(winCount)/10.0, 0.4)
	return math.Min(maxTier*0.6+breadth, 1.0)
}

// ---- helpers ----------------------------------------------------------------

func clamp01(v float64) float64 { return clamp(v, 0, 1) }

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
