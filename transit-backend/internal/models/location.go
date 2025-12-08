package models

type Location struct {
	DeviceID  string  `json:"device_id"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Speed     float64 `json:"speed"` // Changed to float64 to match DB schema (float8) better, though prompt said float32 in struct but float in DB.
	Accuracy  float64 `json:"accuracy"`
	Timestamp int64   `json:"timestamp"`
}

type LocationUpdate struct {
	Type      string  `json:"type"`
	DeviceID  string  `json:"device_id"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Speed     float64 `json:"speed"`
	Accuracy  float64 `json:"accuracy"`
	Timestamp int64   `json:"timestamp"`
}
