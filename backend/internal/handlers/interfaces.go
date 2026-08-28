package handlers

import (
	"context"

	"transit-backend/internal/services"
)

// ArrivalService is the subset of arrival-prediction behavior this handler
// needs. Declared here, in the consumer package, per CLAUDE.md Sec 7.1 ("A
// handler must depend on a service interface, not a concrete type").
type ArrivalService interface {
	GetArrivalsForStop(ctx context.Context, stopID string) ([]services.ArrivalPrediction, error)
}
