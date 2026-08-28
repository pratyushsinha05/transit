package handlers

import (
	"context"

	"transit-backend/internal/models"
	"transit-backend/internal/services"
)

// ArrivalService is the subset of arrival-prediction behavior this handler
// needs. Declared here, in the consumer package, per CLAUDE.md Sec 7.1 ("A
// handler must depend on a service interface, not a concrete type").
type ArrivalService interface {
	GetArrivalsForStop(ctx context.Context, stopID string) ([]services.ArrivalPrediction, error)
}

// StopsService is the subset of stop-lookup behavior the stops handler needs.
type StopsService interface {
	GetStopsByRoute(ctx context.Context, routeID string) ([]models.Stop, error)
}

// RoutesService is the subset of route behavior the routes handler needs.
type RoutesService interface {
	GetRoutes(ctx context.Context) ([]models.Route, error)
	CreateRoute(ctx context.Context, req models.CreateRouteRequest) (*models.CreateRouteResponse, error)
}

// IngestService is the subset of ingest behavior the location handler needs.
type IngestService interface {
	IngestLocation(ctx context.Context, loc *models.Location) error
}

// NearbyService is the subset of geofencing behavior the nearby handler needs.
type NearbyService interface {
	FindNearbyBuses(ctx context.Context, lat, lng float64, radiusMeters int) ([]models.NearbyBus, error)
	FindNearbyStops(ctx context.Context, lat, lng float64, radiusMeters int) ([]models.Stop, error)
	CalculateHex(lat, lng float64) string
	GetHexResolution() int
}
