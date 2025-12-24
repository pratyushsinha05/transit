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

// Session represents cached session data
type Session struct {
	UserID    string `json:"user_id"`
	DeviceID  string `json:"device_id,omitempty"`
	CreatedAt int64  `json:"created_at"`
}

// CacheStore defines the interface for cache operations
// This abstraction allows swapping Redis for Memcached or other implementations
type CacheStore interface {
	// Device location operations (TTL: 5 minutes)
	SetDeviceLocation(ctx context.Context, deviceID string, loc *DeviceLocation) error
	GetDeviceLocation(ctx context.Context, deviceID string) (*DeviceLocation, error)
	DeleteDeviceLocation(ctx context.Context, deviceID string) error

	// Session operations (TTL: 30 minutes)
	SetSession(ctx context.Context, sessionID string, session *Session) error
	GetSession(ctx context.Context, sessionID string) (*Session, error)
	DeleteSession(ctx context.Context, sessionID string) error

	// Generic metric operations (variable TTL)
	SetMetric(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	GetMetric(ctx context.Context, key string) (string, error)

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

	// SessionTTL is the TTL for user session cache entries
	SessionTTL = 30 * time.Minute

	// MetricTTL is the default TTL for aggregated metrics
	MetricTTL = 1 * time.Minute

	// NearbyBusesTTL is the TTL for cached nearby buses results
	NearbyBusesTTL = 10 * time.Second
)

// Key prefixes for cache organization
const (
	KeyPrefixDevice  = "device:"
	KeyPrefixSession = "session:"
	KeyPrefixMetric  = "metric:"
	KeyPrefixGeo     = "geo:"
)
