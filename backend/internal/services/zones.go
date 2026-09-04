package services

import (
	"context"

	"transit-backend/internal/models"
)

// ZonesService handles zone lookups.
type ZonesService struct {
	zoneRepo ZoneRepository
}

// NewZonesService creates a new ZonesService.
func NewZonesService(zoneRepo ZoneRepository) *ZonesService {
	return &ZonesService{zoneRepo: zoneRepo}
}

// GetZonesByRoute returns all zones for a route, ordered by sequence.
func (s *ZonesService) GetZonesByRoute(ctx context.Context, routeID string) ([]models.Zone, error) {
	return s.zoneRepo.GetByRouteID(ctx, routeID)
}
