package database

import (
	"context"
	"fmt"
	"transit-backend/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

type RouteRepository struct {
	db *pgxpool.Pool
}

func NewRouteRepository(db *pgxpool.Pool) *RouteRepository {
	return &RouteRepository{db: db}
}

func (r *RouteRepository) GetAll(ctx context.Context) ([]models.Route, error) {
	query := `SELECT id, name, description FROM routes`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var routes []models.Route
	for rows.Next() {
		var rt models.Route
		if err := rows.Scan(&rt.ID, &rt.Name, &rt.Description); err != nil {
			return nil, err
		}
		routes = append(routes, rt)
	}
	return routes, nil
}

// Create inserts a route and its stops in a single transaction.
// It auto-generates route and stop IDs and populates the PostGIS geom column.
func (r *RouteRepository) Create(ctx context.Context, req models.CreateRouteRequest) (*models.CreateRouteResponse, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Generate route ID
	var routeID string
	err = tx.QueryRow(ctx,
		`INSERT INTO routes (id, name, description) VALUES (
			'route-' || substr(md5(random()::text), 1, 8),
			$1, $2
		) RETURNING id`,
		req.Name, req.Description,
	).Scan(&routeID)
	if err != nil {
		return nil, fmt.Errorf("insert route: %w", err)
	}

	// Insert stops with sequence numbers and PostGIS geom
	stopIDs := make([]string, 0, len(req.Stops))
	for i, stop := range req.Stops {
		var stopID string
		err = tx.QueryRow(ctx,
			`INSERT INTO stops (id, route_id, name, latitude, longitude, sequence_number, geom)
			 VALUES (
				'stop-' || substr(md5(random()::text), 1, 8),
				$1, $2, $3::numeric, $4::numeric, $5,
				ST_SetSRID(ST_MakePoint($4::numeric, $3::numeric), 4326)
			 ) RETURNING id`,
			routeID, stop.Name, stop.Latitude, stop.Longitude, i+1,
		).Scan(&stopID)
		if err != nil {
			return nil, fmt.Errorf("insert stop %d: %w", i+1, err)
		}
		stopIDs = append(stopIDs, stopID)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return &models.CreateRouteResponse{
		ID:          routeID,
		Name:        req.Name,
		Description: req.Description,
		StopCount:   len(req.Stops),
		StopIDs:     stopIDs,
	}, nil
}
