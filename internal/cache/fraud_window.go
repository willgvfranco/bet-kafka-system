package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/neus/bet-kafka-system/internal/domain"
	"github.com/redis/go-redis/v9"
)

type RedisFraudWindow struct {
	client *redis.Client
}

func NewRedisFraudWindow(client *redis.Client) *RedisFraudWindow {
	return &RedisFraudWindow{client: client}
}

func (c *RedisFraudWindow) AddBet(ctx context.Context, userID string, bet *domain.Bet) error {
	data, err := json.Marshal(bet)
	if err != nil {
		return err
	}
	key := fraudWindowKey(userID)
	score := float64(bet.PlacedAt.UnixMilli())
	pipe := c.client.Pipeline()
	pipe.ZAdd(ctx, key, redis.Z{Score: score, Member: string(data)})
	pipe.Expire(ctx, key, 10*time.Minute)
	_, err = pipe.Exec(ctx)
	return err
}

func (c *RedisFraudWindow) GetRecentBets(ctx context.Context, userID string, window time.Duration) ([]domain.Bet, error) {
	key := fraudWindowKey(userID)
	now := time.Now()
	min := fmt.Sprintf("%d", now.Add(-window).UnixMilli())
	max := fmt.Sprintf("%d", now.UnixMilli())

	vals, err := c.client.ZRangeByScore(ctx, key, &redis.ZRangeBy{
		Min: min, Max: max,
	}).Result()
	if err != nil {
		return nil, err
	}

	bets := make([]domain.Bet, 0, len(vals))
	for _, v := range vals {
		var bet domain.Bet
		if err := json.Unmarshal([]byte(v), &bet); err != nil {
			continue
		}
		bets = append(bets, bet)
	}
	return bets, nil
}

func fraudWindowKey(userID string) string { return "user_bets:" + userID }
