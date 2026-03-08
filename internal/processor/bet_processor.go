package processor

import (
	"context"
	"encoding/json"
	"log/slog"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/neus/bet-kafka-system/internal/domain"
	"github.com/neus/bet-kafka-system/internal/observability"
	kafkago "github.com/segmentio/kafka-go"
)

const oddsTolerance = 0.05 // 5% tolerance for odds drift

type BetProcessor struct {
	bets       domain.BetRepository
	users      domain.UserRepository
	txs        domain.TransactionRepository
	odds       domain.OddsCache
	idempotent domain.IdempotencyCache
	suspended  domain.MarketSuspensionCache
	events     domain.EventPublisher
	logger     *slog.Logger
}

func NewBetProcessor(
	bets domain.BetRepository,
	users domain.UserRepository,
	txs domain.TransactionRepository,
	odds domain.OddsCache,
	idempotent domain.IdempotencyCache,
	suspended domain.MarketSuspensionCache,
	events domain.EventPublisher,
) *BetProcessor {
	return &BetProcessor{
		bets:       bets,
		users:      users,
		txs:        txs,
		odds:       odds,
		idempotent: idempotent,
		suspended:  suspended,
		events:     events,
		logger:     slog.With("service", "bet-processor"),
	}
}

func (p *BetProcessor) HandleBetPlaced(ctx context.Context, msg kafkago.Message) error {
	var m domain.BetPlacedMessage
	if err := json.Unmarshal(msg.Value, &m); err != nil {
		p.logger.Error("message_parse_failed", "error", err)
		return nil // don't retry malformed messages
	}
	log := p.logger.With("correlation_id", m.CorrelationID, "bet_id", m.BetID, "user_id", m.UserID)

	// 1. Idempotency check
	alreadyProcessed, err := p.idempotent.Check(ctx, m.IdempotencyKey)
	if err != nil {
		return err
	}
	if alreadyProcessed {
		log.Info("bet_duplicate_skipped")
		observability.BetsRejectedTotal.WithLabelValues("duplicate").Inc()
		return nil
	}

	// 2. Market suspension check
	suspended, err := p.suspended.IsSuspended(ctx, m.MarketID)
	if err != nil {
		return err
	}
	if suspended {
		log.Warn("bet_rejected_market_suspended", "market_id", m.MarketID)
		observability.BetsRejectedTotal.WithLabelValues("market_suspended").Inc()
		return nil
	}

	// 3. Odds validation
	currentOdds, err := p.odds.GetOdds(ctx, m.OutcomeID)
	if err != nil {
		return err
	}
	if currentOdds > 0 {
		drift := math.Abs(currentOdds-m.OddsAtPlacement) / m.OddsAtPlacement
		if drift > oddsTolerance {
			log.Warn("bet_rejected_odds_changed",
				"requested_odds", m.OddsAtPlacement,
				"current_odds", currentOdds,
				"drift", drift,
			)
			observability.BetsRejectedTotal.WithLabelValues("odds_changed").Inc()
			return nil
		}
	}

	// 4. Debit user balance (atomic $inc)
	user, err := p.users.UpdateBalance(ctx, m.UserID, -m.Stake)
	if err != nil {
		log.Warn("bet_rejected_insufficient_balance", "stake", m.Stake, "error", err)
		observability.BetsRejectedTotal.WithLabelValues("insufficient_balance").Inc()
		return nil
	}

	// 5. Persist bet
	bet := &domain.Bet{
		ID:              m.BetID,
		UserID:          m.UserID,
		EventID:         m.EventID,
		MarketID:        m.MarketID,
		OutcomeID:       m.OutcomeID,
		Stake:           m.Stake,
		OddsAtPlacement: m.OddsAtPlacement,
		PotentialPayout: m.PotentialPayout,
		Status:          domain.BetStatusPending,
		IdempotencyKey:  m.IdempotencyKey,
		CorrelationID:   m.CorrelationID,
		PlacedAt:        m.Timestamp,
	}
	if err := p.bets.Save(ctx, bet); err != nil {
		return err
	}

	// 6. Publish wallet transaction (audit trail)
	tx := &domain.Transaction{
		ID:            uuid.New().String(),
		UserID:        m.UserID,
		Type:          domain.TransactionTypeBetPlaced,
		Amount:        -m.Stake,
		BalanceAfter:  user.Balance,
		ReferenceID:   m.BetID,
		CorrelationID: m.CorrelationID,
		CreatedAt:     time.Now(),
	}
	if err := p.txs.Save(ctx, tx); err != nil {
		log.Error("transaction_save_failed", "error", err)
	}
	_ = p.events.PublishWalletTransaction(ctx, tx)

	observability.BetsPlacedTotal.Inc()
	log.Info("bet_processed", "stake", m.Stake, "odds", m.OddsAtPlacement,
		"partition", msg.Partition, "offset", msg.Offset)
	return nil
}

func (p *BetProcessor) HandleBetSettled(ctx context.Context, msg kafkago.Message) error {
	var m domain.BetSettledMessage
	if err := json.Unmarshal(msg.Value, &m); err != nil {
		p.logger.Error("settlement_parse_failed", "error", err)
		return nil
	}
	log := p.logger.With("correlation_id", m.CorrelationID, "market_id", m.MarketID)

	bets, err := p.bets.FindByMarket(ctx, m.MarketID)
	if err != nil {
		return err
	}

	now := time.Now()
	for _, bet := range bets {
		var status domain.BetStatus
		var txType domain.TransactionType
		var amount float64

		if bet.OutcomeID == m.WinningOutcomeID {
			status = domain.BetStatusWon
			txType = domain.TransactionTypeBetWon
			amount = bet.PotentialPayout
		} else {
			status = domain.BetStatusLost
			txType = domain.TransactionTypeBetLost
			amount = 0
		}

		if err := p.bets.UpdateStatus(ctx, bet.ID, status, now); err != nil {
			log.Error("bet_status_update_failed", "bet_id", bet.ID, "error", err)
			continue
		}

		if status == domain.BetStatusWon {
			user, err := p.users.UpdateBalance(ctx, bet.UserID, amount)
			if err != nil {
				log.Error("payout_credit_failed", "bet_id", bet.ID, "error", err)
				continue
			}
			tx := &domain.Transaction{
				ID:            uuid.New().String(),
				UserID:        bet.UserID,
				Type:          txType,
				Amount:        amount,
				BalanceAfter:  user.Balance,
				ReferenceID:   bet.ID,
				CorrelationID: m.CorrelationID,
				CreatedAt:     now,
			}
			if err := p.txs.Save(ctx, tx); err != nil {
				log.Error("transaction_save_failed", "error", err)
			}
			_ = p.events.PublishWalletTransaction(ctx, tx)
			observability.BetsSettledTotal.WithLabelValues("won").Inc()
			log.Info("bet_settled_won", "bet_id", bet.ID, "payout", amount)
		} else {
			observability.BetsSettledTotal.WithLabelValues("lost").Inc()
			log.Info("bet_settled_lost", "bet_id", bet.ID)
		}
	}

	log.Info("market_settlement_complete", "total_bets", len(bets))
	return nil
}
