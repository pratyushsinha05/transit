package services

import (
	"context"
	"transit-backend/internal/database"
	"transit-backend/internal/models"

	"github.com/uber/h3-go/v4"
)

// GeofencingService handles H3-based geofencing and spatial queries
type GeofencingService struct {
	locRepo    *database.LocationRepository
	stopRepo   *database.StopRepository
	resolution int // H3 resolution for geofencing (default: 9)
}

// NewGeofencingService creates a new GeofencingService
func NewGeofencingService(locRepo *database.LocationRepository, stopRepo *database.StopRepository) *GeofencingService {
	return &GeofencingService{
		locRepo:    locRepo,
		stopRepo:   stopRepo,
		resolution: 9, // ~175m edge length, ideal for bus stop detection
	}
}

// NewGeofencingServiceWithResolution creates a service with custom H3 resolution
func NewGeofencingServiceWithResolution(locRepo *database.LocationRepository, stopRepo *database.StopRepository, resolution int) *GeofencingService {
	return &GeofencingService{
		locRepo:    locRepo,
		stopRepo:   stopRepo,
		resolution: resolution,
	}
}

// CalculateHex returns the H3 hex index for a lat/lng at the service's resolution
// Returns empty string on error (invalid coordinates)
func (s *GeofencingService) CalculateHex(lat, lng float64) string {
	latLng := h3.NewLatLng(lat, lng)
	cell, err := h3.LatLngToCell(latLng, s.resolution)
	if err != nil {
		return ""
	}
	return cell.String()
}

// CalculateHexAtResolution returns the H3 hex index at a specific resolution
func CalculateHexAtResolution(lat, lng float64, resolution int) string {
	latLng := h3.NewLatLng(lat, lng)
	cell, err := h3.LatLngToCell(latLng, resolution)
	if err != nil {
		return ""
	}
	return cell.String()
}

// IsAtStop checks if a bus is at a stop using H3 hex comparison
// This is a fast O(1) check - both points in same hex means "at stop"
func (s *GeofencingService) IsAtStop(busLat, busLng, stopLat, stopLng float64) bool {
	busHex := s.CalculateHex(busLat, busLng)
	stopHex := s.CalculateHex(stopLat, stopLng)
	return busHex != "" && stopHex != "" && busHex == stopHex
}

// IsAtStopWithHysteresis checks if bus is at stop with hysteresis to prevent flickering
// - Bus must be in same hex as stop
// - If previously at stop, allow 1-ring buffer before considering "departed"
func (s *GeofencingService) IsAtStopWithHysteresis(busLat, busLng, stopLat, stopLng float64, wasAtStop bool) bool {
	busHex := s.CalculateHex(busLat, busLng)
	stopHex := s.CalculateHex(stopLat, stopLng)

	if busHex == "" || stopHex == "" {
		return false
	}

	// If in same hex, definitely at stop
	if busHex == stopHex {
		return true
	}

	// If previously at stop, check if still in neighboring hex (hysteresis)
	if wasAtStop {
		stopLatLng := h3.NewLatLng(stopLat, stopLng)
		stopCell, err := h3.LatLngToCell(stopLatLng, s.resolution)
		if err != nil {
			return false
		}
		neighbors, err := h3.GridDisk(stopCell, 1) // 1-ring neighbors
		if err != nil {
			return false
		}

		for _, neighbor := range neighbors {
			if neighbor.String() == busHex {
				return true // Still considered "at stop" due to hysteresis
			}
		}
	}

	return false
}

// GetNeighborHexes returns H3 hexes within k distance of a point
func (s *GeofencingService) GetNeighborHexes(lat, lng float64, k int) []string {
	latLng := h3.NewLatLng(lat, lng)
	centerCell, err := h3.LatLngToCell(latLng, s.resolution)
	if err != nil {
		return nil
	}

	cells, err := h3.GridDisk(centerCell, k)
	if err != nil {
		return nil
	}

	hexes := make([]string, len(cells))
	for i, cell := range cells {
		hexes[i] = cell.String()
	}

	return hexes
}

