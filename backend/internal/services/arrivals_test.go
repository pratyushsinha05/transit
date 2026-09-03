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
	getActiveTripsBeforeStopFn func(ctx context.Context, stopSequence int) ([]models.TripWithLocation, error)
}

func (m *mockTripRepo) GetActiveTripsBeforeStop(ctx context.Context, stopSequence int) ([]models.TripWithLocation, error) {
	if m.getActiveTripsBeforeStopFn != nil {
		return m.getActiveTripsBeforeStopFn(ctx, stopSequence)
	}
	return nil, nil
}

func TestArrivalsService_IsApproaching_RealCoordinates(t *testing.T) {
	geoSvc := NewGeofencingServiceWithResolution(nil, nil, 9)
	svc := &ArrivalsService{geoService: geoSvc}

	// Reference stop: Connaught Place, New Delhi
	stopLat, stopLng := 28.6139, 77.2090
	stopCell, err := h3.LatLngToCell(h3.NewLatLng(stopLat, stopLng), 9)
	if err != nil {
		t.Fatalf("failed to calculate stop cell: %v", err)
	}

	// 1. Center cell (bus is right at the stop) -> in k=3 ring -> approaching
	if !svc.isApproaching(stopLat, stopLng, stopLat, stopLng) {
		t.Errorf("bus at stop cell %s should be approaching", stopCell.String())
	}

	// 2. 1-ring neighbor (~175m) -> in k=3 ring -> approaching
	ring1, _ := h3.GridDisk(stopCell, 1)
	var cellK1 h3.Cell
	for _, c := range ring1 {
		if c != stopCell {
			cellK1 = c
			break
		}
	}
	llK1, _ := h3.CellToLatLng(cellK1)
	if !svc.isApproaching(llK1.Lat, llK1.Lng, stopLat, stopLng) {
		t.Errorf("bus in k=1 neighbor cell %s should be approaching", cellK1.String())
	}

	// 3. 3-ring cell (~500m) -> exactly on edge of k=3 ring -> approaching
	ring3, _ := h3.GridDisk(stopCell, 3)
	ring2, _ := h3.GridDisk(stopCell, 2)
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
	if !svc.isApproaching(llK3.Lat, llK3.Lng, stopLat, stopLng) {
		t.Errorf("bus in k=3 cell %s should be approaching", cellK3.String())
	}

	// 4. 4-ring cell (~700m) -> outside k=3 ring -> NOT approaching
	ring4, _ := h3.GridDisk(stopCell, 4)
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
	if svc.isApproaching(llK4.Lat, llK4.Lng, stopLat, stopLng) {
		t.Errorf("bus in k=4 cell %s should NOT be approaching (outside k=3)", cellK4.String())
	}

	// 5. Far away (~5 km) -> NOT approaching
	if svc.isApproaching(28.6600, 77.2600, stopLat, stopLng) {
		t.Error("bus 5km away should NOT be approaching")
	}

	// 6. Invalid resolution (empty busHex) -> false
	badGeoSvc := NewGeofencingServiceWithResolution(nil, nil, -1)
	badArrivalSvc := &ArrivalsService{geoService: badGeoSvc}
	if badArrivalSvc.isApproaching(stopLat, stopLng, stopLat, stopLng) {
		t.Error("empty hex should NOT be approaching")
	}
}

