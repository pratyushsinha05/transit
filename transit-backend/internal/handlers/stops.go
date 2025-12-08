package handlers

import (
	"net/http"
	"transit-backend/internal/database"

	"github.com/labstack/echo/v4"
)

type StopHandler struct {
	repo *database.StopRepository
}

func NewStopHandler(repo *database.StopRepository) *StopHandler {
	return &StopHandler{repo: repo}
}

func (h *StopHandler) GetStops(c echo.Context) error {
	routeID := c.QueryParam("route_id")
	if routeID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "route_id is required"})
	}

	stops, err := h.repo.GetByRouteID(c.Request().Context(), routeID)
	if err != nil {
		c.Logger().Error(err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}

	return c.JSON(http.StatusOK, stops)
}
