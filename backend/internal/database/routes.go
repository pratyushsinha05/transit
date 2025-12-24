package database

import (
	"context"
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
