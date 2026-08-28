package models

type Trip struct {
	ID          string `json:"id"`
	RouteID     string `json:"route_id"`
	DeviceID    string `json:"device_id"`
	CurrentStop int    `json:"current_stop"`
	Status      string `json:"status"`
}

type ArrivalEvent struct {
	TripID     string `json:"trip_id"`
	DeviceID   string `json:"device_id"`
	DeviceName string `json:"device_name"`
	ETAMinutes int    `json:"eta_minutes"`
}

// TripWithLocation is an active trip joined with its device's latest
// known position. Lives in models, not database, so the services layer
// can declare its repository interface without importing database.
type TripWithLocation struct {
	TripID     string
	DeviceID   string
	DeviceName string
	Latitude   float64
	Longitude  float64
	Speed      float64
}
