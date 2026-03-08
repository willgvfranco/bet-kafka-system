package domain

type EventStatus string

const (
	EventStatusUpcoming EventStatus = "UPCOMING"
	EventStatusLive     EventStatus = "LIVE"
	EventStatusFinished EventStatus = "FINISHED"
)

type MarketStatus string

const (
	MarketStatusOpen      MarketStatus = "OPEN"
	MarketStatusSuspended MarketStatus = "SUSPENDED"
	MarketStatusClosed    MarketStatus = "CLOSED"
	MarketStatusSettled   MarketStatus = "SETTLED"
)

type MarketType string

const (
	MarketTypeMatchWinner MarketType = "MATCH_WINNER"
	MarketTypeOverUnder   MarketType = "OVER_UNDER"
)

type OutcomeResult string

const (
	OutcomeResultPending OutcomeResult = "PENDING"
	OutcomeResultWon     OutcomeResult = "WON"
	OutcomeResultLost    OutcomeResult = "LOST"
)

type BetStatus string

const (
	BetStatusPending   BetStatus = "PENDING"
	BetStatusWon       BetStatus = "WON"
	BetStatusLost      BetStatus = "LOST"
	BetStatusCancelled BetStatus = "CANCELLED"
)

type UserStatus string

const (
	UserStatusActive    UserStatus = "ACTIVE"
	UserStatusSuspended UserStatus = "SUSPENDED"
)

type TransactionType string

const (
	TransactionTypeBetPlaced TransactionType = "BET_PLACED"
	TransactionTypeBetWon    TransactionType = "BET_WON"
	TransactionTypeBetLost   TransactionType = "BET_LOST"
	TransactionTypeRefund    TransactionType = "REFUND"
)

type GameEventType string

const (
	GameEventGoal       GameEventType = "GOAL"
	GameEventYellowCard GameEventType = "YELLOW_CARD"
	GameEventRedCard    GameEventType = "RED_CARD"
	GameEventHalftime   GameEventType = "HALFTIME"
	GameEventSecondHalf GameEventType = "SECOND_HALF"
	GameEventFullTime   GameEventType = "FULL_TIME"
)

type FraudAlertType string

const (
	FraudAlertOppositeBets     FraudAlertType = "OPPOSITE_BETS"
	FraudAlertRateLimit        FraudAlertType = "RATE_LIMIT"
	FraudAlertSuspiciousAmount FraudAlertType = "SUSPICIOUS_PATTERN"
)

type FraudSeverity string

const (
	FraudSeverityLow    FraudSeverity = "LOW"
	FraudSeverityMedium FraudSeverity = "MEDIUM"
	FraudSeverityHigh   FraudSeverity = "HIGH"
)
