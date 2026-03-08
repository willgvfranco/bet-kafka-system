package domain

import "time"

type Score struct {
	Home int `json:"home"`
	Away int `json:"away"`
}

type GameEvent struct {
	EventID       string        `json:"event_id"`
	Type          GameEventType `json:"type"`
	Team          string        `json:"team"`
	Player        string        `json:"player"`
	Minute        int           `json:"minute"`
	Score         Score         `json:"score"`
	CorrelationID string        `json:"correlation_id"`
	Timestamp     time.Time     `json:"timestamp"`
}