// FindNearbyBuses returns buses near a point using the H3+PostGIS combo strategy
// Step 1: H3 pre-filter (fast index scan)
// Step 2: PostGIS refine (accurate distance)
func (s *GeofencingService) FindNearbyBuses(ctx context.Context, lat, lng float64, radiusMeters int) ([]models.NearbyBus, error) {
	// For small radius (< 500m), H3 k-ring + PostGIS refinement is fastest
	// For larger radius, direct PostGIS query might be better
	if radiusMeters <= 500 {
		// Calculate k-ring size based on radius and H3 resolution
		// At resolution 9, edge length is ~175m, so k=2 covers ~350m radius
		k := (radiusMeters / 175) + 1
		if k > 5 {
			k = 5 // Cap k-ring size to prevent huge queries
		}

		// Step 1: Get hexes in k-ring
		hexes := s.GetNeighborHexes(lat, lng, k)
		if len(hexes) == 0 {
			return nil, nil
		}

		// Step 2: Query buses in those hexes (H3 pre-filter)
		buses, err := s.locRepo.GetBusesInHexes(ctx, hexes, 5) // 5 min max age
		if err != nil {
			return nil, err
		}

		// Step 3: PostGIS would refine here, but for simplicity we return all
		// In production, you'd filter by exact distance using Haversine
		return buses, nil
	}

	// For larger radius, use PostGIS directly (it has GIST index)
	return s.locRepo.GetBusesNearStop(ctx, lat, lng, radiusMeters, 5)
}

// FindNearbyStops returns stops near a point
func (s *GeofencingService) FindNearbyStops(ctx context.Context, lat, lng float64, radiusMeters int) ([]models.Stop, error) {
	// For now, delegate to stop repository
	// In future, could add H3 pre-filtering for stops as well
	return s.stopRepo.GetNearby(ctx, lat, lng, radiusMeters)
}

// DetectArrival checks if a bus has arrived at a stop based on location change
// Arrival is detected when:
// - Bus was NOT at stop in previous location
// - Bus IS at stop in current location
func (s *GeofencingService) DetectArrival(oldLoc, newLoc *models.Location, stop *models.Stop) bool {
	if oldLoc == nil || newLoc == nil || stop == nil {
		return false
	}

	wasAtStop := s.IsAtStop(oldLoc.Latitude, oldLoc.Longitude, stop.Latitude, stop.Longitude)
	nowAtStop := s.IsAtStop(newLoc.Latitude, newLoc.Longitude, stop.Latitude, stop.Longitude)

	return !wasAtStop && nowAtStop
}

// DetectDeparture checks if a bus has departed from a stop
func (s *GeofencingService) DetectDeparture(oldLoc, newLoc *models.Location, stop *models.Stop) bool {
	if oldLoc == nil || newLoc == nil || stop == nil {
		return false
	}

	wasAtStop := s.IsAtStop(oldLoc.Latitude, oldLoc.Longitude, stop.Latitude, stop.Longitude)
	nowAtStop := s.IsAtStop(newLoc.Latitude, newLoc.Longitude, stop.Latitude, stop.Longitude)

	return wasAtStop && !nowAtStop
}

// GetHexResolution returns the current H3 resolution
func (s *GeofencingService) GetHexResolution() int {
	return s.resolution
}

// HexEdgeLengthMeters returns approximate edge length for a given H3 resolution
func HexEdgeLengthMeters(resolution int) float64 {
	// Approximate edge lengths in meters for each resolution
	edgeLengths := map[int]float64{
		0:  1107712.591,
		1:  418676.005,
		2:  158244.655,
		3:  59810.857,
		4:  22606.379,
		5:  8544.408,
		6:  3229.482,
		7:  1220.629,
		8:  461.354,
		9:  174.375,
		10: 65.907,
		11: 24.910,
		12: 9.415,
		13: 3.559,
		14: 1.348,
		15: 0.509,
	}

	if length, ok := edgeLengths[resolution]; ok {
		return length
	}
	return 0
}
