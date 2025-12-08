package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	"transit-backend/internal/models"

	"github.com/redis/go-redis/v9"
)

type DeviceCache struct {
	rdb *redis.Client
}

func NewDeviceCache(rdb *redis.Client) *DeviceCache {
	return &DeviceCache{rdb: rdb}
}

func (c *DeviceCache) SetDeviceState(ctx context.Context, deviceID string, state map[string]interface{}) error {
	key := fmt.Sprintf("device:%s", deviceID)
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}

	return c.rdb.Set(ctx, key, data, time.Hour).Err()
}

func (c *DeviceCache) GetDeviceState(ctx context.Context, deviceID string) (map[string]interface{}, error) {
	key := fmt.Sprintf("device:%s", deviceID)
	val, err := c.rdb.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var state map[string]interface{}
	if err := json.Unmarshal([]byte(val), &state); err != nil {
		return nil, err
	}
	return state, nil
}
