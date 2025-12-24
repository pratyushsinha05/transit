package handlers

import (
	"net/http"

	"transit-backend/internal/database"

	"github.com/labstack/echo/v4"
)

// RouteHandler handles route-related endpoints
type RouteHandler struct {
	repo *database.RouteRepository
}

// NewRouteHandler creates a new RouteHandler
func NewRouteHandler(repo *database.RouteRepository) *RouteHandler {
	return &RouteHandler{repo: repo}
}

// GetRoutes handles GET /api/routes
// Returns all routes from the database
func (h *RouteHandler) GetRoutes(c echo.Context) error {
	routes, err := h.repo.GetAll(c.Request().Context())
	if err != nil {
		c.Logger().Error(err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}
	return c.JSON(http.StatusOK, routes)
}
