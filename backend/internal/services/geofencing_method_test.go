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
	insertFn           func(ctx context.Context, loc *models.Location) error
	getLatestFn        func(ctx context.Context, deviceID string) (*models.Location, error)
	getBusesInHexesFn  func(ctx context.Context, hexes []string, maxAgeMinutes int) ([]models.NearbyBus, error)
	getBusesNearStopFn func(ctx context.Context, lat, lng float64, radiusMeters int, maxAgeMinutes int) ([]models.NearbyBus, error)
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

func (m *mockLocationRepo) GetBusesInHexes(ctx context.Context, hexes []string, maxAgeMinutes int) ([]models.NearbyBus, error) {
	if m.getBusesInHexesFn != nil {
		return m.getBusesInHexesFn(ctx, hexes, maxAgeMinutes)
	}
	return nil, nil
}

func (m *mockLocationRepo) GetBusesNearStop(ctx context.Context, lat, lng float64, radiusMeters int, maxAgeMinutes int) ([]models.NearbyBus, error) {
	if m.getBusesNearStopFn != nil {
		return m.getBusesNearStopFn(ctx, lat, lng, radiusMeters, maxAgeMinutes)
	}
	return nil, nil
}

// mockStopRepo is a test double for StopRepository.
type mockStopRepo struct {
	getByIDFn      func(ctx context.Context, stopID string) (*models.Stop, error)
	getByRouteIDFn func(ctx context.Context, routeID string) ([]models.Stop, error)
	getNearbyFn    func(ctx context.Context, lat, lng float64, radiusMeters int) ([]models.Stop, error)
}

func (m *mockStopRepo) GetByID(ctx context.Context, stopID string) (*models.Stop, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, stopID)
	}
	return nil, nil
}

func (m *mockStopRepo) GetByRouteID(ctx context.Context, routeID string) ([]models.Stop, error) {
	if m.getByRouteIDFn != nil {
		return m.getByRouteIDFn(ctx, routeID)
	}
	return nil, nil
}

