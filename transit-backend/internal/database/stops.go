package database

import (
	"context"
	"transit-backend/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

type StopRepository struct {
	db *pgxpool.Pool
}

func NewStopRepository(db *pgxpool.Pool) *StopRepository {
	return &StopRepository{db: db}
}

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
