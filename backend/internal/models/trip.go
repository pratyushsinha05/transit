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
