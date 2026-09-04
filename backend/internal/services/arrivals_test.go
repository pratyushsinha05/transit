package services

import (
	"context"
	"fmt"
	"testing"
	"transit-backend/internal/models"
	"transit-backend/pkg/geo"

	"github.com/uber/h3-go/v4"
)

type mockTripRepo struct {
	getActiveTripsBeforeZoneFn func(ctx context.Context, stopSequence int) ([]models.TripWithLocation, error)
}

func (m *mockTripRepo) GetActiveTripsBeforeZone(ctx context.Context, stopSequence int) ([]models.TripWithLocation, error) {
	if m.getActiveTripsBeforeZoneFn != nil {
		return m.getActiveTripsBeforeZoneFn(ctx, stopSequence)
	}
	return nil, nil
}

func TestGeofenceService_IsApproaching_RealCoordinates(t *testing.T) {
	geoSvc := NewGeofencingServiceWithResolution(nil, nil, 9)
	svc := &GeofenceService{geoService: geoSvc}

	// Reference zone: Connaught Place, New Delhi
	zoneLat, zoneLng := 28.6139, 77.2090
	zoneCell, err := h3.LatLngToCell(h3.NewLatLng(zoneLat, zoneLng), 9)
	if err != nil {
		t.Fatalf("failed to calculate zone cell: %v", err)
	}

	// 1. Center cell (device is right at the zone) -> in k=3 ring -> approaching
	if !svc.isApproaching(zoneLat, zoneLng, zoneLat, zoneLng) {
		t.Errorf("device at zone cell %s should be approaching", zoneCell.String())
	}

	// 2. 1-ring neighbor (~175m) -> in k=3 ring -> approaching
	ring1, _ := h3.GridDisk(zoneCell, 1)
	var cellK1 h3.Cell
	for _, c := range ring1 {
		if c != zoneCell {
			cellK1 = c
			break
		}
	}
	llK1, _ := h3.CellToLatLng(cellK1)
	if !svc.isApproaching(llK1.Lat, llK1.Lng, zoneLat, zoneLng) {
		t.Errorf("device in k=1 cell %s should be approaching", cellK1.String())
	}

	// 3. 3-ring neighbor (~500m) -> in k=3 ring -> approaching
	ring3, _ := h3.GridDisk(zoneCell, 3)
	ring2, _ := h3.GridDisk(zoneCell, 2)
	ring2Map := make(map[h3.Cell]bool)
	for _, c := range ring2 {
		ring2Map[c] = true
	}
	var cellK3 h3.Cell
	for _, c := range ring3 {
		if !ring2Map[c] {
			cellK3 = c
			break
		}
	}
	llK3, _ := h3.CellToLatLng(cellK3)
	if !svc.isApproaching(llK3.Lat, llK3.Lng, zoneLat, zoneLng) {
		t.Errorf("device in k=3 cell %s should be approaching", cellK3.String())
	}

	// 4. 4-ring neighbor (~700m) -> OUTSIDE k=3 ring -> NOT approaching
	ring4, _ := h3.GridDisk(zoneCell, 4)
	ring3Map := make(map[h3.Cell]bool)
	for _, c := range ring3 {
		ring3Map[c] = true
	}
	var cellK4 h3.Cell
	for _, c := range ring4 {
		if !ring3Map[c] {
			cellK4 = c
			break
		}
	}
	llK4, _ := h3.CellToLatLng(cellK4)
	if svc.isApproaching(llK4.Lat, llK4.Lng, zoneLat, zoneLng) {
		t.Errorf("device in k=4 cell %s should NOT be approaching (outside k=3)", cellK4.String())
	}

	// 5. Far away (~5 km) -> NOT approaching
	if svc.isApproaching(28.6600, 77.2600, zoneLat, zoneLng) {
		t.Error("device 5km away should NOT be approaching")
	}

	// 6. Invalid resolution (empty deviceHex) -> false
	badGeoSvc := NewGeofencingServiceWithResolution(nil, nil, -1)
	badGeofenceSvc := &GeofenceService{geoService: badGeoSvc}
	if badGeofenceSvc.isApproaching(zoneLat, zoneLng, zoneLat, zoneLng) {
		t.Error("empty hex should NOT be approaching")
	}
}

