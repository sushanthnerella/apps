package db

import (
	"fmt"
	"github.com/lib/pq"
	"github.com/purohit-platform/backend/internal/geo"
	"github.com/purohit-platform/backend/internal/models"
	"math"
	"sort"
	"strings"
)

// GetNearbyPriests finds all active priests around devotee location with real distance and ETA
func (s *Storage) GetNearbyPriests(devLat, devLon, radiusMeters float64, serviceFilter string) ([]models.NearbyPriest, error) {
	if s == nil || s.DB == nil {
		return nil, nil
	}
	if math.IsNaN(devLat) || math.IsNaN(devLon) || math.IsNaN(radiusMeters) || math.IsInf(radiusMeters, 0) || devLat < -90 || devLat > 90 || devLon < -180 || devLon > 180 {
		return nil, fmt.Errorf("invalid discovery coordinates or radius")
	}
	if radiusMeters > 100000 {
		radiusMeters = 100000
	}
	if radiusMeters <= 0 {
		radiusMeters = 50000.0 // 50km default radius for metro region
	}

	query := `
		SELECT id, full_name, phone, sampradaya, veda_shakha, rating_avg, total_completed_poojas, repeat_booking_count,
		       latitude, longitude, eta_minutes, vehicle_type, services_offered, COALESCE(acceptance_rate, 0.5)
		FROM priests
		WHERE is_online = true AND location_confirmed = true
        AND ST_DWithin(ST_SetSRID(ST_MakePoint(longitude, latitude),4326)::geography,
            ST_SetSRID(ST_MakePoint($2, $1),4326)::geography, $3);
	`
	var hasPostGIS bool
	if err := s.DB.QueryRow("SELECT EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'postgis')").Scan(&hasPostGIS); err != nil {
		return nil, err
	}
	if !hasPostGIS {
		query = strings.Replace(query,
			"AND ST_DWithin(ST_SetSRID(ST_MakePoint(longitude, latitude),4326)::geography,\n            ST_SetSRID(ST_MakePoint($2, $1),4326)::geography, $3)",
			`AND latitude BETWEEN $1 - ($3 / 110000.0) AND $1 + ($3 / 110000.0)
            AND 6371000 * 2 * asin(sqrt(LEAST(1.0,
              power(sin(radians(latitude - $1)/2),2) + cos(radians($1))*cos(radians(latitude))*power(sin(radians(longitude - $2)/2),2)))) <= $3`, 1)
	}
	rows, err := s.DB.Query(query, devLat, devLon, radiusMeters)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	allPriests := make([]models.NearbyPriest, 0)
	filteredPriests := make([]models.NearbyPriest, 0)

	for rows.Next() {
		var p models.NearbyPriest
		var services pq.StringArray
		err := rows.Scan(
			&p.ID, &p.FullName, &p.Phone, &p.Sampradaya, &p.VedaShakha,
			&p.RatingAvg, &p.TotalCompletedPoojas, &p.RepeatBookingCount,
			&p.Latitude, &p.Longitude, &p.ETAMinutes, &p.VehicleType, &services,
			&p.AcceptanceRate,
		)
		if err != nil {
			return nil, err
		}
		// Default to a neutral 0.5 acceptance rate for new priests without historical dispatch logs
		if p.AcceptanceRate <= 0.0 {
			p.AcceptanceRate = 0.5
		}
		p.ServicesOffered = []string(services)

		// Compute Haversine Geodesic Distance
		dist := geo.HaversineDistance(devLat, devLon, p.Latitude, p.Longitude)
		p.DistanceMeters = math.Round(dist)
		p.DistanceKm = math.Round((dist/1000.0)*10) / 10

		// Strictly filter priests within the requested radius
		if p.DistanceMeters > radiusMeters {
			continue
		}

		// Update ETA based on transit distance (~2.5 mins per km)
		computedETA := int(math.Ceil(p.DistanceKm * 3.0))
		if computedETA < 4 {
			computedETA = 4
		}
		p.ETAMinutes = computedETA

		// Quick Rapido composite score
		distRatio := p.DistanceMeters / radiusMeters
		if distRatio > 1.0 {
			distRatio = 1.0
		}
		p.Score = math.Round((0.40*(1.0-distRatio)+0.35*(p.RatingAvg/5.0)+0.25*(float64(p.RepeatBookingCount)/50.0))*100) / 100

		allPriests = append(allPriests, p)

		// Filter by service if specified (supports multi-service selection separated by + or ,)
		if serviceFilter != "" {
			filterParts := strings.FieldsFunc(serviceFilter, func(r rune) bool {
				return r == '+' || r == ','
			})
			matches := false
			for _, part := range filterParts {
				trimmed := strings.TrimSpace(part)
				if trimmed == "" {
					continue
				}
				for _, srv := range p.ServicesOffered {
					if strings.Contains(strings.ToLower(srv), strings.ToLower(trimmed)) ||
						strings.Contains(strings.ToLower(trimmed), strings.ToLower(srv)) {
						matches = true
						break
					}
				}
				if matches {
					break
				}
			}
			if matches {
				filteredPriests = append(filteredPriests, p)
			}
		}
	}

	result := filteredPriests
	if serviceFilter == "" {
		result = allPriests
	}

	// Sort nearest first (like Rapido shows nearest captains)
	sort.Slice(result, func(i, j int) bool {
		return result[i].DistanceMeters < result[j].DistanceMeters
	})

	return result, rows.Err()
}
