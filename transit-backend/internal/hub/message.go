package hub

// Message defines the structure of messages broadcasted to clients
type Message struct {
	Type      string      `json:"type"`
	Payload   interface{} `json:"payload,omitempty"`
	DeviceID  string      `json:"device_id,omitempty"`
	Latitude  float64     `json:"latitude,omitempty"`
	Longitude float64     `json:"longitude,omitempty"`
	Speed     float64     `json:"speed,omitempty"`
	Accuracy  float64     `json:"accuracy,omitempty"`
	Timestamp int64       `json:"timestamp,omitempty"`
}

const (
	MsgTypeLocationUpdate = "LOCATION_UPDATE"
)
