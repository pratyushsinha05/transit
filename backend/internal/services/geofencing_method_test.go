package services

import (
	"context"
	"fmt"
	"testing"
	"transit-backend/internal/models"

	"github.com/uber/h3-go/v4"
)

// mockLocationRepo is a test double for LocationRepository.
type mockLocationRepo struct {
	insertFn             func(ctx context.Context, loc *models.Location) error
	getLatestFn          func(ctx context.Context, deviceID string) (*models.Location, error)
	getDevicesInHexesFn  func(ctx context.Context, hexes []string, maxAgeMinutes int) ([]models.NearbyDevice, error)
	getDevicesNearZoneFn func(ctx context.Context, lat, lng float64, radiusMeters int, maxAgeMinutes int) ([]models.NearbyDevice, error)
}

func (m *mockLocationRepo) Insert(ctx context.Context, loc *models.Location) error {
	if m.insertFn != nil {
		return m.insertFn(ctx, loc)
	}
	return nil
}

func (m *mockLocationRepo) GetLatestLocation(ctx context.Context, deviceID string) (*models.Location, error) {
	if m.getLatestFn != nil {
		return m.getLatestFn(ctx, deviceID)
	}
	return nil, nil
}

func (m *mockLocationRepo) GetDevicesInHexes(ctx context.Context, hexes []string, maxAgeMinutes int) ([]models.NearbyDevice, error) {
	if m.getDevicesInHexesFn != nil {
		return m.getDevicesInHexesFn(ctx, hexes, maxAgeMinutes)
	}
	return nil, nil
}

func (m *mockLocationRepo) GetDevicesNearZone(ctx context.Context, lat, lng float64, radiusMeters int, maxAgeMinutes int) ([]models.NearbyDevice, error) {
	if m.getDevicesNearZoneFn != nil {
		return m.getDevicesNearZoneFn(ctx, lat, lng, radiusMeters, maxAgeMinutes)
	}
	return nil, nil
}

// mockZoneRepo is a test double for ZoneRepository.
type mockZoneRepo struct {
	getByIDFn      func(ctx context.Context, stopID string) (*models.Zone, error)
	getByRouteIDFn func(ctx context.Context, routeID string) ([]models.Zone, error)
	getNearbyFn    func(ctx context.Context, lat, lng float64, radiusMeters int) ([]models.Zone, error)
}

func (m *mockZoneRepo) GetByID(ctx context.Context, stopID string) (*models.Zone, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, stopID)
	}
	return nil, nil
}

func (m *mockZoneRepo) GetByRouteID(ctx context.Context, routeID string) ([]models.Zone, error) {
	if m.getByRouteIDFn != nil {
		return m.getByRouteIDFn(ctx, routeID)
	}
	return nil, nil
}

func (m *mockZoneRepo) GetNearby(ctx context.Context, lat, lng float64, radiusMeters int) ([]models.Zone, error) {
	if m.getNearbyFn != nil {
		return m.getNearbyFn(ctx, lat, lng, radiusMeters)
	}
	return nil, nil
}

func TestGeofencingService_CalculateHex(t *testing.T) {
	svc := NewGeofencingService(nil, nil)
	if svc.GetHexResolution() != 9 {
		t.Errorf("expected default resolution 9, got %d", svc.GetHexResolution())
	}

	tests := []struct {
		name      string
		lat       float64
		lng       float64
		wantEmpty bool
	}{
		{"Delhi Connaught Place", 28.6139, 77.2090, false},
		{"New York Times Square", 40.7580, -73.9855, false},
		{"Equator Origin", 0.0, 0.0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hex := svc.CalculateHex(tt.lat, tt.lng)
			if tt.wantEmpty && hex != "" {
				t.Errorf("CalculateHex(%v, %v) = %q, want empty", tt.lat, tt.lng, hex)
			}
			if !tt.wantEmpty && hex == "" {
				t.Errorf("CalculateHex(%v, %v) = empty, want non-empty", tt.lat, tt.lng)
			}
		})
	}

	// Invalid resolution should return empty string
	badSvc := NewGeofencingServiceWithResolution(nil, nil, -1)
	if hex := badSvc.CalculateHex(28.6139, 77.2090); hex != "" {
		t.Errorf("CalculateHex with invalid resolution = %q, want empty", hex)
	}
}

func TestGeofencingService_IsAtZone(t *testing.T) {
	svc := NewGeofencingServiceWithResolution(nil, nil, 9)

	// Connaught Place coordinates
	zoneLat, zoneLng := 28.6139, 77.2090

	// Same coordinates -> same hex -> at zone
	if !svc.IsAtZone(zoneLat, zoneLng, zoneLat, zoneLng) {
		t.Error("expected same coordinates to be at zone")
	}

	// 15 meters away -> still within ~175m edge hex
	if !svc.IsAtZone(28.61395, 77.20905, zoneLat, zoneLng) {
		t.Error("expected 15m away point to be in same hex at resolution 9")
	}

	// ~1.5 km away -> different hex
	if svc.IsAtZone(28.6270, 77.2190, zoneLat, zoneLng) {
		t.Error("expected point 1.5km away to NOT be at zone")
	}

	// Service with invalid resolution returns empty hexes, so IsAtZone returns false
	badSvc := NewGeofencingServiceWithResolution(nil, nil, -1)
	if badSvc.IsAtZone(zoneLat, zoneLng, zoneLat, zoneLng) {
		t.Error("expected invalid resolution to return false")
	}
}

