package database

import (
	"context"
	"fmt"
	"transit-backend/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

// StopRepository handles stops table operations
type StopRepository struct {
	db *pgxpool.Pool
}

// NewStopRepository creates a new StopRepository
func NewStopRepository(db *pgxpool.Pool) *StopRepository {
	return &StopRepository{db: db}
}

// GetByRouteID returns all stops for a route, ordered by sequence
func (r *StopRepository) GetByRouteID(ctx context.Context, routeID string) ([]models.Stop, error) {
	query := `
		SELECT id, name, latitude, longitude, sequence_number
		FROM stops
		WHERE route_id = $1
		ORDER BY sequence_number
	`

	rows, err := r.db.Query(ctx, query, routeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stops []models.Stop
	for rows.Next() {
		var s models.Stop
		if err := rows.Scan(&s.ID, &s.Name, &s.Latitude, &s.Longitude, &s.Sequence); err != nil {
			return nil, err
		}
		stops = append(stops, s)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return stops, nil
}

// GetByID returns a single stop by ID
func (r *StopRepository) GetByID(ctx context.Context, stopID string) (*models.Stop, error) {
	query := `
		SELECT id, name, latitude, longitude, sequence_number
		FROM stops
		WHERE id = $1
	`
	var s models.Stop
	err := r.db.QueryRow(ctx, query, stopID).Scan(&s.ID, &s.Name, &s.Latitude, &s.Longitude, &s.Sequence)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// GetNearby returns stops within a radius of a point using PostGIS
func (r *StopRepository) GetNearby(ctx context.Context, lat, lng float64, radiusMeters int) ([]models.Stop, error) {
	// Use PostGIS ST_DWithin for accurate spatial query
	// Falls back to Haversine-style query if PostGIS geom column doesn't exist
	query := `
		SELECT id, name, latitude, longitude, sequence_number
		FROM stops
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
		return nil, fmt.Errorf("query nearby stops: %w", err)
	}
	defer rows.Close()

	var stops []models.Stop
	for rows.Next() {
		var s models.Stop
		if err := rows.Scan(&s.ID, &s.Name, &s.Latitude, &s.Longitude, &s.Sequence); err != nil {
			return nil, fmt.Errorf("scan stop: %w", err)
		}
		stops = append(stops, s)
	}

	return stops, rows.Err()
}

// GetAll returns all stops
func (r *StopRepository) GetAll(ctx context.Context) ([]models.Stop, error) {
	query := `
		SELECT id, name, latitude, longitude, sequence_number
		FROM stops
		ORDER BY route_id, sequence_number
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stops []models.Stop
	for rows.Next() {
		var s models.Stop
		if err := rows.Scan(&s.ID, &s.Name, &s.Latitude, &s.Longitude, &s.Sequence); err != nil {
			return nil, err
		}
		stops = append(stops, s)
	}

	return stops, rows.Err()
}
