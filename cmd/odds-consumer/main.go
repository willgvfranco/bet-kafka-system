package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/neus/bet-kafka-system/internal/cache"
	"github.com/neus/bet-kafka-system/internal/config"
	"github.com/neus/bet-kafka-system/internal/kafka"
	"github.com/neus/bet-kafka-system/internal/observability"
	"github.com/neus/bet-kafka-system/internal/processor"
	"github.com/redis/go-redis/v9"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	cfg := config.Load()

	redisClient := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
	producer := kafka.NewProducer(cfg.KafkaBrokers)
	oddsCache := cache.NewRedisOddsCache(redisClient)
	suspensionCache := cache.NewRedisMarketSuspensionCache(redisClient)

	observability.StartMetricsServer(cfg.MetricsPort)
	op := processor.NewOddsProcessor(oddsCache, suspensionCache, producer)

	runCtx, runCancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer runCancel()

	consumer := kafka.NewConsumer(cfg.KafkaBrokers, kafka.TopicOddsUpdated, "odds-cache-updater", op.HandleOddsUpdated)

	slog.Info("odds_consumer_started")
	if err := consumer.Run(runCtx); err != nil {
		slog.Error("odds_consumer_failed", "error", err)
	}

	producer.Close()
	redisClient.Close()
}
