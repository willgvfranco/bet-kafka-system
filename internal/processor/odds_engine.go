package processor

import (
	"context"
	"encoding/json"
	"log/slog"
	"math"
	"math/rand"

	"github.com/neus/bet-kafka-system/internal/domain"
	kafkago "github.com/segmentio/kafka-go"
)

type OddsEngine struct {
	odds   domain.OddsCache
	evts   domain.EventRepository
	pub    domain.EventPublisher
	logger *slog.Logger
}

func NewOddsEngine(
	odds domain.OddsCache,
	evts domain.EventRepository,
	pub domain.EventPublisher,
) *OddsEngine {
	return &OddsEngine{
		odds:   odds,
		evts:   evts,
		pub:    pub,
		logger: slog.With("service", "odds-engine"),
	}
}

func (e *OddsEngine) HandleGameEvent(ctx context.Context, msg kafkago.Message) error {
	var ge domain.GameEvent
	if err := json.Unmarshal(msg.Value, &ge); err != nil {
		e.logger.Error("game_event_parse_failed", "error", err)
		return nil
	}
	log := e.logger.With("correlation_id", ge.CorrelationID, "event_id", ge.EventID, "type", ge.Type)

	event, err := e.evts.FindByID(ctx, ge.EventID)
	if err != nil || event == nil {
		log.Error("event_not_found", "error", err)
		return nil
	}

	for _, market := range event.Markets {
		if market.Status != domain.MarketStatusOpen {
			continue
		}
		for _, outcome := range market.Outcomes {
			oldOdds, _ := e.odds.GetOdds(ctx, outcome.OutcomeID)
			if oldOdds == 0 {
				oldOdds = outcome.InitialOdds
			}

			newOdds := e.recalculate(oldOdds, ge, market.Type, outcome.Name)
			newOdds = math.Round(newOdds*100) / 100
			if newOdds < 1.01 {
				newOdds = 1.01
			}

			if err := e.pub.PublishOddsUpdated(ctx, ge.EventID, market.MarketID, outcome.OutcomeID, oldOdds, newOdds, string(ge.Type), ge.CorrelationID); err != nil {
				log.Error("odds_publish_failed", "error", err, "outcome_id", outcome.OutcomeID)
			}
		}
	}

	log.Info("odds_recalculated", "game_event_type", ge.Type, "minute", ge.Minute)
	return nil
}

func (e *OddsEngine) recalculate(currentOdds float64, ge domain.GameEvent, marketType domain.MarketType, outcomeName string) float64 {
	factor := 1.0
	jitter := (rand.Float64() - 0.5) * 0.04 // +/- 2% noise

	switch ge.Type {
	case domain.GameEventGoal:
		if marketType == domain.MarketTypeMatchWinner {
			factor = e.goalImpact(ge.Team, outcomeName)
		} else {
			// Over/Under: goals make "over" more likely
			if contains(outcomeName, "Over") {
				factor = 0.80
			} else {
				factor = 1.25
			}
		}
	case domain.GameEventRedCard:
		if marketType == domain.MarketTypeMatchWinner {
			factor = e.redCardImpact(ge.Team, outcomeName)
		}
	case domain.GameEventYellowCard:
		factor = 1.0 + jitter
	case domain.GameEventHalftime, domain.GameEventSecondHalf:
		factor = 1.0 + jitter*0.5
	case domain.GameEventFullTime:
		return currentOdds // no change at full time
	}

	return currentOdds * (factor + jitter)
}

func (e *OddsEngine) goalImpact(team string, outcomeName string) float64 {
	isHome := team == "home"
	switch {
	case isHome && contains(outcomeName, "Home", "Win") && !contains(outcomeName, "Away"):
		return 0.70 // team scored, their win odds decrease (more likely)
	case !isHome && contains(outcomeName, "Away", "Win") && !contains(outcomeName, "Home"):
		return 0.70
	case contains(outcomeName, "Draw"):
		return 1.30 // draw less likely after a goal
	default:
		return 1.35 // opposing team win less likely
	}
}

func (e *OddsEngine) redCardImpact(team string, outcomeName string) float64 {
	isHome := team == "home"
	switch {
	case isHome && contains(outcomeName, "Home", "Win") && !contains(outcomeName, "Away"):
		return 1.20 // red card on home team, their win odds increase (less likely)
	case !isHome && contains(outcomeName, "Away", "Win") && !contains(outcomeName, "Home"):
		return 1.20
	default:
		return 0.90
	}
}

func contains(s string, substrs ...string) bool {
	for _, sub := range substrs {
		found := false
		for i := 0; i <= len(s)-len(sub); i++ {
			if s[i:i+len(sub)] == sub {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