func TestGeofencingService_IsAtZoneWithHysteresis(t *testing.T) {
	svc := NewGeofencingServiceWithResolution(nil, nil, 9)
	zoneLat, zoneLng := 28.6139, 77.2090
	zoneCell, _ := h3.LatLngToCell(h3.NewLatLng(zoneLat, zoneLng), 9)

	// Find an exact 1-ring neighbor cell
	neighbors, err := h3.GridDisk(zoneCell, 1)
	if err != nil || len(neighbors) < 2 {
		t.Fatalf("failed to get 1-ring neighbors: %v", err)
	}

	// Pick a neighbor cell that is not the center cell itself
	var neighborCell h3.Cell
	for _, n := range neighbors {
		if n != zoneCell {
			neighborCell = n
			break
		}
	}
	neighborLL, _ := h3.CellToLatLng(neighborCell)
	neighborLat, neighborLng := neighborLL.Lat, neighborLL.Lng

	// Case 1: Same hex, wasAtZone = false -> true
	if !svc.IsAtZoneWithHysteresis(zoneLat, zoneLng, zoneLat, zoneLng, false) {
		t.Error("same hex must return true regardless of wasAtZone")
	}

	// Case 2: In neighboring hex, wasAtZone = true -> true (hysteresis buffer holds)
	if !svc.IsAtZoneWithHysteresis(neighborLat, neighborLng, zoneLat, zoneLng, true) {
		t.Errorf("neighbor hex (%s) with wasAtZone=true must return true under hysteresis", neighborCell.String())
	}

	// Case 3: In neighboring hex, wasAtZone = false -> false (no hysteresis buffer for new arrivals)
	if svc.IsAtZoneWithHysteresis(neighborLat, neighborLng, zoneLat, zoneLng, false) {
		t.Errorf("neighbor hex (%s) with wasAtZone=false must return false", neighborCell.String())
	}

	// Case 4: Far away (~5km), wasAtZone = true -> false (outside 1-ring buffer)
	farLat, farLng := 28.6600, 77.2600
	if svc.IsAtZoneWithHysteresis(farLat, farLng, zoneLat, zoneLng, true) {
		t.Error("far away point must return false even with wasAtZone=true")
	}

	// Case 5: Invalid resolution -> false
	badSvc := NewGeofencingServiceWithResolution(nil, nil, -1)
	if badSvc.IsAtZoneWithHysteresis(zoneLat, zoneLng, zoneLat, zoneLng, true) {
		t.Error("invalid resolution must return false")
	}
}

func TestGeofencingService_GetNeighborHexes(t *testing.T) {
	svc := NewGeofencingServiceWithResolution(nil, nil, 9)

	// k=0: only center cell (1)
	hexes0 := svc.GetNeighborHexes(28.6139, 77.2090, 0)
	if len(hexes0) != 1 {
		t.Errorf("k=0 expected 1 hex, got %d", len(hexes0))
	}

	// k=1: center + 6 neighbors (7)
	hexes1 := svc.GetNeighborHexes(28.6139, 77.2090, 1)
	if len(hexes1) != 7 {
		t.Errorf("k=1 expected 7 hexes, got %d", len(hexes1))
	}

	// k=3: 1 + 3*k*(k+1) = 1 + 3*3*4 = 37 cells
	hexes3 := svc.GetNeighborHexes(28.6139, 77.2090, 3)
	if len(hexes3) != 37 {
		t.Errorf("k=3 expected 37 hexes, got %d", len(hexes3))
	}

	// Invalid resolution -> nil
	badSvc := NewGeofencingServiceWithResolution(nil, nil, -1)
	if hexesInv := badSvc.GetNeighborHexes(28.6139, 77.2090, 1); hexesInv != nil {
		t.Errorf("invalid resolution expected nil, got %v", hexesInv)
	}
}

