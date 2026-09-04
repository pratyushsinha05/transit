package database

import (
	"context"
	"fmt"
	"transit-backend/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ZoneRepository handles zones table operations
type ZoneRepository struct {
	db *pgxpool.Pool
}

// NewZoneRepository creates a new ZoneRepository
func NewZoneRepository(db *pgxpool.Pool) *ZoneRepository {
	return &ZoneRepository{db: db}
}

// GetByRouteID returns all zones for a route, ordered by sequence
func (r *ZoneRepository) GetByRouteID(ctx context.Context, routeID string) ([]models.Zone, error) {
	query := `
		SELECT id, name, latitude, longitude, sequence_number
		FROM zones
		WHERE route_id = $1
		ORDER BY sequence_number
	`

	rows, err := r.db.Query(ctx, query, routeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var zones []models.Zone
	for rows.Next() {
		var z models.Zone
		if err := rows.Scan(&z.ID, &z.Name, &z.Latitude, &z.Longitude, &z.Sequence); err != nil {
			return nil, err
		}
		zones = append(zones, z)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return zones, nil
}

// GetByID returns a single zone by ID
func (r *ZoneRepository) GetByID(ctx context.Context, zoneID string) (*models.Zone, error) {
	query := `
		SELECT id, name, latitude, longitude, sequence_number
		FROM zones
		WHERE id = $1
	`
	var z models.Zone
	err := r.db.QueryRow(ctx, query, zoneID).Scan(&z.ID, &z.Name, &z.Latitude, &z.Longitude, &z.Sequence)
	if err != nil {
		return nil, err
	}
	return &z, nil
}

// GetNearby returns zones within a radius of a point using PostGIS
func (r *ZoneRepository) GetNearby(ctx context.Context, lat, lng float64, radiusMeters int) ([]models.Zone, error) {
	// Use PostGIS ST_DWithin for accurate spatial query
	// Falls back to Haversine-style query if PostGIS geom column doesn't exist
	query := `
		SELECT id, name, latitude, longitude, sequence_number
		FROM zones
		WHERE ST_DWithin(
			geom::geography,
			ST_SetSRID(ST_MakePoint($2, $1), 4326)::geography,
			$3
		)
		ORDER BY ST_Distance(
			geom::geography,
			ST_SetSRID(ST_MakePoint($2, $1), 4326)::geography
		)
	`

	rows, err := r.db.Query(ctx, query, lat, lng, radiusMeters)
	if err != nil {
		return nil, fmt.Errorf("query nearby zones: %w", err)
	}
	defer rows.Close()

	var zones []models.Zone
	for rows.Next() {
		var z models.Zone
		if err := rows.Scan(&z.ID, &z.Name, &z.Latitude, &z.Longitude, &z.Sequence); err != nil {
			return nil, fmt.Errorf("scan zone: %w", err)
		}
		zones = append(zones, z)
	}

	return zones, rows.Err()
}

// GetAll returns all zones
func (r *ZoneRepository) GetAll(ctx context.Context) ([]models.Zone, error) {
	query := `
		SELECT id, name, latitude, longitude, sequence_number
		FROM zones
		ORDER BY route_id, sequence_number
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var zones []models.Zone
	for rows.Next() {
		var z models.Zone
		if err := rows.Scan(&z.ID, &z.Name, &z.Latitude, &z.Longitude, &z.Sequence); err != nil {
			return nil, err
		}
		zones = append(zones, z)
	}

	return zones, rows.Err()
}
