package handlers

import (
	"net/http"
	"strconv"

	"transit-backend/internal/services"

	"github.com/labstack/echo/v4"
)

// NearbyHandler handles nearby buses and stops queries. Depends only on the
// NearbyService interface (CLAUDE.md Sec 3.2) -- not the concrete
// *services.GeofencingService.
type NearbyHandler struct {
	geoService NearbyService
}

// NewNearbyHandler creates a new NearbyHandler
func NewNearbyHandler(geoService NearbyService) *NearbyHandler {
	return &NearbyHandler{geoService: geoService}
}

// GetNearbyDevices handles GET /api/nearby/devices
// Query params: lat, lng, radius (meters, default 500)
func (h *NearbyHandler) GetNearbyDevices(c echo.Context) error {
	// Parse query parameters
	latStr := c.QueryParam("lat")
	lngStr := c.QueryParam("lng")
	radiusStr := c.QueryParam("radius")

	if latStr == "" || lngStr == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "lat and lng query parameters are required",
		})
	}

	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil || lat < -90 || lat > 90 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid latitude",
		})
	}

	lng, err := strconv.ParseFloat(lngStr, 64)
	if err != nil || lng < -180 || lng > 180 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid longitude",
		})
	}

	// Default radius: 500 meters
	radius := 500
	if radiusStr != "" {
		r, err := strconv.Atoi(radiusStr)
		if err == nil && r > 0 && r <= 10000 {
			radius = r
		}
	}

	// Query nearby devices using H3+PostGIS strategy
	devices, err := h.geoService.FindNearbyDevices(c.Request().Context(), lat, lng, radius)
	if err != nil {
		c.Logger().Error("failed to find nearby devices: ", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "internal server error",
		})
	}

	// Handle nil/empty result
	deviceCount := 0
	if devices != nil {
		deviceCount = len(devices)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"buses":  devices,
		"count":  deviceCount,
		"radius": radius,
		"center": map[string]float64{
			"lat": lat,
			"lng": lng,
		},
	})
}

// GetNearbyStops handles GET /api/nearby/stops
// Query params: lat, lng, radius (meters, default 500)
func (h *NearbyHandler) GetNearbyZones(c echo.Context) error {
	// Parse query parameters
	latStr := c.QueryParam("lat")
	lngStr := c.QueryParam("lng")
	radiusStr := c.QueryParam("radius")

	if latStr == "" || lngStr == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "lat and lng query parameters are required",
		})
	}

	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil || lat < -90 || lat > 90 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid latitude",
		})
	}

	lng, err := strconv.ParseFloat(lngStr, 64)
	if err != nil || lng < -180 || lng > 180 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid longitude",
		})
	}

	// Default radius: 500 meters
	radius := 500
	if radiusStr != "" {
		r, err := strconv.Atoi(radiusStr)
		if err == nil && r > 0 && r <= 10000 {
			radius = r
		}
	}

	// Query nearby zones using PostGIS
	zones, err := h.geoService.FindNearbyZones(c.Request().Context(), lat, lng, radius)
	if err != nil {
		c.Logger().Error("failed to find nearby zones: ", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "internal server error",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"stops":  zones,
		"count":  len(zones),
		"radius": radius,
		"center": map[string]float64{
			"lat": lat,
			"lng": lng,
		},
	})
}

// GetHexInfo handles GET /api/geo/hex
// Returns H3 hex information for a coordinate
func (h *NearbyHandler) GetHexInfo(c echo.Context) error {
	latStr := c.QueryParam("lat")
	lngStr := c.QueryParam("lng")

	if latStr == "" || lngStr == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "lat and lng query parameters are required",
		})
	}

	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil || lat < -90 || lat > 90 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid latitude",
		})
	}

	lng, err := strconv.ParseFloat(lngStr, 64)
	if err != nil || lng < -180 || lng > 180 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid longitude",
		})
	}

	hex := h.geoService.CalculateHex(lat, lng)
	resolution := h.geoService.GetHexResolution()
	edgeLength := services.HexEdgeLengthMeters(resolution)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"hex":           hex,
		"resolution":    resolution,
		"edge_length_m": edgeLength,
		"coordinate": map[string]float64{
			"lat": lat,
			"lng": lng,
		},
	})
}
