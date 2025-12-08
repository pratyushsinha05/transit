package handlers

import (
	"net/http"
	"time"

	"transit-backend/internal/cache"
	"transit-backend/internal/database"
	"transit-backend/internal/hub"
	"transit-backend/internal/models"

	"github.com/labstack/echo/v4"
)

type LocationHandler struct {
	repo  *database.LocationRepository
	cache *cache.DeviceCache
	hub   *hub.Hub
}

func NewLocationHandler(repo *database.LocationRepository, cache *cache.DeviceCache, hub *hub.Hub) *LocationHandler {
	return &LocationHandler{repo: repo, cache: cache, hub: hub}
}

func (h *LocationHandler) IngestLocation(c echo.Context) error {
	var loc models.Location
	if err := c.Bind(&loc); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	// Validation
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

	// Insert into DB
	if err := h.repo.Insert(c.Request().Context(), &loc); err != nil {
		c.Logger().Error(err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}

	// Update Cache
	deviceState := map[string]interface{}{
		"latitude":  loc.Latitude,
		"longitude": loc.Longitude,
		"speed":     loc.Speed,
		"last_seen": loc.Timestamp,
	}
	if err := h.cache.SetDeviceState(c.Request().Context(), loc.DeviceID, deviceState); err != nil {
		c.Logger().Error("failed to update cache: ", err)
		// Don't fail the request if cache update fails
	}

	// Broadcast to WebSocket
	msg := hub.Message{
		Type:      hub.MsgTypeLocationUpdate,
		DeviceID:  loc.DeviceID,
		Latitude:  loc.Latitude,
		Longitude: loc.Longitude,
		Speed:     loc.Speed,
		Accuracy:  loc.Accuracy,
		Timestamp: loc.Timestamp,
	}
	select {
	case h.hub.Broadcast <- msg:
	default:
		// Drop message if buffer full
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}
