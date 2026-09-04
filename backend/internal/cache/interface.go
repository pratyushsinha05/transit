package cache

import (
	"time"
)

// DeviceLocation represents cached device location data
type DeviceLocation struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Speed     float64 `json:"speed"`
	HexRes9   string  `json:"hex_res9,omitempty"`
	LastSeen  int64   `json:"last_seen"`
}

// TTL constants - every key has a TTL, no permanent keys
const (
	// DeviceLocationTTL is the TTL for device location cache entries
	// 5 minutes is sufficient as bus locations update every few seconds
	DeviceLocationTTL = 5 * time.Minute
)

// Key prefixes for cache organization
const (
	KeyPrefixDevice = "device:"
)
