package cache

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisOddsCache struct {
	client *redis.Client
}

func NewRedisOddsCache(client *redis.Client) *RedisOddsCache {
	return &RedisOddsCache{client: client}
}

func (c *RedisOddsCache) GetOdds(ctx context.Context, outcomeID string) (float64, error) {
	val, err := c.client.Get(ctx, oddsKey(outcomeID)).Result()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return strconv.ParseFloat(val, 64)
}

func (c *RedisOddsCache) SetOdds(ctx context.Context, outcomeID string, odds float64) error {
	return c.client.Set(ctx, oddsKey(outcomeID), fmt.Sprintf("%.2f", odds), 5*time.Minute).Err()
}

func (c *RedisOddsCache) GetEventOdds(ctx context.Context, eventID string) (map[string]float64, error) {
	members, err := c.client.SMembers(ctx, eventOddsSetKey(eventID)).Result()
	if err != nil {
		return nil, err
	}
	result := make(map[string]float64, len(members))
	for _, outcomeID := range members {
		odds, err := c.GetOdds(ctx, outcomeID)
		if err != nil {
			continue
		}
		if odds > 0 {
			result[outcomeID] = odds
		}
	}
	return result, nil
}

func (c *RedisOddsCache) PushOddsHistory(ctx context.Context, outcomeID string, changePercent float64) error {
	key := oddsHistoryKey(outcomeID)
	pipe := c.client.Pipeline()
	pipe.LPush(ctx, key, fmt.Sprintf("%.2f", changePercent))
	pipe.LTrim(ctx, key, 0, 9)
	_, err := pipe.Exec(ctx)
	return err
}

func (c *RedisOddsCache) GetOddsVelocity(ctx context.Context, outcomeID string) (float64, error) {
	vals, err := c.client.LRange(ctx, oddsHistoryKey(outcomeID), 0, 9).Result()
	if err != nil || len(vals) == 0 {
		return 0, err
	}
	var sum float64
	for _, v := range vals {
		f, _ := strconv.ParseFloat(v, 64)
		sum += math.Abs(f)
	}
	return sum / float64(len(vals)), nil
}

func oddsKey(outcomeID string) string        { return "odds:" + outcomeID }
func oddsHistoryKey(outcomeID string) string  { return "odds_history:" + outcomeID }
func eventOddsSetKey(eventID string) string   { return "event_odds:" + eventID }
