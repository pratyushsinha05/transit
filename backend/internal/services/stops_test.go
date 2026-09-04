package services

import (
	"context"
	"fmt"
	"testing"
	"transit-backend/internal/models"
)

func TestZonesService_GetZonesByRoute(t *testing.T) {
	ctx := context.Background()
	mock := &mockZoneRepo{
		getByRouteIDFn: func(ctx context.Context, routeID string) ([]models.Zone, error) {
			if routeID == "route-1" {
				return []models.Zone{{ID: "s-1", Name: "Stop 1", Sequence: 1}}, nil
			}
			return nil, fmt.Errorf("route not found")
		},
	}

	svc := NewZonesService(mock)
	zones, err := svc.GetZonesByRoute(ctx, "route-1")
	if err != nil || len(zones) != 1 || zones[0].ID != "s-1" {
		t.Errorf("unexpected zones: %v, err: %v", zones, err)
	}

	_, err = svc.GetZonesByRoute(ctx, "route-unknown")
	if err == nil {
		t.Error("expected error for unknown route")
	}
}
