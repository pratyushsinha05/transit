package handlers

import (
	"net/http"
	"transit-backend/internal/models"

	"github.com/labstack/echo/v4"
)

// RoutesHandler handles route related endpoints
// Assuming simple static routes for now as per prompt or could extract from DB if exists
// The prompt said: Query all rows from routes table
type RoutesHandler struct {
    // Need DB access, but didn't implement RouteRepository yet.
    // The prompt says "Query all rows from routes table"
    // I should probably add a RouteRepository or just do it inline here with a DB pool if simple.
    // But better to be consistent. I will add a method to get all routes.
}

// Since I didn't create RouteRepository in the previous steps (Wait, I didn't?), I will mock it or add it.
// Actually, I missed creating `internal/database/routes.go` in the previous step.
// I will just use a direct query here for simplicity or define a small struct.
// Ah, the prompt didn't explicitly ask for `routes.go` in `database/` but `internal/handlers/routes.go` logic said "Query all rows".
// I will inject the pool directly into the handler or creating a simple repo wrapper here.
// Let's create a routes repository file later or include it here.
// I'll create `internal/database/routes.go` as well in this step implicitly or just put the logic here.
// I'll put the logic in `internal/database/routes.go` properly and create that file now.

import "transit-backend/internal/database"

type RouteHandler struct {
    repo *database.RouteRepository
}

func NewRouteHandler(repo *database.RouteRepository) *RouteHandler {
    return &RouteHandler{repo: repo}
}

func (h *RouteHandler) GetRoutes(c echo.Context) error {
    routes, err := h.repo.GetAll(c.Request().Context())
    if err != nil {
        c.Logger().Error(err)
        return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
    }
    return c.JSON(http.StatusOK, routes)
}
