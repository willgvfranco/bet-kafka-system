# Bet Kafka System

## Overview
Go backend betting platform prototype — Kafka-centric event-driven architecture with 6 microservices.

## Tech Stack
- Go 1.23, segmentio/kafka-go (pure Go, no CGO), go-redis/v9, mongo-driver
- Docker: apache/kafka:3.7.0 (KRaft, no Zookeeper), Redis 7, MongoDB 7
- Observability: Prometheus client_golang, Grafana, slog JSON logging

## Architecture
- Clean Architecture: `internal/domain/` (pure, zero imports) → `internal/repository/`, `internal/cache/`, `internal/kafka/` (infra) → `internal/processor/` (use cases) → `internal/handler/` (HTTP)
- All interfaces defined in `internal/domain/ports.go` — implementations injected via constructors
- 6 binaries in `cmd/`: api, data-feed-simulator, odds-engine, bet-processor, odds-consumer, fraud-detector
- Seeder in `cmd/seeder/`

## Kafka Topics
game.events, odds.updated, bets.placed, bets.settled, markets.suspended, fraud.alerts, wallet.transactions

## Commands
- `docker compose up -d` — start everything (infra + Go services)
- `go run ./cmd/seeder/` — seed MongoDB + Redis (local dev)
- `go run ./cmd/<service>/` — run a service locally
- `go build ./...` — verify all packages compile
- API: http://localhost:8080, Kafka UI: http://localhost:8085, Grafana: http://localhost:3000 (admin/admin)

## Code Conventions
- Structured logging with `log/slog` JSON handler — event names in snake_case, context as kwargs
- Commit messages: Conventional Commits format (feat, fix, chore, etc.), English, no emojis
- MongoDB atomic balance ops via `$inc` with `$gte` guard — no distributed transactions
- Kafka partition keys: user_id for bets, outcome_id for odds, event_id for game events
- Manual offset commit after successful processing (at-least-once + idempotency via Redis SET NX)
- Fraud rules follow Open/Closed principle — each rule implements `FraudRule` interface

## Gotchas
- `bitnami/kafka` images may not be available — use `apache/kafka:3.7.0`
- When testing Docker services locally, kill stale `go run` processes to avoid port conflicts
- Odds change fast from data-feed-simulator — use live odds from `/events/:id/odds` when placing bets
- Market suspension (30s auto-expire) triggers when odds velocity > 20% — bets rejected during suspension