func TestGeofenceService_CalculateETAWithTraffic(t *testing.T) {
	geoSvc := NewGeofencingServiceWithResolution(nil, nil, 9)
	svc := &GeofenceService{geoService: geoSvc}

	t.Run("NormalSpeed", func(t *testing.T) {
		fromLat, fromLng := 28.5000, 77.0000
		toLat, toLng := 28.6000, 77.1000
		speedKmh := 60.0

		// Hand calculation:
		// dist = Haversine(28.5, 77.0, 28.6, 77.1) = ~14.77 km
		// hours = 14.77 / 60.0 = 0.2461
		// baseETA = round(0.2461 * 60) = 15 minutes
		// trafficAdjusted = int(15 * 1.2) = 18 minutes
		baseETA := geo.CalculateETA(fromLat, fromLng, toLat, toLng, speedKmh)
		wantETA := int(float64(baseETA) * 1.2)

		gotETA := svc.CalculateETAWithTraffic(fromLat, fromLng, toLat, toLng, speedKmh)
		if gotETA != wantETA {
			t.Errorf("CalculateETAWithTraffic = %d, want %d (baseETA = %d)", gotETA, wantETA, baseETA)
		}
		if gotETA != 18 {
			t.Errorf("hand-computed ETA expected 18, got %d", gotETA)
		}
	})

	t.Run("DefaultSpeedFallback_StoppedDevice", func(t *testing.T) {
		// Speed is 0.0 km/h (< 1.0 km/h MinSpeedKmh) -> triggers DefaultSpeed (20.0 km/h)
		fromLat, fromLng := 28.6139, 77.2090
		toLat, toLng := 28.7139, 77.2090
		speedKmh := 0.0

		// Hand calculation:
		// dist = Haversine(28.6139, 77.2090, 28.7139, 77.2090) = 11.1195 km
		// speed < 1.0 -> speed = 20.0 km/h
		// hours = 11.1195 / 20.0 = 0.5560
		// baseETA = round(0.5560 * 60) = 33 minutes
		// trafficAdjusted = int(33 * 1.2) = 39 minutes
		baseETA := geo.CalculateETA(fromLat, fromLng, toLat, toLng, speedKmh)
		if baseETA != 33 {
			t.Errorf("expected baseETA with DefaultSpeed fallback to be 33, got %d", baseETA)
		}

		gotETA := svc.CalculateETAWithTraffic(fromLat, fromLng, toLat, toLng, speedKmh)
		wantETA := int(float64(baseETA) * 1.2)
		if gotETA != wantETA || gotETA != 39 {
			t.Errorf("stopped bus ETA = %d, want %d (hand-computed: 39)", gotETA, wantETA)
		}
	})
}

func TestGeofenceService_GetPredictionsForZone(t *testing.T) {
	ctx := context.Background()
	zoneLat, zoneLng := 28.6139, 77.2090

	mockZone := &mockZoneRepo{
		getByIDFn: func(ctx context.Context, stopID string) (*models.Zone, error) {
			if stopID != "stop-101" {
				return nil, fmt.Errorf("stop not found")
			}
			return &models.Zone{
				ID:        "stop-101",
				Name:      "Connaught Place",
				Latitude:  zoneLat,
				Longitude: zoneLng,
				Sequence:  5,
			}, nil
		},
	}

	mockTrip := &mockTripRepo{
		getActiveTripsBeforeZoneFn: func(ctx context.Context, stopSequence int) ([]models.TripWithLocation, error) {
			if stopSequence != 5 {
				t.Errorf("expected stopSequence 5, got %d", stopSequence)
			}
			return []models.TripWithLocation{
				{
					TripID:     "trip-1",
					DeviceID:   "dev-1",
					DeviceName: "Bus 101",
					Latitude:   28.6139, // At zone
					Longitude:  77.2090,
					Speed:      30.0,
				},
				{
					TripID:     "trip-2",
					DeviceID:   "dev-2",
					DeviceName: "Bus 102",
					Latitude:   28.7139, // ~11.12 km away, STOPPED (0.0 km/h) -> DefaultSpeed fallback!
					Longitude:  77.2090,
					Speed:      0.0,
				},
			}, nil
		},
	}

	geoSvc := NewGeofencingServiceWithResolution(nil, nil, 9)
	svc := NewGeofenceService(mockZone, mockTrip, nil, geoSvc)

	predictions, err := svc.GetPredictionsForZone(ctx, "stop-101")
	if err != nil {
		t.Fatalf("GetPredictionsForZone failed: %v", err)
	}

	if len(predictions) != 2 {
		t.Fatalf("expected 2 predictions, got %d", len(predictions))
	}

	// Trip 1: At zone
	p1 := predictions[0]
	if p1.TripID != "trip-1" || !p1.IsApproaching || p1.ETAMinutes != 0 {
		t.Errorf("unexpected p1: %+v", p1)
	}

	// Trip 2: Stopped bus (0 km/h) -> DefaultSpeed 20.0 km/h
	// Hand-computed ETA: round((11.1195 / 20.0) * 60) = 33 minutes
	p2 := predictions[1]
	if p2.TripID != "trip-2" || p2.CurrentSpeed != 0.0 {
		t.Errorf("unexpected p2: %+v", p2)
	}
	if p2.ETAMinutes != 33 {
		t.Errorf("p2.ETAMinutes with DefaultSpeed = %d, want 33", p2.ETAMinutes)
	}
	if p2.IsApproaching {
		t.Errorf("p2 (~11km away) should NOT be approaching (outside k=3)")
	}
}