func TestGeofencingService_DetectEntryAndExit(t *testing.T) {
	svc := NewGeofencingServiceWithResolution(nil, nil, 9)
	zone := &models.Zone{Latitude: 28.6139, Longitude: 77.2090}

	atZoneLoc := &models.Location{Latitude: 28.6139, Longitude: 77.2090}
	farLoc := &models.Location{Latitude: 28.6500, Longitude: 77.2500}

	// DetectEntry: was far, now at zone -> true
	if !svc.DetectEntry(farLoc, atZoneLoc, zone) {
		t.Error("expected DetectEntry(far, atZone) to be true")
	}

	// DetectEntry: already at zone -> false
	if svc.DetectEntry(atZoneLoc, atZoneLoc, zone) {
		t.Error("expected DetectEntry(atZone, atZone) to be false")
	}

	// DetectEntry: nil guards
	if svc.DetectEntry(nil, atZoneLoc, zone) || svc.DetectEntry(farLoc, nil, zone) || svc.DetectEntry(farLoc, atZoneLoc, nil) {
		t.Error("DetectEntry with nil arguments must return false")
	}

	// DetectExit: was at zone, now far -> true
	if !svc.DetectExit(atZoneLoc, farLoc, zone) {
		t.Error("expected DetectExit(atZone, far) to be true")
	}

	// DetectExit: was far, still far -> false
	if svc.DetectExit(farLoc, farLoc, zone) {
		t.Error("expected DetectExit(far, far) to be false")
	}

	// DetectExit: nil guards
	if svc.DetectExit(nil, farLoc, zone) || svc.DetectExit(atZoneLoc, nil, zone) || svc.DetectExit(atZoneLoc, farLoc, nil) {
		t.Error("DetectExit with nil arguments must return false")
	}
}

func TestGeofencingService_FindNearbyDevices_SmallRadiusKRing(t *testing.T) {
	ctx := context.Background()
	var queryHexes []string
	var queryMaxAge int

	mockLoc := &mockLocationRepo{
		getDevicesInHexesFn: func(ctx context.Context, hexes []string, maxAgeMinutes int) ([]models.NearbyDevice, error) {
			queryHexes = hexes
			queryMaxAge = maxAgeMinutes
			return []models.NearbyDevice{
				{DeviceID: "bus-1", Latitude: 28.6139, Longitude: 77.2090, Speed: 25.0},
			}, nil
		},
	}

	svc := NewGeofencingServiceWithResolution(mockLoc, nil, 9)

	// Radius 350m <= 500m -> uses H3 k-ring path
	devices, err := svc.FindNearbyDevices(ctx, 28.6139, 77.2090, 350)
	if err != nil {
		t.Fatalf("FindNearbyDevices failed: %v", err)
	}

	if len(devices) != 1 || devices[0].DeviceID != "bus-1" {
		t.Errorf("unexpected devices returned: %v", devices)
	}
	if len(queryHexes) == 0 {
		t.Error("expected non-empty hexes to be queried")
	}
	if queryMaxAge != 5 {
		t.Errorf("expected maxAge 5, got %d", queryMaxAge)
	}
}

func TestGeofencingService_FindNearbyDevices_LargeRadiusPostGIS(t *testing.T) {
	ctx := context.Background()
	var directCalled bool

	mockLoc := &mockLocationRepo{
		getDevicesNearZoneFn: func(ctx context.Context, lat, lng float64, radiusMeters int, maxAgeMinutes int) ([]models.NearbyDevice, error) {
			directCalled = true
			return []models.NearbyDevice{
				{DeviceID: "bus-2", Latitude: 28.6139, Longitude: 77.2090, Speed: 15.0},
			}, nil
		},
	}

	svc := NewGeofencingServiceWithResolution(mockLoc, nil, 9)

	// Radius 1000m > 500m -> delegates directly to PostGIS GetDevicesNearZone
	devices, err := svc.FindNearbyDevices(ctx, 28.6139, 77.2090, 1000)
	if err != nil {
		t.Fatalf("FindNearbyDevices failed: %v", err)
	}

	if !directCalled {
		t.Error("expected direct PostGIS GetDevicesNearZone to be called for radius > 500m")
	}
	if len(devices) != 1 || devices[0].DeviceID != "bus-2" {
		t.Errorf("unexpected devices: %v", devices)
	}
}

func TestGeofencingService_FindNearbyZones(t *testing.T) {
	ctx := context.Background()
	mockZone := &mockZoneRepo{
		getNearbyFn: func(ctx context.Context, lat, lng float64, radiusMeters int) ([]models.Zone, error) {
			return []models.Zone{
				{ID: "stop-1", Name: "CP Outer Circle", Latitude: lat, Longitude: lng},
			}, nil
		},
	}

	svc := NewGeofencingServiceWithResolution(nil, mockZone, 9)
	zones, err := svc.FindNearbyZones(ctx, 28.6139, 77.2090, 500)
	if err != nil {
		t.Fatalf("FindNearbyZones failed: %v", err)
	}
	if len(zones) != 1 || zones[0].ID != "stop-1" {
		t.Errorf("unexpected zones: %v", zones)
	}
}

func TestGeofencingService_FindNearbyDevices_RepoError(t *testing.T) {
	ctx := context.Background()
	mockLoc := &mockLocationRepo{
		getDevicesInHexesFn: func(ctx context.Context, hexes []string, maxAgeMinutes int) ([]models.NearbyDevice, error) {
			return nil, fmt.Errorf("db query error")
		},
	}

	svc := NewGeofencingServiceWithResolution(mockLoc, nil, 9)
	_, err := svc.FindNearbyDevices(ctx, 28.6139, 77.2090, 300)
	if err == nil {
		t.Error("expected error from repository failure")
	}
}
