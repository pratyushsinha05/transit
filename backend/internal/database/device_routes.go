package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DeviceRouteRepository resolves a device's currently active route_id.
//
// Deliberately a separate type/file from TripRepository (trips.go): CLAUDE.md
// Sec 5.5 scopes trips.go and models/trip.go out of Phases 0-4.5 pending
// separate review. This is a read-only, additive query against the same
// trips table via a new file instead of an edit to that one.
type DeviceRouteRepository struct {
	db *pgxpool.Pool
}

// NewDeviceRouteRepository creates a new DeviceRouteRepository.
func NewDeviceRouteRepository(db *pgxpool.Pool) *DeviceRouteRepository {
	return &DeviceRouteRepository{db: db}
}

// GetActiveRouteID returns the route_id of the device's current IN_PROGRESS
// trip, or "" if the device has none.
func (r *DeviceRouteRepository) GetActiveRouteID(ctx context.Context, deviceID string) (string, error) {
	var routeID string
	err := r.db.QueryRow(ctx,
		`SELECT route_id FROM trips
		 WHERE device_id = $1 AND status = 'IN_PROGRESS'
		 ORDER BY started_at DESC LIMIT 1`,
		deviceID,
	).Scan(&routeID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		return "", fmt.Errorf("get active route for device %s: %w", deviceID, err)
	}
	return routeID, nil
}
