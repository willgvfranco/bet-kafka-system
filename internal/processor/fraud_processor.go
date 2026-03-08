package processor

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/neus/bet-kafka-system/internal/domain"
	"github.com/neus/bet-kafka-system/internal/observability"
	kafkago "github.com/segmentio/kafka-go"
)

// --- Fraud rule interface (Open/Closed principle) ---

type FraudRule interface {
	Check(ctx context.Context, bet *domain.Bet, recentBets []domain.Bet) *domain.FraudAlert
}

// --- Concrete rules ---

type RateLimitRule struct{}

func (r *RateLimitRule) Check(_ context.Context, bet *domain.Bet, recentBets []domain.Bet) *domain.FraudAlert {
	cutoff := time.Now().Add(-60 * time.Second)
	count := 0
	for _, b := range recentBets {
		if b.PlacedAt.After(cutoff) {
			count++
		}
	}
	if count > 5 {
		return &domain.FraudAlert{
			AlertID:       uuid.New().String(),
			UserID:        bet.UserID,
			AlertType:     domain.FraudAlertRateLimit,
			Details:       fmt.Sprintf("User placed %d bets in last 60 seconds", count),
			CorrelationID: bet.CorrelationID,
			Severity:      domain.FraudSeverityMedium,
			Timestamp:     time.Now(),
		}
	}
	return nil
}

type OppositeBetsRule struct{}

func (r *OppositeBetsRule) Check(_ context.Context, bet *domain.Bet, recentBets []domain.Bet) *domain.FraudAlert {
	for _, b := range recentBets {
		if b.EventID == bet.EventID && b.OutcomeID != bet.OutcomeID {
			return &domain.FraudAlert{
				AlertID:       uuid.New().String(),
				UserID:        bet.UserID,
				AlertType:     domain.FraudAlertOppositeBets,
				Details:       fmt.Sprintf("User placed bets on different outcomes of event %s within 5 minutes", bet.EventID),
				CorrelationID: bet.CorrelationID,
				Severity:      domain.FraudSeverityHigh,
				Timestamp:     time.Now(),
			}
		}
	}
	return nil
}

type SuspiciousAmountRule struct {
	threshold float64
}

func (r *SuspiciousAmountRule) Check(_ context.Context, bet *domain.Bet, recentBets []domain.Bet) *domain.FraudAlert {
	if len(recentBets) < 3 {
		return nil
	}
	var totalStake float64
	for _, b := range recentBets {
		totalStake += b.Stake
	}
	avgStake := totalStake / float64(len(recentBets))
	if bet.Stake > avgStake*r.threshold {
		return &domain.FraudAlert{
			AlertID:       uuid.New().String(),
			UserID:        bet.UserID,
			AlertType:     domain.FraudAlertSuspiciousAmount,
			Details:       fmt.Sprintf("Bet stake %.2f is %.1fx the user average of %.2f", bet.Stake, bet.Stake/avgStake, avgStake),
			CorrelationID: bet.CorrelationID,
			Severity:      domain.FraudSeverityHigh,
			Timestamp:     time.Now(),
		}
	}
	return nil
}

// --- Fraud Processor ---

type FraudProcessor struct {
	window domain.FraudWindow
	pub    domain.EventPublisher
	rules  []FraudRule
	logger *slog.Logger
}

func NewFraudProcessor(window domain.FraudWindow, pub domain.EventPublisher) *FraudProcessor {
	return &FraudProcessor{
		window: window,
		pub:    pub,
		rules: []FraudRule{
			&RateLimitRule{},
			&OppositeBetsRule{},
			&SuspiciousAmountRule{threshold: 10},
		},
		logger: slog.With("service", "fraud-detector"),
	}
}

func (p *FraudProcessor) HandleBetPlaced(ctx context.Context, msg kafkago.Message) error {
	var m domain.BetPlacedMessage
	if err := json.Unmarshal(msg.Value, &m); err != nil {
		p.logger.Error("message_parse_failed", "error", err)
		return nil
	}
	log := p.logger.With("correlation_id", m.CorrelationID, "user_id", m.UserID, "bet_id", m.BetID)

	bet := &domain.Bet{
		ID:              m.BetID,
		UserID:          m.UserID,
		EventID:         m.EventID,
		MarketID:        m.MarketID,
		OutcomeID:       m.OutcomeID,
		Stake:           m.Stake,
		OddsAtPlacement: m.OddsAtPlacement,
		PotentialPayout: m.PotentialPayout,
		CorrelationID:   m.CorrelationID,
		PlacedAt:        m.Timestamp,
	}

	recentBets, err := p.window.GetRecentBets(ctx, m.UserID, 5*time.Minute)
	if err != nil {
		return err
	}

	for _, rule := range p.rules {
		if alert := rule.Check(ctx, bet, recentBets); alert != nil {
			if err := p.pub.PublishFraudAlert(ctx, alert); err != nil {
				log.Error("fraud_alert_publish_failed", "error", err, "alert_type", alert.AlertType)
			} else {
				observability.FraudAlertsTotal.WithLabelValues(string(alert.AlertType)).Inc()
				log.Warn("fraud_alert_triggered", "alert_type", alert.AlertType, "severity", alert.Severity, "details", alert.Details)
			}
		}
	}

	if err := p.window.AddBet(ctx, m.UserID, bet); err != nil {
		log.Error("fraud_window_add_failed", "error", err)
	}

	log.Info("fraud_check_complete", "recent_bets", len(recentBets),
		"partition", msg.Partition, "offset", msg.Offset)
	return nil
}
