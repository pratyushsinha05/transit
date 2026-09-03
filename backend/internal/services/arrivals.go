package services

import (
	"context"
	"transit-backend/internal/models"
	"transit-backend/pkg/geo"
)

// ArrivalsService handles arrival predictions and ETA calculations
type ArrivalsService struct {
	stopRepo   StopRepository
	tripRepo   TripRepository
	locRepo    LocationRepository
	geoService *GeofencingService
}

// NewArrivalsService creates a new ArrivalsService
func NewArrivalsService(
	stopRepo StopRepository,
	tripRepo TripRepository,
	locRepo LocationRepository,
	geoService *GeofencingService,
) *ArrivalsService {
	return &ArrivalsService{
		stopRepo:   stopRepo,
		tripRepo:   tripRepo,
		locRepo:    locRepo,
		geoService: geoService,
	}
}

// ArrivalPrediction represents a predicted arrival with ETA
type ArrivalPrediction struct {
	TripID        string  `json:"trip_id"`
	DeviceID      string  `json:"device_id"`
	DeviceName    string  `json:"device_name"`
	ETAMinutes    int     `json:"eta_minutes"`
	Distance      float64 `json:"distance_km"`
	CurrentSpeed  float64 `json:"current_speed"`
	HexRes9       string  `json:"hex_res9,omitempty"`
	IsApproaching bool    `json:"is_approaching"`
}

// GetArrivalsForStop returns predicted arrivals for a stop using H3+PostGIS strategy
func (s *ArrivalsService) GetArrivalsForStop(ctx context.Context, stopID string) ([]ArrivalPrediction, error) {
	// 1. Get target stop details
	targetStop, err := s.stopRepo.GetByID(ctx, stopID)
	if err != nil {
		return nil, err
	}

	// 2. Get active trips before this stop
	trips, err := s.tripRepo.GetActiveTripsBeforeStop(ctx, targetStop.Sequence)
	if err != nil {
		return nil, err
	}

	// 3. Calculate ETA for each trip
	predictions := make([]ArrivalPrediction, 0, len(trips))
	for _, trip := range trips {
		// Calculate distance using Haversine
		distanceKm := geo.Haversine(trip.Latitude, trip.Longitude, targetStop.Latitude, targetStop.Longitude)

		// Calculate ETA
		etaMinutes := geo.CalculateETA(trip.Latitude, trip.Longitude, targetStop.Latitude, targetStop.Longitude, trip.Speed)

		// Check if bus is approaching (getting closer) using H3
		isApproaching := s.isApproaching(trip.Latitude, trip.Longitude, targetStop.Latitude, targetStop.Longitude)

		predictions = append(predictions, ArrivalPrediction{
			TripID:        trip.TripID,
			DeviceID:      trip.DeviceID,
			DeviceName:    trip.DeviceName,
			ETAMinutes:    etaMinutes,
			Distance:      distanceKm,
			CurrentSpeed:  trip.Speed,
			IsApproaching: isApproaching,
		})
	}

	return predictions, nil
}

// GetNearbyArrivals returns buses approaching any stop near a point
func (s *ArrivalsService) GetNearbyArrivals(ctx context.Context, lat, lng float64, radiusMeters int) ([]ArrivalPrediction, error) {
	// 1. Find nearby stops
	stops, err := s.geoService.FindNearbyStops(ctx, lat, lng, radiusMeters)
	if err != nil {
		return nil, err
	}

	// 2. Get arrivals for each stop
	var allPredictions []ArrivalPrediction
	seen := make(map[string]bool) // Deduplicate by trip ID

	for _, stop := range stops {
		predictions, err := s.GetArrivalsForStop(ctx, stop.ID)
		if err != nil {
			continue // Skip stops with errors
		}

		for _, p := range predictions {
			if !seen[p.TripID] {
				seen[p.TripID] = true
				allPredictions = append(allPredictions, p)
			}
		}
	}

	return allPredictions, nil
}

// isApproaching determines if a bus is approaching a stop
// Uses H3 hex comparison - if bus hex is in stop's k-ring, it's approaching
func (s *ArrivalsService) isApproaching(busLat, busLng, stopLat, stopLng float64) bool {
	// Get k-ring of 3 around stop (~500m at resolution 9)
	neighborHexes := s.geoService.GetNeighborHexes(stopLat, stopLng, 3)
	busHex := s.geoService.CalculateHex(busLat, busLng)

	if busHex == "" {
		return false
	}

	for _, hex := range neighborHexes {
		if hex == busHex {
			return true
		}
	}

	return false
}

// Phase4Reserved: reserved for Phase 4 geofence-event wiring (CLAUDE.md §2).
// Zero callers today. Do not delete — Phase 4 will wire this onto the request path.
//
// CalculateETAWithTraffic calculates ETA considering traffic conditions
// For now, this is a placeholder that uses default speed adjustment
func (s *ArrivalsService) CalculateETAWithTraffic(fromLat, fromLng, toLat, toLng, currentSpeed float64) int {
	// In a real implementation, this would:
	// 1. Query historical traffic data for the route
	// 2. Adjust speed based on time of day
	// 3. Consider road segments between points

	// For now, use simple calculation with 20% traffic buffer
	baseETA := geo.CalculateETA(fromLat, fromLng, toLat, toLng, currentSpeed)
	trafficAdjustedETA := int(float64(baseETA) * 1.2) // 20% buffer

	return trafficAdjustedETA
}

// Phase4Reserved: reserved for Phase 4 geofence-event wiring (CLAUDE.md §2).
// Zero callers today. Do not delete — Phase 4 will wire this onto the request path.
//
// DetectArrivalEvent checks if a bus has just arrived at a stop
func (s *ArrivalsService) DetectArrivalEvent(ctx context.Context, deviceID string, stop *models.Stop) (bool, error) {
	// Get current and previous location
	currentLoc, err := s.locRepo.GetLatestLocation(ctx, deviceID)
	if err != nil {
		return false, err
	}

	// For arrival detection, we need previous location
	// In production, this would come from cache or a separate query
	// For now, we just check if currently at stop
	isAtStop := s.geoService.IsAtStop(
		currentLoc.Latitude, currentLoc.Longitude,
		stop.Latitude, stop.Longitude,
	)

	return isAtStop, nil
}
