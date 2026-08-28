package services

import (
	"context"
	"fmt"
	"log"

	"transit-backend/internal/hub"
	"transit-backend/internal/models"
)

// IngestService owns the write side of a location ping: persist it, resolve
// the device's active route, refresh the hot-state cache, and broadcast it
// to connected WebSocket clients. Bundling these side effects behind one
// service is also the seam Phase 4 hangs geofence evaluation off (CLAUDE.md
// Sec 6).
type IngestService struct {
	locRepo   LocationRepository
	routeRepo DeviceRouteRepository
	cache     DeviceCache
	hub       *hub.Hub
}

// NewIngestService creates a new IngestService.
func NewIngestService(locRepo LocationRepository, routeRepo DeviceRouteRepository, cache DeviceCache, h *hub.Hub) *IngestService {
	return &IngestService{locRepo: locRepo, routeRepo: routeRepo, cache: cache, hub: h}
}

// IngestLocation persists a location ping, resolves the device's active
// route, refreshes the cached hot state, and broadcasts the update. The
// caller is expected to have already validated loc (device_id present,
// coordinate bounds, speed bounds) -- that's HTTP-boundary validation and
// stays in the handler. route_id is never accepted from the caller: it is
// always resolved server-side from the device's active trip (CLAUDE.md
// Sec 7.2).
func (s *IngestService) IngestLocation(ctx context.Context, loc *models.Location) error {
	if err := s.locRepo.Insert(ctx, loc); err != nil {
		return fmt.Errorf("insert location: %w", err)
	}

	// Route resolution, cache, and broadcast are enrichment/side effects
	// that don't fail the request -- the durable write already succeeded.
	// Logged via the standard logger since this layer has no request-scoped
	// logger (structured, levelled logging per Sec 7.1 isn't wired yet --
	// see IDEAS.md D5).
	routeID, err := s.routeRepo.GetActiveRouteID(ctx, loc.DeviceID)
	if err != nil {
		log.Printf("failed to resolve active route for %s: %v", loc.DeviceID, err)
	}

	deviceState := map[string]interface{}{
		"latitude":  loc.Latitude,
		"longitude": loc.Longitude,
		"speed":     loc.Speed,
		"hex_res9":  loc.HexRes9,
		"last_seen": loc.Timestamp,
	}
	if err := s.cache.SetDeviceState(ctx, loc.DeviceID, deviceState); err != nil {
		log.Printf("failed to update device cache for %s: %v", loc.DeviceID, err)
	}

	msg := hub.Message{
		Type:      hub.MsgTypeLocationUpdate,
		DeviceID:  loc.DeviceID,
		RouteID:   routeID,
		Latitude:  loc.Latitude,
		Longitude: loc.Longitude,
		Speed:     loc.Speed,
		Accuracy:  loc.Accuracy,
		H3Hex:     loc.HexRes9,
		Timestamp: loc.Timestamp,
	}
	select {
	case s.hub.Broadcast <- msg:
	default:
		// Drop message if buffer full - backpressure handling
		log.Printf("WebSocket broadcast buffer full, message dropped for device %s", loc.DeviceID)
	}

	return nil
}
