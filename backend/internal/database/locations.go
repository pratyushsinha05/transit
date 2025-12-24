package database

import (
	"context"
	"fmt"
	"time"
	"transit-backend/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/uber/h3-go/v4"
)

// LocationRepository handles location_history database operations
type LocationRepository struct {
	db         *pgxpool.Pool
	resolution int // H3 resolution (default: 9)
}

// NewLocationRepository creates a new LocationRepository
func NewLocationRepository(db *pgxpool.Pool) *LocationRepository {
	return &LocationRepository{
		db:         db,
		resolution: 9, // Resolution 9 = ~175m edge length, ideal for geofencing
	}
}

// NewLocationRepositoryWithResolution creates a LocationRepository with custom H3 resolution
func NewLocationRepositoryWithResolution(db *pgxpool.Pool, resolution int) *LocationRepository {
	return &LocationRepository{
		db:         db,
		resolution: resolution,
	}
}

// CalculateHex computes the H3 hex index for a lat/lng at the repository's resolution
// Returns empty string on error (invalid coordinates)
func (r *LocationRepository) CalculateHex(lat, lng float64) string {
	latLng := h3.NewLatLng(lat, lng)
	cell, err := h3.LatLngToCell(latLng, r.resolution)
	if err != nil {
		return ""
	}
	return cell.String()
}

// Insert stores a location record with H3 hex calculation
func (r *LocationRepository) Insert(ctx context.Context, loc *models.Location) error {
	// Calculate H3 hex if not already set
	if loc.HexRes9 == "" {
		loc.HexRes9 = r.CalculateHex(loc.Latitude, loc.Longitude)
	}

	// Convert timestamp - use current time if not provided
	ts := time.Unix(loc.Timestamp, 0)
	if loc.Timestamp == 0 {
		ts = time.Now()
	}

	// Insert with H3 hex and PostGIS geometry
	// ST_SetSRID(ST_MakePoint(lng, lat), 4326) creates a proper PostGIS point
	query := `
		INSERT INTO location_history (time, device_id, latitude, longitude, speed, accuracy, hex_res9, geom)
		VALUES ($1, $2, $3, $4, $5, $6, $7, ST_SetSRID(ST_MakePoint($4, $3), 4326))
	`

	_, err := r.db.Exec(ctx, query,
		ts,
		loc.DeviceID,
		loc.Latitude,
		loc.Longitude,
		loc.Speed,
		loc.Accuracy,
		loc.HexRes9,
	)
	return err
}

// GetBusesInHex returns all buses within a specific H3 hex (fast index scan)
// This is the first pass of the H3+PostGIS strategy
func (r *LocationRepository) GetBusesInHex(ctx context.Context, hex string, maxAgeMinutes int) ([]models.NearbyBus, error) {
	query := `
		SELECT DISTINCT ON (lh.device_id)
			lh.device_id,
			COALESCE(d.name, lh.device_id) as device_name,
			lh.latitude,
			lh.longitude,
			lh.speed,
			lh.hex_res9
		FROM location_history lh
		LEFT JOIN devices d ON lh.device_id = d.id
		WHERE lh.hex_res9 = $1
		  AND lh.time > NOW() - ($2 || ' minutes')::INTERVAL
		ORDER BY lh.device_id, lh.time DESC
	`

	rows, err := r.db.Query(ctx, query, hex, maxAgeMinutes)
	if err != nil {
		return nil, fmt.Errorf("query buses in hex: %w", err)
	}
	defer rows.Close()

	var buses []models.NearbyBus
	for rows.Next() {
		var b models.NearbyBus
		if err := rows.Scan(&b.DeviceID, &b.DeviceName, &b.Latitude, &b.Longitude, &b.Speed, &b.HexRes9); err != nil {
			return nil, fmt.Errorf("scan bus: %w", err)
		}
		buses = append(buses, b)
	}

	return buses, rows.Err()
}

