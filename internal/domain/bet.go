package domain

import "time"

type Bet struct {
	ID              string    `json:"id" bson:"_id"`
	UserID          string    `json:"user_id" bson:"user_id"`
	EventID         string    `json:"event_id" bson:"event_id"`
	MarketID        string    `json:"market_id" bson:"market_id"`
	OutcomeID       string    `json:"outcome_id" bson:"outcome_id"`
	Stake           float64   `json:"stake" bson:"stake"`
	OddsAtPlacement float64   `json:"odds_at_placement" bson:"odds_at_placement"`
	PotentialPayout float64   `json:"potential_payout" bson:"potential_payout"`
	Status          BetStatus `json:"status" bson:"status"`
	IdempotencyKey  string    `json:"idempotency_key" bson:"idempotency_key"`
	CorrelationID   string    `json:"correlation_id" bson:"correlation_id"`
	PlacedAt        time.Time `json:"placed_at" bson:"placed_at"`
	SettledAt       *time.Time `json:"settled_at,omitempty" bson:"settled_at,omitempty"`
}
