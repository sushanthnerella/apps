package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/purohit-platform/backend/internal/geo"
	"github.com/purohit-platform/backend/internal/models"
)

func main() {
	dispatchEngine := geo.NewDispatchEngine()

	// Mock active online priests in vicinity
	mockActivePriests := []models.PriestCandidate{
		{
			ID:                   "priest_101",
			FullName:             "Brahmasri Venkata Sharma Garu",
			Phone:                "+919876543210",
			Sampradaya:           "Smartha",
			RatingAvg:            4.95,
			TotalCompletedPoojas: 184,
			RepeatBookingCount:   42,
			AcceptanceRate:       0.98,
			DistanceMeters:       1800.0, // 1.8 km
		},
		{
			ID:                   "priest_102",
			FullName:             "Purohit Narayana Acharya",
			Phone:                "+919876543211",
			Sampradaya:           "Vaishnava",
			RatingAvg:            4.70,
			TotalCompletedPoojas: 96,
			RepeatBookingCount:   18,
			AcceptanceRate:       0.85,
			DistanceMeters:       900.0, // 0.9 km (Closer, but fewer repeats)
		},
		{
			ID:                   "priest_103",
			FullName:             "Vedam Subrahmanya Sastry",
			Phone:                "+919876543212",
			Sampradaya:           "Smartha",
			RatingAvg:            4.85,
			TotalCompletedPoojas: 210,
			RepeatBookingCount:   55,
			AcceptanceRate:       0.92,
			DistanceMeters:       3400.0, // 3.4 km
		},
	}

	mux := http.NewServeMux()

	// Rapido-Style Match Endpoint
	mux.HandleFunc("/api/v1/dispatch/match", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		var req models.DispatchRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			// Fallback defaults
			req = models.DispatchRequest{
				BookingID:          "booking_test_001",
				PoojaTitle:         "Gruhapravesam",
				DevoteeLatitude:    17.3850,
				DevoteeLongitude:   78.4867,
				RequiredSampradaya: "Smartha",
				SearchRadiusMeters: 5000.0, // 5 km
			}
		}

		rankedPriests := dispatchEngine.FindNearestPriests(context.Background(), req, mockActivePriests, nil)

		json.NewEncoder(w).Encode(map[string]interface{}{
			"booking_id":       req.BookingID,
			"total_candidates": len(rankedPriests),
			"best_match":       rankedPriests[0],
			"candidates":       rankedPriests,
		})
	})

	port := ":8081"
	fmt.Printf("⚡ Puja Vaahini Dispatch Service listening on port %s\n", port)
	log.Fatal(http.ListenAndServe(port, mux))
}
