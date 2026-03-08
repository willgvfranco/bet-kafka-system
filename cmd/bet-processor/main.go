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
	"github.com/neus/bet-kafka-system/internal/observability"
	"github.com/neus/bet-kafka-system/internal/kafka"
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
	betRepo := repository.NewMongoBetRepo(db)
	userRepo := repository.NewMongoUserRepo(db)
	txRepo := repository.NewMongoTransactionRepo(db)
	oddsCache := cache.NewRedisOddsCache(redisClient)
	idempotencyCache := cache.NewRedisIdempotencyCache(redisClient)
	suspensionCache := cache.NewRedisMarketSuspensionCache(redisClient)

	observability.StartMetricsServer(cfg.MetricsPort)
	bp := processor.NewBetProcessor(betRepo, userRepo, txRepo, oddsCache, idempotencyCache, suspensionCache, producer)

	runCtx, runCancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer runCancel()

	betsConsumer := kafka.NewConsumer(cfg.KafkaBrokers, kafka.TopicBetsPlaced, "bet-processor", bp.HandleBetPlaced)
	settlementConsumer := kafka.NewConsumer(cfg.KafkaBrokers, kafka.TopicBetsSettled, "bet-processor", bp.HandleBetSettled)

	go func() {
		if err := betsConsumer.Run(runCtx); err != nil {
			slog.Error("bets_consumer_failed", "error", err)
		}
	}()

	go func() {
		if err := settlementConsumer.Run(runCtx); err != nil {
			slog.Error("settlement_consumer_failed", "error", err)
		}
	}()

	slog.Info("bet_processor_started")
	<-runCtx.Done()
	slog.Info("bet_processor_stopped")

	producer.Close()
	mongoClient.Disconnect(context.Background())
	redisClient.Close()
}
