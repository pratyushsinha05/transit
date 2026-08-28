package services

import (
	"context"

	"transit-backend/internal/database"
	"transit-backend/internal/models"
)

// StopRepository is the subset of stop-storage behavior the services layer
// needs. Declared here, in the consumer package, per CLAUDE.md Sec 7.1 ("A
// service must depend on a repository interface, not a concrete type").
type StopRepository interface {
	GetByID(ctx context.Context, stopID string) (*models.Stop, error)
	GetNearby(ctx context.Context, lat, lng float64, radiusMeters int) ([]models.Stop, error)
}

// TripRepository is the subset of trip-storage behavior the services layer needs.
type TripRepository interface {
	GetActiveTripsBeforeStop(ctx context.Context, stopSequence int) ([]database.TripWithLocation, error)
}

// LocationRepository is the subset of location-storage behavior the services
// layer needs.
type LocationRepository interface {
	GetLatestLocation(ctx context.Context, deviceID string) (*models.Location, error)
	GetBusesInHexes(ctx context.Context, hexes []string, maxAgeMinutes int) ([]models.NearbyBus, error)
	GetBusesNearStop(ctx context.Context, lat, lng float64, radiusMeters int, maxAgeMinutes int) ([]models.NearbyBus, error)
}
