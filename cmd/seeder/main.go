package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/neus/bet-kafka-system/internal/cache"
	"github.com/neus/bet-kafka-system/internal/config"
	"github.com/neus/bet-kafka-system/internal/domain"
	"github.com/neus/bet-kafka-system/internal/repository"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	cfg := config.Load()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		slog.Error("mongodb_connect_failed", "error", err)
		os.Exit(1)
	}
	defer mongoClient.Disconnect(ctx)
	db := mongoClient.Database("betting")

	redisClient := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
	defer redisClient.Close()

	// Drop existing collections
	db.Collection("users").Drop(ctx)
	db.Collection("events").Drop(ctx)
	db.Collection("bets").Drop(ctx)
	db.Collection("transactions").Drop(ctx)

	userRepo := repository.NewMongoUserRepo(db)
	eventRepo := repository.NewMongoEventRepo(db)
	oddsCache := cache.NewRedisOddsCache(redisClient)

	now := time.Now()

	// --- Users ---
	users := []*domain.User{
		{ID: uuid.New().String(), Username: "william", Balance: 1000.00, Status: domain.UserStatusActive, CreatedAt: now, UpdatedAt: now},
		{ID: uuid.New().String(), Username: "testuser", Balance: 500.00, Status: domain.UserStatusActive, CreatedAt: now, UpdatedAt: now},
	}
	for _, u := range users {
		if err := userRepo.Save(ctx, u); err != nil {
			slog.Error("user_seed_failed", "error", err, "username", u.Username)
			os.Exit(1)
		}
		slog.Info("user_seeded", "id", u.ID, "username", u.Username, "balance", u.Balance)
	}

	// --- Events ---
	events := []*domain.Event{
		{
			ID:        uuid.New().String(),
			Name:      "Flamengo vs Palmeiras",
			Sport:     "football",
			Status:    domain.EventStatusLive,
			StartTime: now.Add(-30 * time.Minute),
			Markets: []domain.Market{
				{
					MarketID: uuid.New().String(),
					Type:     domain.MarketTypeMatchWinner,
					Status:   domain.MarketStatusOpen,
					Outcomes: []domain.Outcome{
						{OutcomeID: uuid.New().String(), Name: "Flamengo Win", InitialOdds: 1.85, Result: domain.OutcomeResultPending},
						{OutcomeID: uuid.New().String(), Name: "Draw", InitialOdds: 3.40, Result: domain.OutcomeResultPending},
						{OutcomeID: uuid.New().String(), Name: "Palmeiras Win", InitialOdds: 2.10, Result: domain.OutcomeResultPending},
					},
				},
				{
					MarketID: uuid.New().String(),
					Type:     domain.MarketTypeOverUnder,
					Status:   domain.MarketStatusOpen,
					Outcomes: []domain.Outcome{
						{OutcomeID: uuid.New().String(), Name: "Over 2.5 Goals", InitialOdds: 1.90, Result: domain.OutcomeResultPending},
						{OutcomeID: uuid.New().String(), Name: "Under 2.5 Goals", InitialOdds: 1.95, Result: domain.OutcomeResultPending},
					},
				},
			},
		},
		{
			ID:        uuid.New().String(),
			Name:      "Barcelona vs Real Madrid",
			Sport:     "football",
			Status:    domain.EventStatusLive,
			StartTime: now.Add(-15 * time.Minute),
			Markets: []domain.Market{
				{
					MarketID: uuid.New().String(),
					Type:     domain.MarketTypeMatchWinner,
					Status:   domain.MarketStatusOpen,
					Outcomes: []domain.Outcome{
						{OutcomeID: uuid.New().String(), Name: "Barcelona Win", InitialOdds: 2.20, Result: domain.OutcomeResultPending},
						{OutcomeID: uuid.New().String(), Name: "Draw", InitialOdds: 3.25, Result: domain.OutcomeResultPending},
						{OutcomeID: uuid.New().String(), Name: "Real Madrid Win", InitialOdds: 2.80, Result: domain.OutcomeResultPending},
					},
				},
				{
					MarketID: uuid.New().String(),
					Type:     domain.MarketTypeOverUnder,
					Status:   domain.MarketStatusOpen,
					Outcomes: []domain.Outcome{
						{OutcomeID: uuid.New().String(), Name: "Over 2.5 Goals", InitialOdds: 1.75, Result: domain.OutcomeResultPending},
						{OutcomeID: uuid.New().String(), Name: "Under 2.5 Goals", InitialOdds: 2.10, Result: domain.OutcomeResultPending},
					},
				},
			},
		},
		{
			ID:        uuid.New().String(),
			Name:      "Lakers vs Celtics",
			Sport:     "basketball",
			Status:    domain.EventStatusLive,
			StartTime: now.Add(-10 * time.Minute),
			Markets: []domain.Market{
				{
					MarketID: uuid.New().String(),
					Type:     domain.MarketTypeMatchWinner,
					Status:   domain.MarketStatusOpen,
					Outcomes: []domain.Outcome{
						{OutcomeID: uuid.New().String(), Name: "Lakers Win", InitialOdds: 1.95, Result: domain.OutcomeResultPending},
						{OutcomeID: uuid.New().String(), Name: "Celtics Win", InitialOdds: 1.85, Result: domain.OutcomeResultPending},
					},
				},
				{
					MarketID: uuid.New().String(),
					Type:     domain.MarketTypeOverUnder,
					Status:   domain.MarketStatusOpen,
					Outcomes: []domain.Outcome{
						{OutcomeID: uuid.New().String(), Name: "Over 210.5 Points", InitialOdds: 1.90, Result: domain.OutcomeResultPending},
						{OutcomeID: uuid.New().String(), Name: "Under 210.5 Points", InitialOdds: 1.90, Result: domain.OutcomeResultPending},
					},
				},
			},
		},
	}

	for _, e := range events {
		if err := eventRepo.Save(ctx, e); err != nil {
			slog.Error("event_seed_failed", "error", err, "name", e.Name)
			os.Exit(1)
		}

		// Seed initial odds into Redis
		for _, m := range e.Markets {
			for _, o := range m.Outcomes {
				if err := oddsCache.SetOdds(ctx, o.OutcomeID, o.InitialOdds); err != nil {
					slog.Error("odds_cache_seed_failed", "error", err, "outcome_id", o.OutcomeID)
				}
				// Register outcome in event set for lookups
				redisClient.SAdd(ctx, "event_odds:"+e.ID, o.OutcomeID)
			}
		}

		slog.Info("event_seeded", "id", e.ID, "name", e.Name, "status", e.Status, "markets", len(e.Markets))
	}

	slog.Info("seed_complete", "users", len(users), "events", len(events))
}
