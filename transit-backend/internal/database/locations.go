package database

import (
	"context"
	"time"
	"transit-backend/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

type LocationRepository struct {
	db *pgxpool.Pool
}

func NewLocationRepository(db *pgxpool.Pool) *LocationRepository {
	return &LocationRepository{db: db}
}

func (r *LocationRepository) Insert(ctx context.Context, loc *models.Location) error {
	query := `
		INSERT INTO location_history (time, device_id, latitude, longitude, speed, accuracy)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	// Convert timestamp int64 to time.Time
	ts := time.Unix(loc.Timestamp, 0)
	// If timestamp is 0 or very old, use current time?
	// The prompt sends timestamp in payload, let's use it or default to Now if missing/zero
	if loc.Timestamp == 0 {
		ts = time.Now()
	}

	_, err := r.db.Exec(ctx, query, ts, loc.DeviceID, loc.Latitude, loc.Longitude, loc.Speed, loc.Accuracy)
	return err
}
