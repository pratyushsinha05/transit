package models

type Route struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// CreateRouteRequest is the payload for POST /api/routes
type CreateRouteRequest struct {
	Name        string            `json:"name" validate:"required"`
	Description string            `json:"description"`
	Zones       []CreateZoneInput `json:"stops" validate:"required,min=2"`
}

// CreateZoneInput represents a single zone in the create route request
type CreateZoneInput struct {
	Name      string  `json:"name" validate:"required"`
	Latitude  float64 `json:"latitude" validate:"required"`
	Longitude float64 `json:"longitude" validate:"required"`
}

// CreateRouteResponse is returned after successful route creation
type CreateRouteResponse struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	ZoneCount   int      `json:"stop_count"`
	ZoneIDs     []string `json:"stop_ids"`
}
