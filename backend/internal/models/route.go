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
	Stops       []CreateStopInput `json:"stops" validate:"required,min=2"`
}

// CreateStopInput represents a single stop in the create route request
type CreateStopInput struct {
	Name      string  `json:"name" validate:"required"`
	Latitude  float64 `json:"latitude" validate:"required"`
	Longitude float64 `json:"longitude" validate:"required"`
}

// CreateRouteResponse is returned after successful route creation
type CreateRouteResponse struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	StopCount   int      `json:"stop_count"`
	StopIDs     []string `json:"stop_ids"`
}
