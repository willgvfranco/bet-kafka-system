# Bet Kafka System

A backend-only betting platform prototype built with **Go**, **Apache Kafka**, **MongoDB**, and **Redis**. Demonstrates Kafka-centric event-driven architecture with 6 microservices processing bets, odds, fraud detection, and market suspension in real time.

## Architecture

```
                   ┌──────────────┐
                   │  REST API    │ :8080
                   └──────┬───────┘
                          │ publishes
                          ▼
┌─────────────────────────────────────────────────────────────────┐
│                        Apache Kafka                             │
│  game.events │ odds.updated │ bets.placed │ bets.settled │ ... │
└──┬───────┬──────────┬───────────┬──────────────┬────────────────┘
   │       │          │           │              │
   ▼       ▼          ▼           ▼              ▼
┌──────┐ ┌──────┐ ┌──────────┐ ┌──────────┐ ┌──────────────┐
│ Odds │ │ Odds │ │   Bet    │ │  Fraud   │ │  Data Feed   │
│Engine│ │Consu-│ │Processor │ │Detector  │ │  Simulator   │
│      │ │ mer  │ │          │ │          │ │              │
└──────┘ └──────┘ └──────────┘ └──────────┘ └──────────────┘
                       │
                  ┌────┴────┐
                  │MongoDB  │  │Redis│
                  └─────────┘  └─────┘
```

### Services

| Service | Description |
|---------|-------------|
| **api** | REST API for placing bets, listing events, checking wallets, admin settlement |
| **data-feed-simulator** | Simulates a live sports data provider (like Sportradar), producing game events |
| **odds-engine** | Consumes game events, recalculates odds based on event impact (goals, cards, etc.) |
| **odds-consumer** | Updates Redis cache with latest odds, detects velocity spikes, triggers market suspension |
| **bet-processor** | Processes bet placement (validation, balance debit, persistence) and settlement (payout) |
| **fraud-detector** | Parallel consumer analyzing bets for fraud patterns (rate limiting, opposite bets, suspicious amounts) |

### Kafka Topics

| Topic | Partitions | Key | Purpose |
|-------|-----------|-----|---------|
| `game.events` | 6 | event_id | Live match events (goals, cards, halftime) |
| `odds.updated` | 6 | outcome_id | Odds recalculations |
| `bets.placed` | 12 | user_id | Bet placement requests |
| `bets.settled` | 6 | event_id | Market settlement triggers |
| `markets.suspended` | 3 | market_id | Market suspension notifications |
| `fraud.alerts` | 3 | user_id | Fraud detection alerts |
| `wallet.transactions` | 6 | user_id | Balance change audit trail |

## Tech Stack

- **Go 1.23** with standard library HTTP server (Go 1.22+ routing patterns)
- **Apache Kafka 3.7.0** (KRaft mode, no Zookeeper)
- **segmentio/kafka-go** (pure Go Kafka client, no CGO)
- **MongoDB 7** for persistence
- **Redis 7** for caching, idempotency, market suspension flags, fraud sliding windows
- **Prometheus + Grafana** for observability
- **Docker Compose** for the full stack

## Quick Start

### Prerequisites

- Docker and Docker Compose

### Run Everything

```bash
docker compose up -d
```

This starts all infrastructure (Kafka, MongoDB, Redis, Prometheus, Grafana) plus all 6 Go services. The seeder runs first to populate initial data (users, events, markets, odds).

### Endpoints

| URL | Description |
|-----|-------------|
| http://localhost:8080 | REST API |
| http://localhost:8085 | Kafka UI |
| http://localhost:3000 | Grafana (admin/admin) |
| http://localhost:9090 | Prometheus |

## API Reference

### Events

```bash
# List all events
curl http://localhost:8080/api/v1/events

# Get event details
curl http://localhost:8080/api/v1/events/{id}

# Get live odds for an event
curl http://localhost:8080/api/v1/events/{id}/odds
```

### Bets

```bash
# Place a bet (async — returns 202 Accepted)
curl -X POST http://localhost:8080/api/v1/bets \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "william",
    "event_id": "<event_id>",
    "market_id": "<market_id>",
    "outcome_id": "<outcome_id>",
    "stake": 50,
    "odds": 1.85
  }'

# Get bets for a user
curl "http://localhost:8080/api/v1/bets?user_id=william"

# Get bet by ID
curl http://localhost:8080/api/v1/bets/{id}
```

### Wallet

```bash
# Get wallet balance and transactions
curl "http://localhost:8080/api/v1/wallet?user_id=william"
```

