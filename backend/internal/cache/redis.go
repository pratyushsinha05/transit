package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"transit-backend/internal/config"

	"github.com/redis/go-redis/v9"
)

// RedisCache implements CacheStore interface using Redis
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

// GetDeviceLocation retrieves cached device location
// Returns nil, nil if key doesn't exist or expired (graceful handling)
func (c *RedisCache) GetDeviceLocation(ctx context.Context, deviceID string) (*DeviceLocation, error) {
	key := fmt.Sprintf("%s%s:loc", KeyPrefixDevice, deviceID)

	val, err := c.client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return nil, nil // Key expired or not found - not an error
	}
	if err != nil {
		return nil, fmt.Errorf("redis get error: %w", err)
	}

	var loc DeviceLocation
	if err := json.Unmarshal([]byte(val), &loc); err != nil {
		return nil, fmt.Errorf("failed to unmarshal location: %w", err)
	}

	return &loc, nil
}

// DeleteDeviceLocation removes device location from cache
func (c *RedisCache) DeleteDeviceLocation(ctx context.Context, deviceID string) error {
	key := fmt.Sprintf("%s%s:loc", KeyPrefixDevice, deviceID)
	return c.client.Del(ctx, key).Err()
}

// --- Session Operations ---

// SetSession caches session with 30-minute TTL
func (c *RedisCache) SetSession(ctx context.Context, sessionID string, session *Session) error {
	key := fmt.Sprintf("%s%s", KeyPrefixSession, sessionID)

	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	return c.client.Set(ctx, key, data, SessionTTL).Err()
}

// GetSession retrieves cached session
// Returns nil, nil if session doesn't exist or expired
func (c *RedisCache) GetSession(ctx context.Context, sessionID string) (*Session, error) {
	key := fmt.Sprintf("%s%s", KeyPrefixSession, sessionID)

	val, err := c.client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return nil, nil // Session expired - not an error
	}
	if err != nil {
		return nil, fmt.Errorf("redis get error: %w", err)
	}

	var session Session
	if err := json.Unmarshal([]byte(val), &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	return &session, nil
}

// DeleteSession removes session from cache
func (c *RedisCache) DeleteSession(ctx context.Context, sessionID string) error {
	key := fmt.Sprintf("%s%s", KeyPrefixSession, sessionID)
	return c.client.Del(ctx, key).Err()
}

// --- Metric Operations ---

// SetMetric caches a metric with specified TTL
func (c *RedisCache) SetMetric(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	fullKey := fmt.Sprintf("%s%s", KeyPrefixMetric, key)

	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal metric: %w", err)
	}

	return c.client.Set(ctx, fullKey, data, ttl).Err()
}

// GetMetric retrieves a cached metric as string
// Returns empty string and nil if not found
func (c *RedisCache) GetMetric(ctx context.Context, key string) (string, error) {
	fullKey := fmt.Sprintf("%s%s", KeyPrefixMetric, key)

	val, err := c.client.Get(ctx, fullKey).Result()
	if errors.Is(err, redis.Nil) {
		return "", nil // Metric not found - not an error
	}
	if err != nil {
		return "", fmt.Errorf("redis get error: %w", err)
	}

	return val, nil
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
// Deprecated: Use CacheStore interface methods instead
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

// GetDeviceState maintains backward compatibility
func (c *DeviceCache) GetDeviceState(ctx context.Context, deviceID string) (map[string]interface{}, error) {
	loc, err := c.rdb.GetDeviceLocation(ctx, deviceID)
	if err != nil {
		return nil, err
	}
	if loc == nil {
		return nil, nil
	}

	return map[string]interface{}{
		"latitude":  loc.Latitude,
		"longitude": loc.Longitude,
		"speed":     loc.Speed,
		"hex_res9":  loc.HexRes9,
		"last_seen": loc.LastSeen,
	}, nil
}

// Ensure RedisCache implements CacheStore interface
var _ CacheStore = (*RedisCache)(nil)
