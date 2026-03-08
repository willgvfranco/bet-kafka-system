package processor

import (
	"context"
	"encoding/json"
	"log/slog"
	"math"
	"time"

	"github.com/neus/bet-kafka-system/internal/domain"
	"github.com/neus/bet-kafka-system/internal/observability"
	kafkago "github.com/segmentio/kafka-go"
)

const velocityThreshold = 20.0

type OddsProcessor struct {
	odds      domain.OddsCache
	suspended domain.MarketSuspensionCache
	pub       domain.EventPublisher
	logger    *slog.Logger
}

func NewOddsProcessor(
	odds domain.OddsCache,
	suspended domain.MarketSuspensionCache,
	pub domain.EventPublisher,
) *OddsProcessor {
	return &OddsProcessor{
		odds:      odds,
		suspended: suspended,
		pub:       pub,
		logger:    slog.With("service", "odds-consumer"),
	}
}

func (p *OddsProcessor) HandleOddsUpdated(ctx context.Context, msg kafkago.Message) error {
	var m domain.OddsUpdatedMessage
	if err := json.Unmarshal(msg.Value, &m); err != nil {
		p.logger.Error("odds_parse_failed", "error", err)
		return nil
	}
	log := p.logger.With("correlation_id", m.CorrelationID, "outcome_id", m.OutcomeID)

	// 1. Update Redis cache
	if err := p.odds.SetOdds(ctx, m.OutcomeID, m.NewOdds); err != nil {
		return err
	}
	observability.OddsUpdatesTotal.Inc()
	observability.OddsChangePercent.Observe(math.Abs(m.ChangePercent))

	// 2. Track velocity
	if err := p.odds.PushOddsHistory(ctx, m.OutcomeID, m.ChangePercent); err != nil {
		log.Error("odds_history_push_failed", "error", err)
	}

	// 3. Check velocity for market suspension
	velocity, err := p.odds.GetOddsVelocity(ctx, m.OutcomeID)
	if err != nil {
		log.Error("odds_velocity_check_failed", "error", err)
	} else if math.Abs(velocity) > velocityThreshold {
		if err := p.suspended.Suspend(ctx, m.MarketID, 30*time.Second); err != nil {
			log.Error("market_suspend_failed", "error", err)
		}
		_ = p.pub.PublishMarketSuspended(ctx, m.EventID, m.MarketID, velocity, m.CorrelationID)
		observability.MarketSuspensionsTotal.Inc()
		log.Warn("market_suspended", "market_id", m.MarketID, "velocity", velocity)
	}

	log.Info("odds_cached", "outcome_id", m.OutcomeID, "new_odds", m.NewOdds,
		"change_percent", m.ChangePercent, "partition", msg.Partition, "offset", msg.Offset)
	return nil
}
