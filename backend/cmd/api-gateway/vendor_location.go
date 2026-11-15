package main

import (
	"errors"
	"math"
	"net/url"
	"strconv"
)

func parseVendorSearch(q url.Values) (float64, float64, float64, error) {
	lat, latErr := strconv.ParseFloat(q.Get("lat"), 64)
	lon, lonErr := strconv.ParseFloat(q.Get("lng"), 64)
	if latErr != nil || lonErr != nil || math.IsNaN(lat) || math.IsNaN(lon) || math.IsInf(lat, 0) || math.IsInf(lon, 0) || lat < -90 || lat > 90 || lon < -180 || lon > 180 {
		return 0, 0, 0, errors.New("valid latitude and longitude are required")
	}
	radius := 10000.0
	if raw := q.Get("radius"); raw != "" {
		var err error
		radius, err = strconv.ParseFloat(raw, 64)
		if err != nil || math.IsNaN(radius) || math.IsInf(radius, 0) || radius <= 0 {
			return 0, 0, 0, errors.New("radius must be positive")
		}
	}
	return lat, lon, radius, nil
}
