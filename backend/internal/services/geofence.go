package services

import (
	"context"
	"transit-backend/internal/models"
	"transit-backend/pkg/geo"
)

// GeofenceService handles geofence event predictions and ETA calculations
type GeofenceService struct {
	zoneRepo   ZoneRepository
	tripRepo   TripRepository
	locRepo    LocationRepository
	geoService *GeofencingService
}

// NewGeofenceService creates a new GeofenceService
func NewGeofenceService(
	zoneRepo ZoneRepository,
	tripRepo TripRepository,
	locRepo LocationRepository,
	geoService *GeofencingService,
) *GeofenceService {
	return &GeofenceService{
		zoneRepo:   zoneRepo,
		tripRepo:   tripRepo,
		locRepo:    locRepo,
		geoService: geoService,
	}
}

// GeofencePrediction represents a predicted geofence arrival with ETA
type GeofencePrediction struct {
	TripID        string  `json:"trip_id"`
	DeviceID      string  `json:"device_id"`
	DeviceName    string  `json:"device_name"`
	ETAMinutes    int     `json:"eta_minutes"`
	Distance      float64 `json:"distance_km"`
	CurrentSpeed  float64 `json:"current_speed"`
	HexRes9       string  `json:"hex_res9,omitempty"`
	IsApproaching bool    `json:"is_approaching"`
}

// GetPredictionsForZone returns predicted arrivals for a zone using H3+PostGIS strategy
func (s *GeofenceService) GetPredictionsForZone(ctx context.Context, zoneID string) ([]GeofencePrediction, error) {
	// 1. Get target zone details
	targetZone, err := s.zoneRepo.GetByID(ctx, zoneID)
	if err != nil {
		return nil, err
	}

	// 2. Get active trips before this zone
	trips, err := s.tripRepo.GetActiveTripsBeforeZone(ctx, targetZone.Sequence)
	if err != nil {
		return nil, err
	}

	// 3. Calculate ETA for each trip
	predictions := make([]GeofencePrediction, 0, len(trips))
	for _, trip := range trips {
		// Calculate distance using Haversine
		distanceKm := geo.Haversine(trip.Latitude, trip.Longitude, targetZone.Latitude, targetZone.Longitude)

		// Calculate ETA
		etaMinutes := geo.CalculateETA(trip.Latitude, trip.Longitude, targetZone.Latitude, targetZone.Longitude, trip.Speed)

		// Check if device is approaching (getting closer) using H3
		isApproaching := s.isApproaching(trip.Latitude, trip.Longitude, targetZone.Latitude, targetZone.Longitude)

		predictions = append(predictions, GeofencePrediction{
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

// GetNearbyPredictions returns devices approaching any zone near a point
func (s *GeofenceService) GetNearbyPredictions(ctx context.Context, lat, lng float64, radiusMeters int) ([]GeofencePrediction, error) {
	// 1. Find nearby zones
	zones, err := s.geoService.FindNearbyZones(ctx, lat, lng, radiusMeters)
	if err != nil {
		return nil, err
	}

	// 2. Get predictions for each zone
	var allPredictions []GeofencePrediction
	seen := make(map[string]bool) // Deduplicate by trip ID

	for _, zone := range zones {
		predictions, err := s.GetPredictionsForZone(ctx, zone.ID)
		if err != nil {
			continue // Skip zones with errors
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

// isApproaching determines if a device is approaching a zone
// Uses H3 hex comparison - if device hex is in zone's k-ring, it's approaching
func (s *GeofenceService) isApproaching(deviceLat, deviceLng, zoneLat, zoneLng float64) bool {
	// Get k-ring of 3 around zone (~500m at resolution 9)
	neighborHexes := s.geoService.GetNeighborHexes(zoneLat, zoneLng, 3)
	deviceHex := s.geoService.CalculateHex(deviceLat, deviceLng)

	if deviceHex == "" {
		return false
	}

	for _, hex := range neighborHexes {
		if hex == deviceHex {
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
func (s *GeofenceService) CalculateETAWithTraffic(fromLat, fromLng, toLat, toLng, currentSpeed float64) int {
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
// DetectEntryEvent checks if a device has just entered a zone
func (s *GeofenceService) DetectEntryEvent(ctx context.Context, deviceID string, zone *models.Zone) (bool, error) {
	// Get current and previous location
	currentLoc, err := s.locRepo.GetLatestLocation(ctx, deviceID)
	if err != nil {
		return false, err
	}

	// For arrival detection, we need previous location
	// In production, this would come from cache or a separate query
	// For now, we just check if currently at zone
	isAtZone := s.geoService.IsAtZone(
		currentLoc.Latitude, currentLoc.Longitude,
		zone.Latitude, zone.Longitude,
	)

	return isAtZone, nil
}
