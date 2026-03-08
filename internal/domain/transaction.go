package domain

import "time"

type Transaction struct {
	ID            string          `json:"id" bson:"_id"`
	UserID        string          `json:"user_id" bson:"user_id"`
	Type          TransactionType `json:"type" bson:"type"`
	Amount        float64         `json:"amount" bson:"amount"`
	BalanceAfter  float64         `json:"balance_after" bson:"balance_after"`
	ReferenceID   string          `json:"reference_id" bson:"reference_id"`
	CorrelationID string          `json:"correlation_id" bson:"correlation_id"`
	CreatedAt     time.Time       `json:"created_at" bson:"created_at"`
}
