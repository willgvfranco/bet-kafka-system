package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisIdempotencyCache struct {
	client *redis.Client
}

func NewRedisIdempotencyCache(client *redis.Client) *RedisIdempotencyCache {
	return &RedisIdempotencyCache{client: client}
}

func (c *RedisIdempotencyCache) Check(ctx context.Context, key string) (bool, error) {
	set, err := c.client.SetNX(ctx, "idempotency:"+key, "processed", 60*time.Second).Result()
	if err != nil {
		return false, err
	}
	// Returns true if already processed (key existed), false if new
	return !set, nil
}
