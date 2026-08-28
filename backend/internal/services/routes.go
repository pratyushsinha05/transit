package services

import (
	"context"

	"transit-backend/internal/models"
)

// RoutesService handles route listing and creation.
type RoutesService struct {
	routeRepo RouteRepository
}

// NewRoutesService creates a new RoutesService.
func NewRoutesService(routeRepo RouteRepository) *RoutesService {
	return &RoutesService{routeRepo: routeRepo}
}

// GetRoutes returns all routes.
func (s *RoutesService) GetRoutes(ctx context.Context) ([]models.Route, error) {
	return s.routeRepo.GetAll(ctx)
}

// CreateRoute creates a new route with its stops.
func (s *RoutesService) CreateRoute(ctx context.Context, req models.CreateRouteRequest) (*models.CreateRouteResponse, error) {
	return s.routeRepo.Create(ctx, req)
}