func TestGeofenceService_GetPredictionsForZone_Errors(t *testing.T) {
	ctx := context.Background()

	t.Run("ZoneNotFound", func(t *testing.T) {
		mockZone := &mockZoneRepo{
			getByIDFn: func(ctx context.Context, stopID string) (*models.Zone, error) {
				return nil, fmt.Errorf("stop not found")
			},
		}
		svc := NewGeofenceService(mockZone, nil, nil, nil)
		_, err := svc.GetPredictionsForZone(ctx, "missing")
		if err == nil {
			t.Error("expected error for missing zone")
		}
	})

	t.Run("TripRepoError", func(t *testing.T) {
		mockZone := &mockZoneRepo{
			getByIDFn: func(ctx context.Context, stopID string) (*models.Zone, error) {
				return &models.Zone{ID: "stop-1", Sequence: 2}, nil
			},
		}
		mockTrip := &mockTripRepo{
			getActiveTripsBeforeZoneFn: func(ctx context.Context, stopSequence int) ([]models.TripWithLocation, error) {
				return nil, fmt.Errorf("trip db error")
			},
		}
		svc := NewGeofenceService(mockZone, mockTrip, nil, nil)
		_, err := svc.GetPredictionsForZone(ctx, "stop-1")
		if err == nil {
			t.Error("expected error for trip repo failure")
		}
	})
}

func TestGeofenceService_GetNearbyPredictions(t *testing.T) {
	ctx := context.Background()

	mockZone := &mockZoneRepo{
		getByIDFn: func(ctx context.Context, stopID string) (*models.Zone, error) {
			return &models.Zone{ID: stopID, Latitude: 28.6139, Longitude: 77.2090, Sequence: 1}, nil
		},
		getNearbyFn: func(ctx context.Context, lat, lng float64, radiusMeters int) ([]models.Zone, error) {
			return []models.Zone{
				{ID: "stop-1", Latitude: 28.6139, Longitude: 77.2090, Sequence: 1},
				{ID: "stop-2", Latitude: 28.6140, Longitude: 77.2091, Sequence: 2},
			}, nil
		},
	}

	mockTrip := &mockTripRepo{
		getActiveTripsBeforeZoneFn: func(ctx context.Context, stopSequence int) ([]models.TripWithLocation, error) {
			return []models.TripWithLocation{
				{TripID: "trip-unique", DeviceID: "dev-1", Latitude: 28.6139, Longitude: 77.2090, Speed: 20.0},
			}, nil
		},
	}

	geoSvc := NewGeofencingServiceWithResolution(nil, mockZone, 9)
	svc := NewGeofenceService(mockZone, mockTrip, nil, geoSvc)

	predictions, err := svc.GetNearbyPredictions(ctx, 28.6139, 77.2090, 500)
	if err != nil {
		t.Fatalf("GetNearbyPredictions failed: %v", err)
	}

	// Should deduplicate "trip-unique" across zones
	if len(predictions) != 1 || predictions[0].TripID != "trip-unique" {
		t.Errorf("expected 1 deduplicated prediction for trip-unique, got %v", predictions)
	}
}

func TestGeofenceService_DetectEntryEvent(t *testing.T) {
	ctx := context.Background()
	zone := &models.Zone{Latitude: 28.6139, Longitude: 77.2090}

	t.Run("AtZone", func(t *testing.T) {
		mockLoc := &mockLocationRepo{
			getLatestFn: func(ctx context.Context, deviceID string) (*models.Location, error) {
				return &models.Location{Latitude: 28.6139, Longitude: 77.2090}, nil
			},
		}
		geoSvc := NewGeofencingServiceWithResolution(mockLoc, nil, 9)
		svc := NewGeofenceService(nil, nil, mockLoc, geoSvc)

		atZone, err := svc.DetectEntryEvent(ctx, "dev-1", zone)
		if err != nil || !atZone {
			t.Errorf("expected true, got %v (err: %v)", atZone, err)
		}
	})

	t.Run("LocationRepoError", func(t *testing.T) {
		mockLoc := &mockLocationRepo{
			getLatestFn: func(ctx context.Context, deviceID string) (*models.Location, error) {
				return nil, fmt.Errorf("location error")
			},
		}
		geoSvc := NewGeofencingServiceWithResolution(mockLoc, nil, 9)
		svc := NewGeofenceService(nil, nil, mockLoc, geoSvc)

		_, err := svc.DetectEntryEvent(ctx, "dev-1", zone)
		if err == nil {
			t.Error("expected error from location repo failure")
		}
	})
}
