package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/neus/bet-kafka-system/internal/cache"
	"github.com/neus/bet-kafka-system/internal/config"
	"github.com/neus/bet-kafka-system/internal/kafka"
	"github.com/neus/bet-kafka-system/internal/observability"
	"github.com/neus/bet-kafka-system/internal/processor"
	"github.com/neus/bet-kafka-system/internal/repository"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	cfg := config.Load()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.MongoURI))
	cancel()
	if err != nil {
		slog.Error("mongodb_connect_failed", "error", err)
		os.Exit(1)
	}
	db := mongoClient.Database("betting")

	redisClient := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
	producer := kafka.NewProducer(cfg.KafkaBrokers)
	eventRepo := repository.NewMongoEventRepo(db)
	oddsCache := cache.NewRedisOddsCache(redisClient)

	observability.StartMetricsServer(cfg.MetricsPort)
	engine := processor.NewOddsEngine(oddsCache, eventRepo, producer)

	runCtx, runCancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer runCancel()

	consumer := kafka.NewConsumer(cfg.KafkaBrokers, kafka.TopicGameEvents, "odds-engine", engine.HandleGameEvent)

	slog.Info("odds_engine_started")
	if err := consumer.Run(runCtx); err != nil {
		slog.Error("odds_engine_consumer_failed", "error", err)
	}

	producer.Close()
	mongoClient.Disconnect(context.Background())
	redisClient.Close()
}