func (m *mockStopRepo) GetNearby(ctx context.Context, lat, lng float64, radiusMeters int) ([]models.Stop, error) {
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

func TestGeofencingService_IsAtStop(t *testing.T) {
	svc := NewGeofencingServiceWithResolution(nil, nil, 9)

	// Connaught Place coordinates
	stopLat, stopLng := 28.6139, 77.2090

	// Same coordinates -> same hex -> at stop
	if !svc.IsAtStop(stopLat, stopLng, stopLat, stopLng) {
		t.Error("expected same coordinates to be at stop")
	}

	// 15 meters away -> still within ~175m edge hex
	if !svc.IsAtStop(28.61395, 77.20905, stopLat, stopLng) {
		t.Error("expected 15m away point to be in same hex at resolution 9")
	}

	// ~1.5 km away -> different hex
	if svc.IsAtStop(28.6270, 77.2190, stopLat, stopLng) {
		t.Error("expected point 1.5km away to NOT be at stop")
	}

	// Service with invalid resolution returns empty hexes, so IsAtStop returns false
	badSvc := NewGeofencingServiceWithResolution(nil, nil, -1)
	if badSvc.IsAtStop(stopLat, stopLng, stopLat, stopLng) {
		t.Error("expected invalid resolution to return false")
	}
}

func TestGeofencingService_IsAtStopWithHysteresis(t *testing.T) {
	svc := NewGeofencingServiceWithResolution(nil, nil, 9)
	stopLat, stopLng := 28.6139, 77.2090
	stopCell, _ := h3.LatLngToCell(h3.NewLatLng(stopLat, stopLng), 9)

	// Find an exact 1-ring neighbor cell
	neighbors, err := h3.GridDisk(stopCell, 1)
	if err != nil || len(neighbors) < 2 {
		t.Fatalf("failed to get 1-ring neighbors: %v", err)
	}

	// Pick a neighbor cell that is not the center cell itself
	var neighborCell h3.Cell
	for _, n := range neighbors {
		if n != stopCell {
			neighborCell = n
			break
		}
	}
	neighborLL, _ := h3.CellToLatLng(neighborCell)
	neighborLat, neighborLng := neighborLL.Lat, neighborLL.Lng

	// Case 1: Same hex, wasAtStop = false -> true
	if !svc.IsAtStopWithHysteresis(stopLat, stopLng, stopLat, stopLng, false) {
		t.Error("same hex must return true regardless of wasAtStop")
	}

	// Case 2: In neighboring hex, wasAtStop = true -> true (hysteresis buffer holds)
	if !svc.IsAtStopWithHysteresis(neighborLat, neighborLng, stopLat, stopLng, true) {
		t.Errorf("neighbor hex (%s) with wasAtStop=true must return true under hysteresis", neighborCell.String())
	}

	// Case 3: In neighboring hex, wasAtStop = false -> false (no hysteresis buffer for new arrivals)
	if svc.IsAtStopWithHysteresis(neighborLat, neighborLng, stopLat, stopLng, false) {
		t.Errorf("neighbor hex (%s) with wasAtStop=false must return false", neighborCell.String())
	}

	// Case 4: Far away (~5km), wasAtStop = true -> false (outside 1-ring buffer)
	farLat, farLng := 28.6600, 77.2600
	if svc.IsAtStopWithHysteresis(farLat, farLng, stopLat, stopLng, true) {
		t.Error("far away point must return false even with wasAtStop=true")
	}

	// Case 5: Invalid resolution -> false
	badSvc := NewGeofencingServiceWithResolution(nil, nil, -1)
	if badSvc.IsAtStopWithHysteresis(stopLat, stopLng, stopLat, stopLng, true) {
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

func TestGeofencingService_DetectArrivalAndDeparture(t *testing.T) {
	svc := NewGeofencingServiceWithResolution(nil, nil, 9)
	stop := &models.Stop{Latitude: 28.6139, Longitude: 77.2090}

	atStopLoc := &models.Location{Latitude: 28.6139, Longitude: 77.2090}
	farLoc := &models.Location{Latitude: 28.6500, Longitude: 77.2500}

	// DetectArrival: was far, now at stop -> true
	if !svc.DetectArrival(farLoc, atStopLoc, stop) {
		t.Error("expected DetectArrival(far, atStop) to be true")
	}

	// DetectArrival: already at stop -> false
	if svc.DetectArrival(atStopLoc, atStopLoc, stop) {
		t.Error("expected DetectArrival(atStop, atStop) to be false")
	}

	// DetectArrival: nil guards
	if svc.DetectArrival(nil, atStopLoc, stop) || svc.DetectArrival(farLoc, nil, stop) || svc.DetectArrival(farLoc, atStopLoc, nil) {
		t.Error("DetectArrival with nil arguments must return false")
	}

	// DetectDeparture: was at stop, now far -> true
	if !svc.DetectDeparture(atStopLoc, farLoc, stop) {
		t.Error("expected DetectDeparture(atStop, far) to be true")
	}

	// DetectDeparture: was far, still far -> false
	if svc.DetectDeparture(farLoc, farLoc, stop) {
		t.Error("expected DetectDeparture(far, far) to be false")
	}

	// DetectDeparture: nil guards
	if svc.DetectDeparture(nil, farLoc, stop) || svc.DetectDeparture(atStopLoc, nil, stop) || svc.DetectDeparture(atStopLoc, farLoc, nil) {
		t.Error("DetectDeparture with nil arguments must return false")
	}
}

func TestGeofencingService_FindNearbyBuses_SmallRadiusKRing(t *testing.T) {
	ctx := context.Background()
	var queryHexes []string
	var queryMaxAge int

	mockLoc := &mockLocationRepo{
		getBusesInHexesFn: func(ctx context.Context, hexes []string, maxAgeMinutes int) ([]models.NearbyBus, error) {
			queryHexes = hexes
			queryMaxAge = maxAgeMinutes
			return []models.NearbyBus{
				{DeviceID: "bus-1", Latitude: 28.6139, Longitude: 77.2090, Speed: 25.0},
			}, nil
		},
	}

	svc := NewGeofencingServiceWithResolution(mockLoc, nil, 9)

	// Radius 350m <= 500m -> uses H3 k-ring path
	buses, err := svc.FindNearbyBuses(ctx, 28.6139, 77.2090, 350)
	if err != nil {
		t.Fatalf("FindNearbyBuses failed: %v", err)
	}

	if len(buses) != 1 || buses[0].DeviceID != "bus-1" {
		t.Errorf("unexpected buses returned: %v", buses)
	}
	if len(queryHexes) == 0 {
		t.Error("expected non-empty hexes to be queried")
	}
	if queryMaxAge != 5 {
		t.Errorf("expected maxAge 5, got %d", queryMaxAge)
	}
}

func TestGeofencingService_FindNearbyBuses_LargeRadiusPostGIS(t *testing.T) {
	ctx := context.Background()
	var directCalled bool

	mockLoc := &mockLocationRepo{
		getBusesNearStopFn: func(ctx context.Context, lat, lng float64, radiusMeters int, maxAgeMinutes int) ([]models.NearbyBus, error) {
			directCalled = true
			return []models.NearbyBus{
				{DeviceID: "bus-2", Latitude: 28.6139, Longitude: 77.2090, Speed: 15.0},
			}, nil
		},
	}

	svc := NewGeofencingServiceWithResolution(mockLoc, nil, 9)

	// Radius 1000m > 500m -> delegates directly to PostGIS GetBusesNearStop
	buses, err := svc.FindNearbyBuses(ctx, 28.6139, 77.2090, 1000)
	if err != nil {
		t.Fatalf("FindNearbyBuses failed: %v", err)
	}

	if !directCalled {
		t.Error("expected direct PostGIS GetBusesNearStop to be called for radius > 500m")
	}
	if len(buses) != 1 || buses[0].DeviceID != "bus-2" {
		t.Errorf("unexpected buses: %v", buses)
	}
}

func TestGeofencingService_FindNearbyStops(t *testing.T) {
	ctx := context.Background()
	mockStop := &mockStopRepo{
		getNearbyFn: func(ctx context.Context, lat, lng float64, radiusMeters int) ([]models.Stop, error) {
			return []models.Stop{
				{ID: "stop-1", Name: "CP Outer Circle", Latitude: lat, Longitude: lng},
			}, nil
		},
	}

	svc := NewGeofencingServiceWithResolution(nil, mockStop, 9)
	stops, err := svc.FindNearbyStops(ctx, 28.6139, 77.2090, 500)
	if err != nil {
		t.Fatalf("FindNearbyStops failed: %v", err)
	}
	if len(stops) != 1 || stops[0].ID != "stop-1" {
		t.Errorf("unexpected stops: %v", stops)
	}
}

func TestGeofencingService_FindNearbyBuses_RepoError(t *testing.T) {
	ctx := context.Background()
	mockLoc := &mockLocationRepo{
		getBusesInHexesFn: func(ctx context.Context, hexes []string, maxAgeMinutes int) ([]models.NearbyBus, error) {
			return nil, fmt.Errorf("db query error")
		},
	}

	svc := NewGeofencingServiceWithResolution(mockLoc, nil, 9)
	_, err := svc.FindNearbyBuses(ctx, 28.6139, 77.2090, 300)
	if err == nil {
		t.Error("expected error from repository failure")
	}
}
