package domain

import "time"

type Outcome struct {
	OutcomeID  string        `json:"outcome_id" bson:"outcome_id"`
	Name       string        `json:"name" bson:"name"`
	InitialOdds float64      `json:"initial_odds" bson:"initial_odds"`
	Result     OutcomeResult `json:"result" bson:"result"`
}

type Market struct {
	MarketID string       `json:"market_id" bson:"market_id"`
	Type     MarketType   `json:"type" bson:"type"`
	Status   MarketStatus `json:"status" bson:"status"`
	Outcomes []Outcome    `json:"outcomes" bson:"outcomes"`
}

type Event struct {
	ID        string      `json:"id" bson:"_id"`
	Name      string      `json:"name" bson:"name"`
	Sport     string      `json:"sport" bson:"sport"`
	Status    EventStatus `json:"status" bson:"status"`
	StartTime time.Time   `json:"start_time" bson:"start_time"`
	Markets   []Market    `json:"markets" bson:"markets"`
}
