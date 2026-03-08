package kafka

import (
	"context"
	"encoding/json"
	"time"

	"github.com/neus/bet-kafka-system/internal/domain"
	"github.com/neus/bet-kafka-system/internal/observability"
	kafkago "github.com/segmentio/kafka-go"
)

type Producer struct {
	writer *kafkago.Writer
}

func NewProducer(brokers string) *Producer {
	return &Producer{
		writer: &kafkago.Writer{
			Addr:         kafkago.TCP(brokers),
			Balancer:     &kafkago.Hash{},
			BatchTimeout: 10 * time.Millisecond,
			RequiredAcks: kafkago.RequireAll,
		},
	}
}

func (p *Producer) publish(ctx context.Context, topic string, key []byte, value any, correlationID string) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	err = p.writer.WriteMessages(ctx, kafkago.Message{
		Topic: topic,
		Key:   key,
		Value: data,
		Headers: []kafkago.Header{
			{Key: "correlation_id", Value: []byte(correlationID)},
		},
	})
	if err == nil {
		observability.KafkaMessagesProduced.WithLabelValues(topic).Inc()
	}
	return err
}

func (p *Producer) PublishBetPlaced(ctx context.Context, bet *domain.Bet) error {
	msg := domain.BetPlacedMessage{
		BetID:           bet.ID,
		UserID:          bet.UserID,
		EventID:         bet.EventID,
		MarketID:        bet.MarketID,
		OutcomeID:       bet.OutcomeID,
		Stake:           bet.Stake,
		OddsAtPlacement: bet.OddsAtPlacement,
		PotentialPayout: bet.PotentialPayout,
		IdempotencyKey:  bet.IdempotencyKey,
		CorrelationID:   bet.CorrelationID,
		Timestamp:       bet.PlacedAt,
	}
	return p.publish(ctx, TopicBetsPlaced, []byte(bet.UserID), msg, bet.CorrelationID)
}

func (p *Producer) PublishBetSettled(ctx context.Context, eventID, marketID, winningOutcomeID, correlationID string) error {
	msg := domain.BetSettledMessage{
		EventID:          eventID,
		MarketID:         marketID,
		WinningOutcomeID: winningOutcomeID,
		CorrelationID:    correlationID,
		Timestamp:        time.Now(),
	}
	return p.publish(ctx, TopicBetsSettled, []byte(marketID), msg, correlationID)
}

func (p *Producer) PublishOddsUpdated(ctx context.Context, eventID, marketID, outcomeID string, oldOdds, newOdds float64, trigger string, correlationID string) error {
	changePercent := 0.0
	if oldOdds > 0 {
		changePercent = ((newOdds - oldOdds) / oldOdds) * 100
	}
	msg := domain.OddsUpdatedMessage{
		EventID:       eventID,
		MarketID:      marketID,
		OutcomeID:     outcomeID,
		OldOdds:       oldOdds,
		NewOdds:       newOdds,
		ChangePercent: changePercent,
		Trigger:       trigger,
		CorrelationID: correlationID,
		Timestamp:     time.Now(),
	}
	return p.publish(ctx, TopicOddsUpdated, []byte(outcomeID), msg, correlationID)
}

func (p *Producer) PublishGameEvent(ctx context.Context, event *domain.GameEvent) error {
	return p.publish(ctx, TopicGameEvents, []byte(event.EventID), event, event.CorrelationID)
}

func (p *Producer) PublishMarketSuspended(ctx context.Context, eventID, marketID string, avgChange float64, correlationID string) error {
	msg := domain.MarketSuspendedMessage{
		EventID:          eventID,
		MarketID:         marketID,
		Reason:           "ODDS_VELOCITY",
		AvgChangePercent: avgChange,
		DurationSeconds:  30,
		CorrelationID:    correlationID,
		Timestamp:        time.Now(),
	}
	return p.publish(ctx, TopicMarketsSuspended, []byte(marketID), msg, correlationID)
}

func (p *Producer) PublishFraudAlert(ctx context.Context, alert *domain.FraudAlert) error {
	return p.publish(ctx, TopicFraudAlerts, []byte(alert.UserID), alert, alert.CorrelationID)
}

func (p *Producer) PublishWalletTransaction(ctx context.Context, tx *domain.Transaction) error {
	return p.publish(ctx, TopicWalletTransactions, []byte(tx.UserID), tx, tx.CorrelationID)
}

func (p *Producer) Close() error {
	return p.writer.Close()
}
