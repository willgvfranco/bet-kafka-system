package domain

import (
	"context"
	"time"
)

// --- Repository ports (MongoDB) ---

type UserRepository interface {
	FindByID(ctx context.Context, id string) (*User, error)
	UpdateBalance(ctx context.Context, id string, amount float64) (*User, error)
	Save(ctx context.Context, user *User) error
	FindAll(ctx context.Context) ([]User, error)
}

type EventRepository interface {
	FindByID(ctx context.Context, id string) (*Event, error)
	FindAll(ctx context.Context) ([]Event, error)
	FindByStatus(ctx context.Context, status EventStatus) ([]Event, error)
	Save(ctx context.Context, event *Event) error
	UpdateStatus(ctx context.Context, id string, status EventStatus) error
	UpdateMarketStatus(ctx context.Context, eventID, marketID string, status MarketStatus) error
	UpdateOutcomeResult(ctx context.Context, eventID, marketID, outcomeID string, result OutcomeResult) error
}

type BetRepository interface {
	Save(ctx context.Context, bet *Bet) error
	FindByID(ctx context.Context, id string) (*Bet, error)
	FindByUser(ctx context.Context, userID string) ([]Bet, error)
	FindByMarket(ctx context.Context, marketID string) ([]Bet, error)
	UpdateStatus(ctx context.Context, id string, status BetStatus, settledAt time.Time) error
}

type TransactionRepository interface {
	Save(ctx context.Context, tx *Transaction) error
	FindByUser(ctx context.Context, userID string) ([]Transaction, error)
}

// --- Cache ports (Redis) ---

type OddsCache interface {
	GetOdds(ctx context.Context, outcomeID string) (float64, error)
	SetOdds(ctx context.Context, outcomeID string, odds float64) error
	GetEventOdds(ctx context.Context, eventID string) (map[string]float64, error)
	PushOddsHistory(ctx context.Context, outcomeID string, changePercent float64) error
	GetOddsVelocity(ctx context.Context, outcomeID string) (float64, error)
}

type IdempotencyCache interface {
	Check(ctx context.Context, key string) (bool, error)
}

type MarketSuspensionCache interface {
	IsSuspended(ctx context.Context, marketID string) (bool, error)
	Suspend(ctx context.Context, marketID string, duration time.Duration) error
}

type FraudWindow interface {
	AddBet(ctx context.Context, userID string, bet *Bet) error
	GetRecentBets(ctx context.Context, userID string, window time.Duration) ([]Bet, error)
}

// --- Event publisher port (Kafka) ---

type EventPublisher interface {
	PublishBetPlaced(ctx context.Context, bet *Bet) error
	PublishBetSettled(ctx context.Context, eventID, marketID, winningOutcomeID, correlationID string) error
	PublishOddsUpdated(ctx context.Context, eventID, marketID, outcomeID string, oldOdds, newOdds float64, trigger string, correlationID string) error
	PublishGameEvent(ctx context.Context, event *GameEvent) error
	PublishMarketSuspended(ctx context.Context, eventID, marketID string, avgChange float64, correlationID string) error
	PublishFraudAlert(ctx context.Context, alert *FraudAlert) error
	PublishWalletTransaction(ctx context.Context, tx *Transaction) error
	Close() error
}

// --- Fraud types ---

type FraudAlert struct {
	AlertID       string         `json:"alert_id"`
	UserID        string         `json:"user_id"`
	AlertType     FraudAlertType `json:"alert_type"`
	Details       string         `json:"details"`
	CorrelationID string         `json:"correlation_id"`
	Severity      FraudSeverity  `json:"severity"`
	Timestamp     time.Time      `json:"timestamp"`
}

// --- Kafka message schemas ---

type OddsUpdatedMessage struct {
	EventID       string  `json:"event_id"`
	MarketID      string  `json:"market_id"`
	OutcomeID     string  `json:"outcome_id"`
	OldOdds       float64 `json:"old_odds"`
	NewOdds       float64 `json:"new_odds"`
	ChangePercent float64 `json:"change_percent"`
	Trigger       string  `json:"trigger"`
	CorrelationID string  `json:"correlation_id"`
	Timestamp     time.Time `json:"timestamp"`
}

type BetPlacedMessage struct {
	BetID           string  `json:"bet_id"`
	UserID          string  `json:"user_id"`
	EventID         string  `json:"event_id"`
	MarketID        string  `json:"market_id"`
	OutcomeID       string  `json:"outcome_id"`
	Stake           float64 `json:"stake"`
	OddsAtPlacement float64 `json:"odds_at_placement"`
	PotentialPayout float64 `json:"potential_payout"`
	IdempotencyKey  string  `json:"idempotency_key"`
	CorrelationID   string  `json:"correlation_id"`
	Timestamp       time.Time `json:"timestamp"`
}

type BetSettledMessage struct {
	EventID          string `json:"event_id"`
	MarketID         string `json:"market_id"`
	WinningOutcomeID string `json:"winning_outcome_id"`
	CorrelationID    string `json:"correlation_id"`
	Timestamp        time.Time `json:"timestamp"`
}

type MarketSuspendedMessage struct {
	EventID          string  `json:"event_id"`
	MarketID         string  `json:"market_id"`
	Reason           string  `json:"reason"`
	AvgChangePercent float64 `json:"avg_change_percent"`
	DurationSeconds  int     `json:"duration_seconds"`
	CorrelationID    string  `json:"correlation_id"`
	Timestamp        time.Time `json:"timestamp"`
}
