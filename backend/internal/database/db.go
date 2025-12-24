package database

import (
	"context"
	"fmt"
	"time"

	"transit-backend/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

// New creates a new PostgreSQL connection pool with optimized settings
func New(cfg *config.Config) (*pgxpool.Pool, error) {
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName)

	poolConfig, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, fmt.Errorf("unable to parse database config: %w", err)
	}

	// Connection pool settings optimized for high concurrency
	// With 1000+ WebSocket clients sending location updates:
	// - Each update hits DB once
	// - 50 max conns handles bursts while preventing DB overload
	// - Connection reuse via pool is critical for latency
	poolConfig.MaxConns = int32(cfg.DBMaxConns)
	poolConfig.MinConns = int32(cfg.DBMinConns)
	poolConfig.MaxConnLifetime = 1 * time.Hour     // Recycle connections hourly
	poolConfig.MaxConnIdleTime = 30 * time.Minute  // Close idle connections after 30min
	poolConfig.HealthCheckPeriod = 1 * time.Minute // Periodic health checks

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}

	return pool, nil
}
