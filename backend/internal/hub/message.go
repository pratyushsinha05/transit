package hub

// Message defines the structure of messages broadcast to clients over the
// WebSocket connection. Canonical shape: CLAUDE.md Sec 7.2. Flat, no `data`
// wrapper, no `omitempty` on numerics -- a stopped device must serialize
// "speed": 0, not omit the key.
type Message struct {
	Type      string  `json:"type"`
	DeviceID  string  `json:"device_id"`
	RouteID   string  `json:"route_id"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Speed     float64 `json:"speed"`
	Accuracy  float64 `json:"accuracy"`
	H3Hex     string  `json:"h3_hex"`
	Timestamp int64   `json:"timestamp"`
}

const (
	MsgTypeLocationUpdate = "LOCATION_UPDATE"
)
