package handlers

import (
	"net/http"
	"transit-backend/internal/database"
	"transit-backend/internal/models"
	"transit-backend/pkg/geo"

	"github.com/labstack/echo/v4"
)

type ArrivalHandler struct {
	stopRepo *database.StopRepository
	tripRepo *database.TripRepository
}

func NewArrivalHandler(stopRepo *database.StopRepository, tripRepo *database.TripRepository) *ArrivalHandler {
	return &ArrivalHandler{
		stopRepo: stopRepo,
		tripRepo: tripRepo,
	}
}

func (h *ArrivalHandler) GetArrivals(c echo.Context) error {
	stopID := c.QueryParam("stop_id")
	if stopID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "stop_id is required"})
	}

	// Get target stop details
	targetStop, err := h.stopRepo.GetByID(c.Request().Context(), stopID)
	if err != nil {
		c.Logger().Error("failed to get stop", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}

	// Get active trips before this stop
	trips, err := h.tripRepo.GetActiveTripsBeforeStop(c.Request().Context(), targetStop.Sequence)
	if err != nil {
		c.Logger().Error("failed to get trips", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}

	var arrivals []models.ArrivalEvent
	for _, trip := range trips {
		eta := geo.CalculateETA(trip.Latitude, trip.Longitude, targetStop.Latitude, targetStop.Longitude, trip.Speed)
		arrivals = append(arrivals, models.ArrivalEvent{
			TripID:     trip.TripID,
			DeviceID:   trip.DeviceID,
			DeviceName: trip.DeviceName,
			ETAMinutes: eta,
		})
	}

	return c.JSON(http.StatusOK, arrivals)
}
