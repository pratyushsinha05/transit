package models

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
