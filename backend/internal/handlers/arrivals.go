package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// ArrivalHandler serves arrival predictions. It depends only on the
// ArrivalService interface (CLAUDE.md Sec 3.2) -- no repository imports.
type ArrivalHandler struct {
	service ArrivalService
}

func NewArrivalHandler(service ArrivalService) *ArrivalHandler {
	return &ArrivalHandler{service: service}
}

// GetArrivals handles GET /api/arrivals?stop_id=...
// Returns []services.ArrivalPrediction: trip_id, device_id, device_name,
// eta_minutes, distance_km, current_speed, hex_res9, is_approaching.
func (h *ArrivalHandler) GetArrivals(c echo.Context) error {
	stopID := c.QueryParam("stop_id")
	if stopID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "stop_id is required"})
	}

	predictions, err := h.service.GetArrivalsForStop(c.Request().Context(), stopID)
	if err != nil {
		c.Logger().Error("failed to get arrivals: ", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}

	return c.JSON(http.StatusOK, predictions)
}
