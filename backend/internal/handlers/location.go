package handlers

import (
	"net/http"
	"time"

	"transit-backend/internal/models"

	"github.com/labstack/echo/v4"
)

// LocationHandler handles GPS location ingestion. Depends only on the
// IngestService interface (CLAUDE.md Sec 3.2) -- no repository, cache, or
// hub imports.
type LocationHandler struct {
	service IngestService
}

// NewLocationHandler creates a new LocationHandler
func NewLocationHandler(service IngestService) *LocationHandler {
	return &LocationHandler{service: service}
}

// IngestLocation handles POST /api/location
// Receives GPS data, validates it, then delegates storage, cache, and
// broadcast to IngestService.
func (h *LocationHandler) IngestLocation(c echo.Context) error {
	var loc models.Location
	if err := c.Bind(&loc); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	// Validation
	if loc.DeviceID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "device_id is required"})
	}
	if loc.Latitude < -90 || loc.Latitude > 90 || loc.Longitude < -180 || loc.Longitude > 180 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid coordinates"})
	}
	if loc.Speed < 0 || loc.Speed > 350 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid speed"})
	}

	// Set timestamp if missing
	if loc.Timestamp == 0 {
		loc.Timestamp = time.Now().Unix()
	}

	// IngestService.IngestLocation mutates loc.HexRes9 as a side effect
	// (calculated during Insert), which is why we still have it below.
	if err := h.service.IngestLocation(c.Request().Context(), &loc); err != nil {
		c.Logger().Error(err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}

	// Return success with H3 hex
	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":   "ok",
		"hex_res9": loc.HexRes9,
	})
}
