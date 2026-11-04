package geo

import (
	"context"
	"math"

	"github.com/purohit-platform/backend/internal/models"
	"github.com/purohit-platform/backend/internal/ranking"
)

const EarthRadiusMeters = 6371000.0

// HaversineDistance computes geodesic distance between two lat/long points in meters
func HaversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	dLat := (lat2 - lat1) * (math.Pi / 180.0)
	dLon := (lon2 - lon1) * (math.Pi / 180.0)

	rLat1 := lat1 * (math.Pi / 180.0)
	rLat2 := lat2 * (math.Pi / 180.0)

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Sin(dLon/2)*math.Sin(dLon/2)*math.Cos(rLat1)*math.Cos(rLat2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return EarthRadiusMeters * c
}

type DispatchEngine struct{}

func NewDispatchEngine() *DispatchEngine {
	return &DispatchEngine{}
}

// FindNearestPriests calculates distances from devotee coordinate, filters by radius, and scores candidates
func (e *DispatchEngine) FindNearestPriests(
	ctx context.Context,
	req models.DispatchRequest,
	activePriests []models.PriestCandidate,
	priestCoords map[string][2]float64,
) []models.PriestCandidate {
	var matchedCandidates []models.PriestCandidate

	for _, priest := range activePriests {
		// Calculate Haversine distance if coords available
		if coords, ok := priestCoords[priest.ID]; ok {
			priest.DistanceMeters = HaversineDistance(req.DevoteeLatitude, req.DevoteeLongitude, coords[0], coords[1])
		} else if priest.DistanceMeters == 0 {
			priest.DistanceMeters = 1500.0 // Default 1.5 km fallback
		}

		if priest.DistanceMeters <= req.SearchRadiusMeters {
			matchedCandidates = append(matchedCandidates, priest)
		}
	}

	// Apply dynamic Rapido-style ranking formula
	return ranking.RankCandidates(matchedCandidates, req.SearchRadiusMeters, req.RequiredSampradaya)
}
