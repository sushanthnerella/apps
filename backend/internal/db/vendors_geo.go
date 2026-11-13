package db

import (
	"fmt"
	"math"

	"github.com/purohit-platform/backend/internal/geo"
	"github.com/purohit-platform/backend/internal/models"
)

// GetNearbyVendors finds open pooja stores ranked by distance.
func (s *Storage) GetNearbyVendors(lat, lon, radiusMeters float64) ([]models.Vendor, error) {
	query := `
		SELECT id, shop_name, owner_name, phone, COALESCE(profile_picture_url, ''), 
		       address, city, latitude, longitude, rating_avg, is_open 
		FROM vendors 
		WHERE is_open = TRUE
	`
	rows, err := s.DB.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query vendors: %w", err)
	}
	defer rows.Close()

	var vendors []models.Vendor
	for rows.Next() {
		var v models.Vendor
		if err := rows.Scan(
			&v.ID, &v.ShopName, &v.OwnerName, &v.Phone, &v.ProfilePictureURL,
			&v.Address, &v.City, &v.Latitude, &v.Longitude, &v.RatingAvg, &v.IsOpen,
		); err != nil {
			continue
		}

		distMeters := geo.HaversineDistance(lat, lon, v.Latitude, v.Longitude)
		if distMeters <= radiusMeters {
			v.DistanceKm = math.Round((distMeters/1000.0)*10) / 10
			vendors = append(vendors, v)
		}
	}
	return vendors, nil
}
