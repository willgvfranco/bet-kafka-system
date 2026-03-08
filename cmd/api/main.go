package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/neus/bet-kafka-system/internal/cache"
	"github.com/neus/bet-kafka-system/internal/config"
	"github.com/neus/bet-kafka-system/internal/handler"
	"github.com/neus/bet-kafka-system/internal/kafka"
	"github.com/neus/bet-kafka-system/internal/observability"
	"github.com/neus/bet-kafka-system/internal/repository"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	cfg := config.Load()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// MongoDB
	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		slog.Error("mongodb_connect_failed", "error", err)
		os.Exit(1)
	}
	db := mongoClient.Database("betting")

	// Redis
	redisClient := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
	if err := redisClient.Ping(ctx).Err(); err != nil {
		slog.Error("redis_connect_failed", "error", err)
		os.Exit(1)
	}

	// Dependencies
	producer := kafka.NewProducer(cfg.KafkaBrokers)
	betRepo := repository.NewMongoBetRepo(db)
	eventRepo := repository.NewMongoEventRepo(db)
	userRepo := repository.NewMongoUserRepo(db)
	txRepo := repository.NewMongoTransactionRepo(db)
	oddsCache := cache.NewRedisOddsCache(redisClient)

	// Handlers
	betHandler := handler.NewBetHandler(betRepo, producer)
	eventHandler := handler.NewEventHandler(eventRepo, oddsCache)
	walletHandler := handler.NewWalletHandler(userRepo, txRepo)
	adminHandler := handler.NewAdminHandler(eventRepo, producer)

	// Routes
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/bets", betHandler.PlaceBet)
	mux.HandleFunc("GET /api/v1/bets", betHandler.GetBets)
	mux.HandleFunc("GET /api/v1/bets/{id}", betHandler.GetBetByID)
	mux.HandleFunc("GET /api/v1/events", eventHandler.ListEvents)
	mux.HandleFunc("GET /api/v1/events/{id}", eventHandler.GetEvent)
	mux.HandleFunc("GET /api/v1/events/{id}/odds", eventHandler.GetEventOdds)
	mux.HandleFunc("GET /api/v1/wallet", walletHandler.GetWallet)
	mux.HandleFunc("POST /api/v1/admin/settle", adminHandler.SettleMarket)
	mux.HandleFunc("POST /api/v1/admin/event-status", adminHandler.UpdateEventStatus)
	mux.Handle("/metrics", observability.MetricsHandler())

	server := &http.Server{Addr: ":" + cfg.HTTPPort, Handler: observability.MetricsMiddleware(mux)}

	go func() {
		slog.Info("api_started", "port", cfg.HTTPPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("api_listen_failed", "error", err)
			os.Exit(1)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	slog.Info("api_shutting_down")
	shutCtx, shutCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutCancel()
	server.Shutdown(shutCtx)
	producer.Close()
	mongoClient.Disconnect(shutCtx)
	redisClient.Close()
}
