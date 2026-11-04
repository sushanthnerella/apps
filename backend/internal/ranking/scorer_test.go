package ranking

import (
	"testing"

	"github.com/purohit-platform/backend/internal/models"
)

func TestCompositeScoringPrefersHighRatingAndRepeats(t *testing.T) {
	priestHighQuality := models.PriestCandidate{
		ID:                   "p1",
		FullName:             "Brahmasri Venkata Sharma",
		DistanceMeters:       1800.0,
		RatingAvg:            4.95,
		RepeatBookingCount:   45,
		AcceptanceRate:       0.98,
		Sampradaya:           "Smartha",
	}

	priestCloserLowQuality := models.PriestCandidate{
		ID:                   "p2",
		FullName:             "Purohit Narayana",
		DistanceMeters:       800.0, // Closer by 1000m
		RatingAvg:            4.10,
		RepeatBookingCount:   3,
		AcceptanceRate:       0.70,
		Sampradaya:           "Vaishnava",
	}

	score1 := CalculateScore(&priestHighQuality, 5000.0, "Smartha")
	score2 := CalculateScore(&priestCloserLowQuality, 5000.0, "Smartha")

	if score1 <= score2 {
		t.Fatalf("Expected priest with high repeats and ratings to outscore closer priest, got %f vs %f", score1, score2)
	}
}

func TestRankCandidatesOrdering(t *testing.T) {
	candidates := []models.PriestCandidate{
		{
			ID:                 "p2",
			FullName:           "Candidate Low",
			DistanceMeters:     4500.0,
			RatingAvg:          4.0,
			RepeatBookingCount: 1,
			AcceptanceRate:     0.6,
			Sampradaya:         "Madhwa",
		},
		{
			ID:                 "p1",
			FullName:           "Candidate High",
			DistanceMeters:     500.0,
			RatingAvg:          4.9,
			RepeatBookingCount: 40,
			AcceptanceRate:     0.95,
			Sampradaya:         "Smartha",
		},
	}

	ranked := RankCandidates(candidates, 5000.0, "Smartha")
	if ranked[0].ID != "p1" {
		t.Fatalf("Expected ranked[0] to be p1, got %s", ranked[0].ID)
	}
	if ranked[0].Score <= ranked[1].Score {
		t.Fatalf("Expected ranked[0].Score (%f) > ranked[1].Score (%f)", ranked[0].Score, ranked[1].Score)
	}
}
