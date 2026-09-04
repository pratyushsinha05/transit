package models

// Location represents a GPS location update from a device
type Location struct {
	DeviceID  string  `json:"device_id"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Speed     float64 `json:"speed"`
	Accuracy  float64 `json:"accuracy"`
	Timestamp int64   `json:"timestamp"`
	HexRes9   string  `json:"hex_res9,omitempty"` // H3 hex at resolution 9 (~175m)
}

// LocationUpdate represents a WebSocket location broadcast message
type LocationUpdate struct {
	Type      string  `json:"type"`
	DeviceID  string  `json:"device_id"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Speed     float64 `json:"speed"`
	Accuracy  float64 `json:"accuracy"`
	Timestamp int64   `json:"timestamp"`
	HexRes9   string  `json:"hex_res9,omitempty"`
}

// NearbyDevice represents a device near a specific location
type NearbyDevice struct {
	DeviceID   string  `json:"device_id"`
	DeviceName string  `json:"device_name,omitempty"`
	Latitude   float64 `json:"latitude"`
	Longitude  float64 `json:"longitude"`
	Speed      float64 `json:"speed"`
	Distance   float64 `json:"distance_meters"` // Distance from query point
	HexRes9    string  `json:"hex_res9,omitempty"`
}
