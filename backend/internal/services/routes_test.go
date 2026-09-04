package services

import (
	"context"
	"fmt"
	"testing"
	"transit-backend/internal/models"
)

type mockRouteRepo struct {
	getAllFn func(ctx context.Context) ([]models.Route, error)
	createFn func(ctx context.Context, req models.CreateRouteRequest) (*models.CreateRouteResponse, error)
}

func (m *mockRouteRepo) GetAll(ctx context.Context) ([]models.Route, error) {
	if m.getAllFn != nil {
		return m.getAllFn(ctx)
	}
	return nil, nil
}

func (m *mockRouteRepo) Create(ctx context.Context, req models.CreateRouteRequest) (*models.CreateRouteResponse, error) {
	if m.createFn != nil {
		return m.createFn(ctx, req)
	}
	return nil, nil
}

func TestRoutesService_GetRoutes(t *testing.T) {
	ctx := context.Background()
	mock := &mockRouteRepo{
		getAllFn: func(ctx context.Context) ([]models.Route, error) {
			return []models.Route{{ID: "r-1", Name: "Route 1"}}, nil
		},
	}

	svc := NewRoutesService(mock)
	routes, err := svc.GetRoutes(ctx)
	if err != nil || len(routes) != 1 || routes[0].ID != "r-1" {
		t.Errorf("unexpected result: %v, err: %v", routes, err)
	}

	// Error case
	mockErr := &mockRouteRepo{
		getAllFn: func(ctx context.Context) ([]models.Route, error) {
			return nil, fmt.Errorf("db error")
		},
	}
	svcErr := NewRoutesService(mockErr)
	_, err = svcErr.GetRoutes(ctx)
	if err == nil {
		t.Error("expected error from GetRoutes")
	}
}

func TestRoutesService_CreateRoute(t *testing.T) {
	ctx := context.Background()
	req := models.CreateRouteRequest{
		Name: "New Route",
		Zones: []models.CreateZoneInput{
			{Name: "Stop 1", Latitude: 28.6, Longitude: 77.2},
			{Name: "Stop 2", Latitude: 28.7, Longitude: 77.3},
		},
	}

	mock := &mockRouteRepo{
		createFn: func(ctx context.Context, r models.CreateRouteRequest) (*models.CreateRouteResponse, error) {
			return &models.CreateRouteResponse{
				ID:        "route-created-1",
				Name:      r.Name,
				ZoneCount: len(r.Zones),
				ZoneIDs:   []string{"s1", "s2"},
			}, nil
		},
	}

	svc := NewRoutesService(mock)
	res, err := svc.CreateRoute(ctx, req)
	if err != nil || res.ID != "route-created-1" || res.ZoneCount != 2 {
		t.Errorf("unexpected CreateRoute result: %+v, err: %v", res, err)
	}
}
