package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"transit-backend/internal/config"

	"github.com/redis/go-redis/v9"
)

// RedisCache implements Redis caching operations
type RedisCache struct {
	client *redis.Client
}

// New creates a new Redis client with optimized settings
func New(cfg *config.Config) (*RedisCache, error) {
	addr := fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort)

	rdb := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     cfg.RedisPassword,
		DB:           0,
		PoolSize:     50,              // Match DB pool for consistency
		MinIdleConns: 10,              // Keep warm connections
		MaxRetries:   3,               // Retry on transient failures
		DialTimeout:  5 * time.Second, // Connection timeout
		ReadTimeout:  3 * time.Second, // Read timeout
		WriteTimeout: 3 * time.Second, // Write timeout
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("unable to connect to redis: %w", err)
	}

	return &RedisCache{client: rdb}, nil
}

// --- Device Location Operations ---

// SetDeviceLocation caches device location with 5-minute TTL
func (c *RedisCache) SetDeviceLocation(ctx context.Context, deviceID string, loc *DeviceLocation) error {
	key := fmt.Sprintf("%s%s:loc", KeyPrefixDevice, deviceID)

	data, err := json.Marshal(loc)
	if err != nil {
		return fmt.Errorf("failed to marshal location: %w", err)
	}

	return c.client.Set(ctx, key, data, DeviceLocationTTL).Err()
}

// --- Health & Lifecycle ---

// Ping checks Redis connection health
func (c *RedisCache) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

// Close closes the Redis connection
func (c *RedisCache) Close() error {
	return c.client.Close()
}

// --- Legacy Compatibility ---

// DeviceCache is kept for backward compatibility with existing handlers
type DeviceCache struct {
	rdb *RedisCache
}

// NewDeviceCache creates a DeviceCache wrapper for backward compatibility
func NewDeviceCache(rdb *RedisCache) *DeviceCache {
	return &DeviceCache{rdb: rdb}
}

// SetDeviceState maintains backward compatibility with existing handler code
func (c *DeviceCache) SetDeviceState(ctx context.Context, deviceID string, state map[string]interface{}) error {
	// Extract location data from state map
	loc := &DeviceLocation{}

	if lat, ok := state["latitude"].(float64); ok {
		loc.Latitude = lat
	}
	if lng, ok := state["longitude"].(float64); ok {
		loc.Longitude = lng
	}
	if speed, ok := state["speed"].(float64); ok {
		loc.Speed = speed
	}
	if hex, ok := state["hex_res9"].(string); ok {
		loc.HexRes9 = hex
	}
	if lastSeen, ok := state["last_seen"].(int64); ok {
		loc.LastSeen = lastSeen
	}

	return c.rdb.SetDeviceLocation(ctx, deviceID, loc)
}
