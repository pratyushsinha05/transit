package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// ZoneHandler serves zone lookups. Depends only on the ZonesService
// interface (CLAUDE.md Sec 3.2) -- no repository imports.
type ZoneHandler struct {
	service ZonesService
}

func NewZoneHandler(service ZonesService) *ZoneHandler {
	return &ZoneHandler{service: service}
}

func (h *ZoneHandler) GetZones(c echo.Context) error {
	routeID := c.QueryParam("route_id")
	if routeID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "route_id is required"})
	}

	zones, err := h.service.GetZonesByRoute(c.Request().Context(), routeID)
	if err != nil {
		c.Logger().Error(err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}

	return c.JSON(http.StatusOK, zones)
}
