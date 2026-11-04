package ranking

import (
	"math"
	"sort"
	"strings"

	"github.com/purohit-platform/backend/internal/models"
)

const (
	WeightDistance   = 0.35
	WeightRating     = 0.25
	WeightRepeat     = 0.20
	WeightAcceptance = 0.10
	WeightSampradaya = 0.10

	RepeatThresholdCeiling = 50.0 // 50+ repeats is max score for repeat factor
)

// CalculateScore computes the composite Rapido-style ranking score for a priest candidate
func CalculateScore(candidate *models.PriestCandidate, maxRadiusMeters float64, preferredSampradaya string) float64 {
	// 1. Normalized Distance Score (Closer = Higher score, 1.0 at 0m, 0.0 at maxRadius)
	distRatio := candidate.DistanceMeters / maxRadiusMeters
	if distRatio > 1.0 {
		distRatio = 1.0
	}
	distanceScore := 1.0 - distRatio

	// 2. Normalized Rating Score (out of 5.0)
	ratingScore := candidate.RatingAvg / 5.0
	if ratingScore > 1.0 {
		ratingScore = 1.0
	}

	// 3. Repeat Booking Frequency Score
	repeatScore := float64(candidate.RepeatBookingCount) / RepeatThresholdCeiling
	if repeatScore > 1.0 {
		repeatScore = 1.0
	}

	// 4. Acceptance Rate Score
	acceptanceScore := candidate.AcceptanceRate
	if acceptanceScore > 1.0 {
		acceptanceScore = 1.0
	}

	// 5. Sampradaya / Tradition Match Score
	sampradayaScore := 0.5 // Default baseline for qualified Vedic priest
	if strings.EqualFold(candidate.Sampradaya, preferredSampradaya) {
		sampradayaScore = 1.0
	}

	// Composite Weighted Sum
	totalScore := (WeightDistance * distanceScore) +
		(WeightRating * ratingScore) +
		(WeightRepeat * repeatScore) +
		(WeightAcceptance * acceptanceScore) +
		(WeightSampradaya * sampradayaScore)

	return math.Round(totalScore*1000) / 1000
}

// RankCandidates scores and sorts priests descending by score
func RankCandidates(candidates []models.PriestCandidate, maxRadiusMeters float64, preferredSampradaya string) []models.PriestCandidate {
	for i := range candidates {
		candidates[i].Score = CalculateScore(&candidates[i], maxRadiusMeters, preferredSampradaya)
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Score > candidates[j].Score
	})

	return candidates
}
