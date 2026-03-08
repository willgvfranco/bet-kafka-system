package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisMarketSuspensionCache struct {
	client *redis.Client
}

func NewRedisMarketSuspensionCache(client *redis.Client) *RedisMarketSuspensionCache {
	return &RedisMarketSuspensionCache{client: client}
}

func (c *RedisMarketSuspensionCache) IsSuspended(ctx context.Context, marketID string) (bool, error) {
	exists, err := c.client.Exists(ctx, "market_suspended:"+marketID).Result()
	return exists > 0, err
}

func (c *RedisMarketSuspensionCache) Suspend(ctx context.Context, marketID string, duration time.Duration) error {
	return c.client.Set(ctx, "market_suspended:"+marketID, "1", duration).Err()
}
