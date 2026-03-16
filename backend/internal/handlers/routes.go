package handlers

import (
	"net/http"

	"transit-backend/internal/database"
	"transit-backend/internal/models"

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

// CreateRoute handles POST /api/routes
// Creates a new route with its stops
func (h *RouteHandler) CreateRoute(c echo.Context) error {
	var req models.CreateRouteRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	// Validate required fields
	if req.Name == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "name is required"})
	}
	if len(req.Stops) < 2 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "at least 2 stops are required"})
	}

	// Validate each stop has coordinates
	for i, stop := range req.Stops {
		if stop.Latitude == 0 && stop.Longitude == 0 {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "stop " + string(rune('1'+i)) + " has invalid coordinates",
			})
		}
	}

	result, err := h.repo.Create(c.Request().Context(), req)
	if err != nil {
		c.Logger().Error("Failed to create route:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create route"})
	}

	return c.JSON(http.StatusCreated, result)
}
