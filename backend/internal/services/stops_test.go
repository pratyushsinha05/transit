package services

import (
	"context"
	"fmt"
	"testing"
	"transit-backend/internal/models"
)

func TestStopsService_GetStopsByRoute(t *testing.T) {
	ctx := context.Background()
	mock := &mockStopRepo{
		getByRouteIDFn: func(ctx context.Context, routeID string) ([]models.Stop, error) {
			if routeID == "route-1" {
				return []models.Stop{{ID: "s-1", Name: "Stop 1", Sequence: 1}}, nil
			}
			return nil, fmt.Errorf("route not found")
		},
	}

	svc := NewStopsService(mock)
	stops, err := svc.GetStopsByRoute(ctx, "route-1")
	if err != nil || len(stops) != 1 || stops[0].ID != "s-1" {
		t.Errorf("unexpected stops: %v, err: %v", stops, err)
	}

	_, err = svc.GetStopsByRoute(ctx, "route-unknown")
	if err == nil {
		t.Error("expected error for unknown route")
	}
}
