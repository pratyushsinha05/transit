package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// StopHandler serves stop lookups. Depends only on the StopsService
// interface (CLAUDE.md Sec 3.2) -- no repository imports.
type StopHandler struct {
	service StopsService
}

func NewStopHandler(service StopsService) *StopHandler {
	return &StopHandler{service: service}
}

func (h *StopHandler) GetStops(c echo.Context) error {
	routeID := c.QueryParam("route_id")
	if routeID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "route_id is required"})
	}

	stops, err := h.service.GetStopsByRoute(c.Request().Context(), routeID)
	if err != nil {
		c.Logger().Error(err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}

	return c.JSON(http.StatusOK, stops)
}
