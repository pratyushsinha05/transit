package cache

import (
	"context"
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

// CacheStore defines the interface for cache operations
type CacheStore interface {
	// Device location operations (TTL: 5 minutes)
	SetDeviceLocation(ctx context.Context, deviceID string, loc *DeviceLocation) error

	// Health check
	Ping(ctx context.Context) error

	// Close connection
	Close() error
}

// TTL constants - every key has a TTL, no permanent keys
const (
	// DeviceLocationTTL is the TTL for device location cache entries
	// 5 minutes is sufficient as bus locations update every few seconds
	DeviceLocationTTL = 5 * time.Minute

	// NearbyDevicesTTL is the TTL for cached nearby devices results
	NearbyDevicesTTL = 10 * time.Second
)

// Key prefixes for cache organization
const (
	KeyPrefixDevice  = "device:"
	KeyPrefixSession = "session:"
	KeyPrefixMetric  = "metric:"
	KeyPrefixGeo     = "geo:"
)
