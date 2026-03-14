package scoring

const (
	// minVotes is the minimum votes required for a title to be included in
	// the Bayesian weighted average. Titles with fewer votes regress to the mean.
	minVotes = 1000

	// meanRating is the prior mean rating used in the Bayesian formula.
	// Computed from TMDB's global mean; approximated here as a constant.
	meanRating = 6.5

	// maxScore is the output range upper bound.
	maxScore = 100.0
)

// ComputeCinovaScore returns a 0–100 score that combines a Bayesian-weighted
// rating average with a normalised graph prestige signal (PageRank-like).
//
// Formula:
//
//	bayesian = (v*R + m*C) / (v + m)
//	   v = voteCount
//	   R = voteAverage  (0–10)
//	   m = minVotes     (1000)
//	   C = meanRating   (6.5)
//
//	score = ((bayesian / 10) * 0.8 + graphPrestige * 0.2) * 100
//
// graphPrestige is expected to be normalised in the range [0, 1].
// A prestige of 0 disables the graph signal entirely.
func ComputeCinovaScore(voteAverage float64, voteCount int, graphPrestige float64) float64 {
	v := float64(voteCount)
	r := voteAverage
	m := float64(minVotes)
	c := meanRating

	// Bayesian weighted average (result in 0–10 range)
	bayesian := (v*r + m*c) / (v + m)

	// Clamp graphPrestige to [0, 1]
	if graphPrestige < 0 {
		graphPrestige = 0
	}
	if graphPrestige > 1 {
		graphPrestige = 1
	}

	// Weighted combination: 80% rating signal, 20% graph prestige
	combined := (bayesian/10.0)*0.8 + graphPrestige*0.2

	score := combined * maxScore

	// Clamp to [0, 100]
	if score < 0 {
		return 0
	}
	if score > maxScore {
		return maxScore
	}
	return score
}
