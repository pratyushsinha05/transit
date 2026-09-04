package handlers

import (
	"context"

	"transit-backend/internal/models"
	"transit-backend/internal/services"
)

// GeofenceService is the subset of arrival-prediction behavior this handler
// needs. Declared here, in the consumer package, per CLAUDE.md Sec 7.1 ("A
// handler must depend on a service interface, not a concrete type").
type GeofenceService interface {
	GetPredictionsForZone(ctx context.Context, zoneID string) ([]services.GeofencePrediction, error)
}

// ZonesService is the subset of zone-lookup behavior the zones handler needs.
type ZonesService interface {
	GetZonesByRoute(ctx context.Context, routeID string) ([]models.Zone, error)
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
	FindNearbyDevices(ctx context.Context, lat, lng float64, radiusMeters int) ([]models.NearbyDevice, error)
	FindNearbyZones(ctx context.Context, lat, lng float64, radiusMeters int) ([]models.Zone, error)
	CalculateHex(lat, lng float64) string
	GetHexResolution() int
}
