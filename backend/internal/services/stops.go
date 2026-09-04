package services

import (
	"context"

	"transit-backend/internal/models"
)

// StopsService handles stop lookups.
type StopsService struct {
	stopRepo StopRepository
}

// NewStopsService creates a new StopsService.
func NewStopsService(stopRepo StopRepository) *StopsService {
	return &StopsService{stopRepo: stopRepo}
}

// GetStopsByRoute returns all stops for a route, ordered by sequence.
func (s *StopsService) GetStopsByRoute(ctx context.Context, routeID string) ([]models.Zone, error) {
	return s.stopRepo.GetByRouteID(ctx, routeID)
}