### Admin

```bash
# Settle a market (declare winning outcome)
curl -X POST http://localhost:8080/api/v1/admin/settle \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "<event_id>",
    "market_id": "<market_id>",
    "winning_outcome_id": "<outcome_id>"
  }'

# Update event status
curl -X POST http://localhost:8080/api/v1/admin/event-status \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "<event_id>",
    "status": "FINISHED"
  }'
```

### Typical Flow

1. List events: `GET /api/v1/events`
2. Get live odds: `GET /api/v1/events/{id}/odds`
3. Place a bet using current odds: `POST /api/v1/bets`
4. The bet flows through Kafka: validation, balance debit, fraud check
5. Settle the market: `POST /api/v1/admin/settle`
6. Check wallet for payout: `GET /api/v1/wallet?user_id=william`

> **Tip:** Always fetch live odds before placing a bet. The odds-engine recalculates continuously based on simulated game events, and bets are rejected if odds drift exceeds 5%.

## Project Structure

```
cmd/
  api/                    REST API server
  bet-processor/          Bet placement + settlement consumer
  data-feed-simulator/    Game event simulator (producer)
  fraud-detector/         Fraud pattern detection consumer
  odds-consumer/          Redis cache updater + market suspension
  odds-engine/            Odds recalculation consumer
  seeder/                 Initial data seeder

internal/
  domain/                 Pure domain: entities, enums, interfaces (ports)
  repository/             MongoDB implementations
  cache/                  Redis implementations (odds, idempotency, suspension, fraud)
  kafka/                  Kafka producer + consumer infrastructure
  processor/              Use cases (bet processing, odds engine, fraud rules)
  handler/                HTTP handlers
  config/                 Environment-based configuration
  observability/          Prometheus metrics + middleware

grafana/                  Dashboard + datasource provisioning
prometheus/               Scrape configuration
planning/                 Architecture spec and design notes
```

## Design Decisions

### Clean Architecture
Strict dependency rule: `domain/` (pure, zero imports) -> `repository/`, `cache/`, `kafka/` (infrastructure) -> `processor/` (use cases) -> `handler/` (HTTP). All interfaces defined in `domain/ports.go`, implementations injected via constructors.

### At-Least-Once Delivery + Idempotency
Kafka consumers commit offsets manually after successful processing. Duplicate messages are handled via Redis `SET NX` idempotency keys with TTL.

### Atomic Balance Operations
No distributed transactions. MongoDB `$inc` with `$gte` guard ensures atomic balance debit — if balance is insufficient, the update fails atomically.

### Market Suspension
When the odds-consumer detects average odds velocity exceeding 20%, the market is automatically suspended for 30 seconds via a Redis key with TTL. All bets on suspended markets are rejected.

### Fraud Detection (Open/Closed Principle)
Each fraud rule implements the `FraudRule` interface. New rules can be added without modifying existing code:
- **Rate Limit**: > 5 bets in 60 seconds
- **Opposite Bets**: Same event, different outcomes
- **Suspicious Amount**: Stake > 10x user's average

### Partition Key Strategy
- `user_id` for bets and wallet — ensures per-user ordering
- `outcome_id` for odds — groups odds updates for the same outcome
- `event_id` for game events — groups events for the same match

## Local Development

Run infrastructure only, then start services individually:

```bash
# Start infra
docker compose up -d kafka redis mongodb prometheus grafana kafka-ui

# Wait for Kafka to be healthy, then create topics
docker compose up kafka-init

# Seed data
go run ./cmd/seeder/

# Run any service
go run ./cmd/api/
go run ./cmd/odds-engine/
go run ./cmd/bet-processor/
# etc.
```

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `KAFKA_BROKERS` | `localhost:9092` | Kafka bootstrap servers |
| `REDIS_ADDR` | `localhost:6379` | Redis address |
| `MONGO_URI` | `mongodb://localhost:27017/betting` | MongoDB connection URI |
| `HTTP_PORT` | `8080` | API server port |
| `METRICS_PORT` | `9100` | Prometheus metrics port (non-API services) |

### Build

```bash
go build ./...
```

## Observability

All services export Prometheus metrics. Pre-built Grafana dashboard is auto-provisioned at startup with panels for:

- **Kafka Health**: Messages produced/consumed per second, processing duration P95, errors
- **Business Metrics**: Bets placed/rejected/settled per minute, fraud alerts
- **Odds**: Updates per second, change distribution, market suspensions
- **API**: Request rate by endpoint, response time P95
- **Game Simulation**: Events by type

## License

MIT
