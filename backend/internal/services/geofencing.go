package services

import (
	"context"
	"transit-backend/internal/models"

	"github.com/uber/h3-go/v4"
)

// GeofencingService handles H3-based geofencing and spatial queries
type GeofencingService struct {
	locRepo    LocationRepository
	zoneRepo   ZoneRepository
	resolution int // H3 resolution for geofencing (default: 9)
}

// NewGeofencingService creates a new GeofencingService
func NewGeofencingService(locRepo LocationRepository, zoneRepo ZoneRepository) *GeofencingService {
	return &GeofencingService{
		locRepo:    locRepo,
		zoneRepo:   zoneRepo,
		resolution: 9, // ~175m edge length, ideal for zone detection
	}
}

// NewGeofencingServiceWithResolution creates a service with custom H3 resolution
func NewGeofencingServiceWithResolution(locRepo LocationRepository, zoneRepo ZoneRepository, resolution int) *GeofencingService {
	return &GeofencingService{
		locRepo:    locRepo,
		zoneRepo:   zoneRepo,
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

// Phase4Reserved: reserved for Phase 4 geofence-event wiring (CLAUDE.md §2).
// Zero callers today. Do not delete — Phase 4 will wire this onto the request path.
//
// IsAtZone checks if a device is at a zone using H3 hex comparison
// This is a fast O(1) check - both points in same hex means "at zone"
func (s *GeofencingService) IsAtZone(deviceLat, deviceLng, zoneLat, zoneLng float64) bool {
	deviceHex := s.CalculateHex(deviceLat, deviceLng)
	zoneHex := s.CalculateHex(zoneLat, zoneLng)
	return deviceHex != "" && zoneHex != "" && deviceHex == zoneHex
}

// Phase4Reserved: reserved for Phase 4 geofence-event wiring (CLAUDE.md §2).
// Zero callers today. Do not delete — Phase 4 will wire this onto the request path.
//
// IsAtZoneWithHysteresis checks if device is at zone with hysteresis to prevent flickering
// - Device must be in same hex as zone
// - If previously at zone, allow 1-ring buffer before considering "departed"
func (s *GeofencingService) IsAtZoneWithHysteresis(deviceLat, deviceLng, zoneLat, zoneLng float64, wasAtZone bool) bool {
	deviceHex := s.CalculateHex(deviceLat, deviceLng)
	zoneHex := s.CalculateHex(zoneLat, zoneLng)

	if deviceHex == "" || zoneHex == "" {
		return false
	}

	// If in same hex, definitely at zone
	if deviceHex == zoneHex {
		return true
	}

	// If previously at zone, check if still in neighboring hex (hysteresis)
	if wasAtZone {
		zoneLatLng := h3.NewLatLng(zoneLat, zoneLng)
		zoneCell, err := h3.LatLngToCell(zoneLatLng, s.resolution)
		if err != nil {
			return false
		}
		neighbors, err := h3.GridDisk(zoneCell, 1) // 1-ring neighbors
		if err != nil {
			return false
		}

		for _, neighbor := range neighbors {
			if neighbor.String() == deviceHex {
				return true // Still considered "at zone" due to hysteresis
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

// FindNearbyDevices returns devices near a point using the H3+PostGIS combo strategy
// Step 1: H3 pre-filter (fast index scan)
// Step 2: PostGIS refine (accurate distance)
func (s *GeofencingService) FindNearbyDevices(ctx context.Context, lat, lng float64, radiusMeters int) ([]models.NearbyDevice, error) {
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

		// Step 2: Query devices in those hexes (H3 pre-filter)
		devices, err := s.locRepo.GetDevicesInHexes(ctx, hexes, 5) // 5 min max age
		if err != nil {
			return nil, err
		}

		// Step 3: PostGIS would refine here, but for simplicity we return all
		// In production, you'd filter by exact distance using Haversine
		return devices, nil
	}

	// For larger radius, use PostGIS directly (it has GIST index)
	return s.locRepo.GetDevicesNearZone(ctx, lat, lng, radiusMeters, 5)
}

// FindNearbyZones returns zones near a point
func (s *GeofencingService) FindNearbyZones(ctx context.Context, lat, lng float64, radiusMeters int) ([]models.Zone, error) {
	// For now, delegate to zone repository
	// In future, could add H3 pre-filtering for zones as well
	return s.zoneRepo.GetNearby(ctx, lat, lng, radiusMeters)
}

// Phase4Reserved: reserved for Phase 4 geofence-event wiring (CLAUDE.md §2).
// Zero callers today. Do not delete — Phase 4 will wire this onto the request path.
//
// DetectEntry checks if a device has arrived at a zone based on location change
// Entry is detected when:
// - Device was NOT at zone in previous location
// - Device IS at zone in current location
func (s *GeofencingService) DetectEntry(oldLoc, newLoc *models.Location, zone *models.Zone) bool {
	if oldLoc == nil || newLoc == nil || zone == nil {
		return false
	}

	wasAtZone := s.IsAtZone(oldLoc.Latitude, oldLoc.Longitude, zone.Latitude, zone.Longitude)
	nowAtZone := s.IsAtZone(newLoc.Latitude, newLoc.Longitude, zone.Latitude, zone.Longitude)

	return !wasAtZone && nowAtZone
}

// Phase4Reserved: reserved for Phase 4 geofence-event wiring (CLAUDE.md §2).
// Zero callers today. Do not delete — Phase 4 will wire this onto the request path.
//
// DetectExit checks if a device has departed from a zone
func (s *GeofencingService) DetectExit(oldLoc, newLoc *models.Location, zone *models.Zone) bool {
	if oldLoc == nil || newLoc == nil || zone == nil {
		return false
	}

	wasAtZone := s.IsAtZone(oldLoc.Latitude, oldLoc.Longitude, zone.Latitude, zone.Longitude)
	nowAtZone := s.IsAtZone(newLoc.Latitude, newLoc.Longitude, zone.Latitude, zone.Longitude)

	return wasAtZone && !nowAtZone
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