// GetBusesInHexes returns buses in multiple H3 hexes (for k-ring queries)
func (r *LocationRepository) GetBusesInHexes(ctx context.Context, hexes []string, maxAgeMinutes int) ([]models.NearbyBus, error) {
	if len(hexes) == 0 {
		return nil, nil
	}

	query := `
		SELECT DISTINCT ON (lh.device_id)
			lh.device_id,
			COALESCE(d.name, lh.device_id) as device_name,
			lh.latitude,
			lh.longitude,
			lh.speed,
			lh.hex_res9
		FROM location_history lh
		LEFT JOIN devices d ON lh.device_id = d.id
		WHERE lh.hex_res9 = ANY($1)
		  AND lh.time > NOW() - ($2 || ' minutes')::INTERVAL
		ORDER BY lh.device_id, lh.time DESC
	`

	rows, err := r.db.Query(ctx, query, hexes, maxAgeMinutes)
	if err != nil {
		return nil, fmt.Errorf("query buses in hexes: %w", err)
	}
	defer rows.Close()

	var buses []models.NearbyBus
	for rows.Next() {
		var b models.NearbyBus
		if err := rows.Scan(&b.DeviceID, &b.DeviceName, &b.Latitude, &b.Longitude, &b.Speed, &b.HexRes9); err != nil {
			return nil, fmt.Errorf("scan bus: %w", err)
		}
		buses = append(buses, b)
	}

	return buses, rows.Err()
}

// GetBusesNearStop uses PostGIS for accurate distance queries
// This is the refinement pass of the H3+PostGIS strategy
func (r *LocationRepository) GetBusesNearStop(ctx context.Context, lat, lng float64, radiusMeters int, maxAgeMinutes int) ([]models.NearbyBus, error) {
	// Use PostGIS ST_DWithin for accurate spatial query with GIST index
	// ST_DWithin expects geography type (meters) rather than geometry (degrees)
	query := `
		SELECT DISTINCT ON (lh.device_id)
			lh.device_id,
			COALESCE(d.name, lh.device_id) as device_name,
			lh.latitude,
			lh.longitude,
			lh.speed,
			ST_Distance(
				lh.geom::geography,
				ST_SetSRID(ST_MakePoint($2, $1), 4326)::geography
			) as distance_meters,
			lh.hex_res9
		FROM location_history lh
		LEFT JOIN devices d ON lh.device_id = d.id
		WHERE ST_DWithin(
			lh.geom::geography,
			ST_SetSRID(ST_MakePoint($2, $1), 4326)::geography,
			$3
		)
		AND lh.time > NOW() - ($4 || ' minutes')::INTERVAL
		ORDER BY lh.device_id, lh.time DESC
	`

	rows, err := r.db.Query(ctx, query, lat, lng, radiusMeters, maxAgeMinutes)
	if err != nil {
		return nil, fmt.Errorf("query buses near stop: %w", err)
	}
	defer rows.Close()

	var buses []models.NearbyBus
	for rows.Next() {
		var b models.NearbyBus
		if err := rows.Scan(&b.DeviceID, &b.DeviceName, &b.Latitude, &b.Longitude, &b.Speed, &b.Distance, &b.HexRes9); err != nil {
			return nil, fmt.Errorf("scan bus: %w", err)
		}
		buses = append(buses, b)
	}

	return buses, rows.Err()
}

// GetLatestLocation returns the most recent location for a device
func (r *LocationRepository) GetLatestLocation(ctx context.Context, deviceID string) (*models.Location, error) {
	query := `
		SELECT device_id, latitude, longitude, speed, accuracy, 
			   EXTRACT(EPOCH FROM time)::bigint as timestamp, hex_res9
		FROM location_history
		WHERE device_id = $1
		ORDER BY time DESC
		LIMIT 1
	`

	var loc models.Location
	err := r.db.QueryRow(ctx, query, deviceID).Scan(
		&loc.DeviceID,
		&loc.Latitude,
		&loc.Longitude,
		&loc.Speed,
		&loc.Accuracy,
		&loc.Timestamp,
		&loc.HexRes9,
	)
	if err != nil {
		return nil, err
	}

	return &loc, nil
}

// GetNeighborHexes returns the H3 k-ring (neighboring hexes) for a location
func (r *LocationRepository) GetNeighborHexes(lat, lng float64, k int) []string {
	latLng := h3.NewLatLng(lat, lng)
	centerCell, err := h3.LatLngToCell(latLng, r.resolution)
	if err != nil {
		return nil
	}

	// GridDisk returns cells within k distance of the center
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
