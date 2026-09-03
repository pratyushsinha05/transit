package services

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"transit-backend/internal/hub"
	"transit-backend/internal/models"
)

type mockDeviceRouteRepo struct {
	getActiveRouteIDFn func(ctx context.Context, deviceID string) (string, error)
}

func (m *mockDeviceRouteRepo) GetActiveRouteID(ctx context.Context, deviceID string) (string, error) {
	if m.getActiveRouteIDFn != nil {
		return m.getActiveRouteIDFn(ctx, deviceID)
	}
	return "", nil
}

type mockDeviceCache struct {
	setDeviceStateFn func(ctx context.Context, deviceID string, state map[string]interface{}) error
}

func (m *mockDeviceCache) SetDeviceState(ctx context.Context, deviceID string, state map[string]interface{}) error {
	if m.setDeviceStateFn != nil {
		return m.setDeviceStateFn(ctx, deviceID, state)
	}
	return nil
}

func TestIngestService_HappyPath(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var mu sync.Mutex
	var insertedLoc *models.Location
	var cachedState map[string]interface{}

	mockLoc := &mockLocationRepo{
		insertFn: func(ctx context.Context, loc *models.Location) error {
			mu.Lock()
			insertedLoc = loc
			mu.Unlock()
			return nil
		},
	}

	mockRoute := &mockDeviceRouteRepo{
		getActiveRouteIDFn: func(ctx context.Context, deviceID string) (string, error) {
			if deviceID == "dev-1" {
				return "route-101", nil
			}
			return "", nil
		},
	}

	mockCache := &mockDeviceCache{
		setDeviceStateFn: func(ctx context.Context, deviceID string, state map[string]interface{}) error {
			mu.Lock()
			cachedState = state
			mu.Unlock()
			return nil
		},
	}

	h := hub.New()
	hubCtx, hubCancel := context.WithCancel(context.Background())
	defer func() {
		hubCancel()
		h.Shutdown()
	}()
	go h.Run(hubCtx)

	// Register a client to receive the broadcast
	client := &hub.Client{Hub: h, Send: make(chan interface{}, 16)}
	h.Register <- client

	// Send a handshake message to deterministically verify registration
	h.Broadcast <- "ready"
	select {
	case msg := <-client.Send:
		if msg != "ready" {
			t.Fatalf("unexpected ready message: %v", msg)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for client registration")
	}

	svc := NewIngestService(mockLoc, mockRoute, mockCache, h)

	loc := &models.Location{
		DeviceID:  "dev-1",
		Latitude:  28.6139,
		Longitude: 77.2090,
		Speed:     35.5,
		Accuracy:  5.0,
		Timestamp: 1672531199,
		HexRes9:   "891f5a44a4bffff",
	}

	err := svc.IngestLocation(ctx, loc)
	if err != nil {
		t.Fatalf("IngestLocation failed: %v", err)
	}

	// Verify persistence call
	mu.Lock()
	if insertedLoc == nil || insertedLoc.DeviceID != "dev-1" {
		t.Errorf("expected location to be inserted, got %v", insertedLoc)
	}

	// Verify cache call
	if cachedState == nil {
		t.Error("expected cache to be updated")
	} else {
		if cachedState["latitude"] != 28.6139 || cachedState["speed"] != 35.5 {
			t.Errorf("unexpected cached state: %v", cachedState)
		}
	}
	mu.Unlock()

	// Verify broadcast on hub
	select {
	case rawMsg := <-client.Send:
		msg, ok := rawMsg.(hub.Message)
		if !ok {
			t.Fatalf("received message not hub.Message, got %T", rawMsg)
		}
		if msg.DeviceID != "dev-1" || msg.RouteID != "route-101" || msg.Speed != 35.5 {
			t.Errorf("unexpected broadcast message: %+v", msg)
		}
		if msg.Type != hub.MsgTypeLocationUpdate {
			t.Errorf("message Type = %q, want %q", msg.Type, hub.MsgTypeLocationUpdate)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for broadcast message on client")
	}
}

func TestIngestService_InsertError(t *testing.T) {
	ctx := context.Background()

	mockLoc := &mockLocationRepo{
		insertFn: func(ctx context.Context, loc *models.Location) error {
			return fmt.Errorf("connection refused")
		},
	}

	var routeCalled, cacheCalled bool
	mockRoute := &mockDeviceRouteRepo{
		getActiveRouteIDFn: func(ctx context.Context, deviceID string) (string, error) {
			routeCalled = true
			return "route-1", nil
		},
	}
	mockCache := &mockDeviceCache{
		setDeviceStateFn: func(ctx context.Context, deviceID string, state map[string]interface{}) error {
			cacheCalled = true
			return nil
		},
	}

	h := hub.New()
	svc := NewIngestService(mockLoc, mockRoute, mockCache, h)

	loc := &models.Location{DeviceID: "dev-err"}
	err := svc.IngestLocation(ctx, loc)
	if err == nil {
		t.Fatal("expected error from IngestLocation when Insert fails")
	}

	if routeCalled || cacheCalled {
		t.Error("route and cache should not be called if Insert fails")
	}
}

func TestIngestService_RouteResolveError_NonFatal(t *testing.T) {
	ctx := context.Background()

	mockLoc := &mockLocationRepo{
		insertFn: func(ctx context.Context, loc *models.Location) error { return nil },
	}
	mockRoute := &mockDeviceRouteRepo{
		getActiveRouteIDFn: func(ctx context.Context, deviceID string) (string, error) {
			return "", fmt.Errorf("trip query timeout")
		},
	}
	mockCache := &mockDeviceCache{
		setDeviceStateFn: func(ctx context.Context, deviceID string, state map[string]interface{}) error { return nil },
	}

	h := hub.New()
	svc := NewIngestService(mockLoc, mockRoute, mockCache, h)

	loc := &models.Location{DeviceID: "dev-no-route"}
	err := svc.IngestLocation(ctx, loc)
	if err != nil {
		t.Errorf("route resolution error should be non-fatal, got %v", err)
	}
}

func TestIngestService_CacheError_NonFatal(t *testing.T) {
	ctx := context.Background()

	mockLoc := &mockLocationRepo{
		insertFn: func(ctx context.Context, loc *models.Location) error { return nil },
	}
	mockRoute := &mockDeviceRouteRepo{
		getActiveRouteIDFn: func(ctx context.Context, deviceID string) (string, error) { return "route-1", nil },
	}
	mockCache := &mockDeviceCache{
		setDeviceStateFn: func(ctx context.Context, deviceID string, state map[string]interface{}) error {
			return fmt.Errorf("redis OOM")
		},
	}

	h := hub.New()
	svc := NewIngestService(mockLoc, mockRoute, mockCache, h)

	loc := &models.Location{DeviceID: "dev-cache-fail"}
	err := svc.IngestLocation(ctx, loc)
	if err != nil {
		t.Errorf("cache error should be non-fatal, got %v", err)
	}
}

func TestIngestService_BroadcastBackpressureDrop(t *testing.T) {
	ctx := context.Background()

	mockLoc := &mockLocationRepo{
		insertFn: func(ctx context.Context, loc *models.Location) error { return nil },
	}
	mockRoute := &mockDeviceRouteRepo{
		getActiveRouteIDFn: func(ctx context.Context, deviceID string) (string, error) { return "r-1", nil },
	}
	mockCache := &mockDeviceCache{
		setDeviceStateFn: func(ctx context.Context, deviceID string, state map[string]interface{}) error { return nil },
	}

	h := hub.New()
	// Fill Broadcast buffer completely (cap is 256)
	for i := 0; i < 256; i++ {
		h.Broadcast <- "filler"
	}

	// Now Broadcast channel is full, so next IngestLocation hits default: branch
	svc := NewIngestService(mockLoc, mockRoute, mockCache, h)
	loc := &models.Location{DeviceID: "dev-drop"}
	err := svc.IngestLocation(ctx, loc)
	if err != nil {
		t.Errorf("buffer full should drop message and return nil, got %v", err)
	}
}