func TestArrivalsService_CalculateETAWithTraffic(t *testing.T) {
	geoSvc := NewGeofencingServiceWithResolution(nil, nil, 9)
	svc := &ArrivalsService{geoService: geoSvc}

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

	t.Run("DefaultSpeedFallback_StoppedBus", func(t *testing.T) {
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

func TestArrivalsService_GetArrivalsForStop(t *testing.T) {
	ctx := context.Background()
	stopLat, stopLng := 28.6139, 77.2090

	mockStop := &mockStopRepo{
		getByIDFn: func(ctx context.Context, stopID string) (*models.Stop, error) {
			if stopID != "stop-101" {
				return nil, fmt.Errorf("stop not found")
			}
			return &models.Stop{
				ID:        "stop-101",
				Name:      "Connaught Place",
				Latitude:  stopLat,
				Longitude: stopLng,
				Sequence:  5,
			}, nil
		},
	}

	mockTrip := &mockTripRepo{
		getActiveTripsBeforeStopFn: func(ctx context.Context, stopSequence int) ([]models.TripWithLocation, error) {
			if stopSequence != 5 {
				t.Errorf("expected stopSequence 5, got %d", stopSequence)
			}
			return []models.TripWithLocation{
				{
					TripID:     "trip-1",
					DeviceID:   "dev-1",
					DeviceName: "Bus 101",
					Latitude:   28.6139, // At stop
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
	svc := NewArrivalsService(mockStop, mockTrip, nil, geoSvc)

	predictions, err := svc.GetArrivalsForStop(ctx, "stop-101")
	if err != nil {
		t.Fatalf("GetArrivalsForStop failed: %v", err)
	}

	if len(predictions) != 2 {
		t.Fatalf("expected 2 predictions, got %d", len(predictions))
	}

	// Trip 1: At stop
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

func TestArrivalsService_GetArrivalsForStop_Errors(t *testing.T) {
	ctx := context.Background()

	t.Run("StopNotFound", func(t *testing.T) {
		mockStop := &mockStopRepo{
			getByIDFn: func(ctx context.Context, stopID string) (*models.Stop, error) {
				return nil, fmt.Errorf("stop not found")
			},
		}
		svc := NewArrivalsService(mockStop, nil, nil, nil)
		_, err := svc.GetArrivalsForStop(ctx, "missing")
		if err == nil {
			t.Error("expected error for missing stop")
		}
	})

	t.Run("TripRepoError", func(t *testing.T) {
		mockStop := &mockStopRepo{
			getByIDFn: func(ctx context.Context, stopID string) (*models.Stop, error) {
				return &models.Stop{ID: "stop-1", Sequence: 2}, nil
			},
		}
		mockTrip := &mockTripRepo{
			getActiveTripsBeforeStopFn: func(ctx context.Context, stopSequence int) ([]models.TripWithLocation, error) {
				return nil, fmt.Errorf("trip db error")
			},
		}
		svc := NewArrivalsService(mockStop, mockTrip, nil, nil)
		_, err := svc.GetArrivalsForStop(ctx, "stop-1")
		if err == nil {
			t.Error("expected error for trip repo failure")
		}
	})
}

func TestArrivalsService_GetNearbyArrivals(t *testing.T) {
	ctx := context.Background()

	mockStop := &mockStopRepo{
		getByIDFn: func(ctx context.Context, stopID string) (*models.Stop, error) {
			return &models.Stop{ID: stopID, Latitude: 28.6139, Longitude: 77.2090, Sequence: 1}, nil
		},
		getNearbyFn: func(ctx context.Context, lat, lng float64, radiusMeters int) ([]models.Stop, error) {
			return []models.Stop{
				{ID: "stop-1", Latitude: 28.6139, Longitude: 77.2090, Sequence: 1},
				{ID: "stop-2", Latitude: 28.6140, Longitude: 77.2091, Sequence: 2},
			}, nil
		},
	}

	mockTrip := &mockTripRepo{
		getActiveTripsBeforeStopFn: func(ctx context.Context, stopSequence int) ([]models.TripWithLocation, error) {
			return []models.TripWithLocation{
				{TripID: "trip-unique", DeviceID: "dev-1", Latitude: 28.6139, Longitude: 77.2090, Speed: 20.0},
			}, nil
		},
	}

	geoSvc := NewGeofencingServiceWithResolution(nil, mockStop, 9)
	svc := NewArrivalsService(mockStop, mockTrip, nil, geoSvc)

	predictions, err := svc.GetNearbyArrivals(ctx, 28.6139, 77.2090, 500)
	if err != nil {
		t.Fatalf("GetNearbyArrivals failed: %v", err)
	}

	// Should deduplicate "trip-unique" across stops
	if len(predictions) != 1 || predictions[0].TripID != "trip-unique" {
		t.Errorf("expected 1 deduplicated prediction for trip-unique, got %v", predictions)
	}
}

func TestArrivalsService_DetectArrivalEvent(t *testing.T) {
	ctx := context.Background()
	stop := &models.Stop{Latitude: 28.6139, Longitude: 77.2090}

	t.Run("AtStop", func(t *testing.T) {
		mockLoc := &mockLocationRepo{
			getLatestFn: func(ctx context.Context, deviceID string) (*models.Location, error) {
				return &models.Location{Latitude: 28.6139, Longitude: 77.2090}, nil
			},
		}
		geoSvc := NewGeofencingServiceWithResolution(mockLoc, nil, 9)
		svc := NewArrivalsService(nil, nil, mockLoc, geoSvc)

		atStop, err := svc.DetectArrivalEvent(ctx, "dev-1", stop)
		if err != nil || !atStop {
			t.Errorf("expected true, got %v (err: %v)", atStop, err)
		}
	})

	t.Run("LocationRepoError", func(t *testing.T) {
		mockLoc := &mockLocationRepo{
			getLatestFn: func(ctx context.Context, deviceID string) (*models.Location, error) {
				return nil, fmt.Errorf("location error")
			},
		}
		geoSvc := NewGeofencingServiceWithResolution(mockLoc, nil, 9)
		svc := NewArrivalsService(nil, nil, mockLoc, geoSvc)

		_, err := svc.DetectArrivalEvent(ctx, "dev-1", stop)
		if err == nil {
			t.Error("expected error from location repo failure")
		}
	})
}
