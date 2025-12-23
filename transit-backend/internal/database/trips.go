package database

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type TripRepository struct {
	db *pgxpool.Pool
}

func NewTripRepository(db *pgxpool.Pool) *TripRepository {
	return &TripRepository{db: db}
}

type TripWithLocation struct {
	TripID     string
	DeviceID   string
	DeviceName string // In a real app we'd join with devices table
	Latitude   float64
	Longitude  float64
	Speed      float64
}

func (r *TripRepository) GetActiveTripsBeforeStop(ctx context.Context, stopSequence int) ([]TripWithLocation, error) {
	// Logic:
	// 1. Get active trips (current_stop < sequence)
	// 2. Lateral join with location_history to get latest position
	query := `
		SELECT 
            t.id, t.device_id, d.name,
            lh.latitude, lh.longitude, lh.speed
        FROM trips t
        JOIN devices d ON t.device_id = d.id
        LEFT JOIN LATERAL (
            SELECT latitude, longitude, speed
            FROM location_history
            WHERE device_id = t.device_id
            ORDER BY time DESC
            LIMIT 1
        ) lh ON true
        WHERE t.status = 'IN_PROGRESS' 
          AND t.current_stop < $1
          AND lh.latitude IS NOT NULL
	`

	rows, err := r.db.Query(ctx, query, stopSequence)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []TripWithLocation
	for rows.Next() {
		var t TripWithLocation
		if err := rows.Scan(&t.TripID, &t.DeviceID, &t.DeviceName, &t.Latitude, &t.Longitude, &t.Speed); err != nil {
			return nil, err
		}
		results = append(results, t)
	}

	return results, nil
}
