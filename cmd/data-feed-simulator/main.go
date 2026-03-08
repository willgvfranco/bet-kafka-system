package main

import (
	"context"
	"log/slog"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/neus/bet-kafka-system/internal/config"
	"github.com/neus/bet-kafka-system/internal/domain"
	"github.com/neus/bet-kafka-system/internal/kafka"
	"github.com/neus/bet-kafka-system/internal/observability"
	"github.com/neus/bet-kafka-system/internal/repository"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type gameState struct {
	minute int
	score  domain.Score
}

var players = map[string][]string{
	"home": {"Player A", "Player B", "Player C", "Player D", "Player E"},
	"away": {"Player F", "Player G", "Player H", "Player I", "Player J"},
}

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

	producer := kafka.NewProducer(cfg.KafkaBrokers)
	eventRepo := repository.NewMongoEventRepo(db)

	runCtx, runCancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer runCancel()

	slog.Info("data_feed_simulator_started")

	states := make(map[string]*gameState)

	for {
		select {
		case <-runCtx.Done():
			slog.Info("data_feed_simulator_stopped")
			producer.Close()
			mongoClient.Disconnect(context.Background())
			return
		default:
		}

		// Random interval 5-10 seconds
		delay := time.Duration(5+rand.Intn(6)) * time.Second
		time.Sleep(delay)

		events, err := eventRepo.FindByStatus(runCtx, domain.EventStatusLive)
		if err != nil || len(events) == 0 {
			continue
		}

		event := events[rand.Intn(len(events))]
		state, ok := states[event.ID]
		if !ok {
			state = &gameState{minute: 1}
			states[event.ID] = state
		}

		// Advance game minute
		state.minute += rand.Intn(5) + 1
		if state.minute > 90 {
			state.minute = 90
		}

		ge := generateGameEvent(event.ID, state)
		if err := producer.PublishGameEvent(runCtx, &ge); err != nil {
			slog.Error("game_event_publish_failed", "error", err)
			continue
		}
		observability.GameEventsTotal.WithLabelValues(string(ge.Type)).Inc()

		slog.Info("game_event_published",
			"event_id", event.ID,
			"event_name", event.Name,
			"type", ge.Type,
			"minute", ge.Minute,
			"score", ge.Score,
		)
	}
}

func generateGameEvent(eventID string, state *gameState) domain.GameEvent {
	team := "home"
	if rand.Float64() > 0.5 {
		team = "away"
	}

	var eventType domain.GameEventType
	r := rand.Float64()
	switch {
	case state.minute == 45:
		eventType = domain.GameEventHalftime
	case state.minute == 46:
		eventType = domain.GameEventSecondHalf
	case state.minute >= 90:
		eventType = domain.GameEventFullTime
	case r < 0.15:
		eventType = domain.GameEventGoal
		if team == "home" {
			state.score.Home++
		} else {
			state.score.Away++
		}
	case r < 0.30:
		eventType = domain.GameEventYellowCard
	case r < 0.35:
		eventType = domain.GameEventRedCard
	default:
		eventType = domain.GameEventYellowCard // default to yellow card for more events
	}

	playerList := players[team]
	player := playerList[rand.Intn(len(playerList))]

	return domain.GameEvent{
		EventID:       eventID,
		Type:          eventType,
		Team:          team,
		Player:        player,
		Minute:        state.minute,
		Score:         state.score,
		CorrelationID: uuid.New().String(),
		Timestamp:     time.Now(),
	}
}
