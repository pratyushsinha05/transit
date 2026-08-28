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
	GetByRouteID(ctx context.Context, routeID string) ([]models.Stop, error)
	GetNearby(ctx context.Context, lat, lng float64, radiusMeters int) ([]models.Stop, error)
}

// TripRepository is the subset of trip-storage behavior the services layer needs.
type TripRepository interface {
	GetActiveTripsBeforeStop(ctx context.Context, stopSequence int) ([]database.TripWithLocation, error)
}

// LocationRepository is the subset of location-storage behavior the services
// layer needs.
type LocationRepository interface {
	Insert(ctx context.Context, loc *models.Location) error
	GetLatestLocation(ctx context.Context, deviceID string) (*models.Location, error)
	GetBusesInHexes(ctx context.Context, hexes []string, maxAgeMinutes int) ([]models.NearbyBus, error)
	GetBusesNearStop(ctx context.Context, lat, lng float64, radiusMeters int, maxAgeMinutes int) ([]models.NearbyBus, error)
}

// RouteRepository is the subset of route-storage behavior the services layer needs.
type RouteRepository interface {
	GetAll(ctx context.Context) ([]models.Route, error)
	Create(ctx context.Context, req models.CreateRouteRequest) (*models.CreateRouteResponse, error)
}

// DeviceCache is the subset of hot-state cache behavior the services layer
// needs. Satisfied by *cache.DeviceCache (cache/redis.go's legacy wrapper).
type DeviceCache interface {
	SetDeviceState(ctx context.Context, deviceID string, state map[string]interface{}) error
}
