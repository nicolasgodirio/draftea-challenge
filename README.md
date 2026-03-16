# Draftea Payment System

A microservices-based payment system built in Go, structured as individual AWS Lambda functions. The system is composed of three independent services that share a single PostgreSQL database.

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                      API Gateway                        │
└──────────────┬──────────────┬──────────────┬────────────┘
               │              │              │
       POST /payments   GET /balances  GET /transactions
               │              │              │
     ┌─────────▼──────┐ ┌─────▼──────┐ ┌────▼────────────┐
     │ createpayment  │ │ getbalances│ │ gettransactions │
     └─────────┬──────┘ └─────┬──────┘ └────┬────────────┘
               │              │              │
               └──────────────▼──────────────┘
                          PostgreSQL
```

### Services

| Service | Method | Path | Description |
|---------|--------|------|-------------|
| `createpayment` | POST | `/payments` | Creates a new payment for the authenticated user, validates balance, publishes a Kafka event |
| `getbalances` | GET | `/balances` | Returns the wallet balance for the authenticated user |
| `gettransactions` | GET | `/transactions` | Returns the transaction history for the authenticated user |

### Technology Stack

- **Language**: Go 1.25
- **Database**: PostgreSQL 16 (via GORM query builder)
- **Tracing**: OpenTelemetry (stdout exporter — swap for OTLP/X-Ray in production)
- **Event publishing**: Kafka (simulated — stubbed for wiring)
- **Runtime**: AWS Lambda (via `aws-lambda-go` + `aws-lambda-go-api-proxy`)
- **Testing**: `go test` + `go.uber.org/mock`

### Design Decisions

- **Per-service module isolation**: each service (`createpayment`, `getbalances`, `gettransactions`) has its own `internal/` tree — domain, usecase, infra — with no shared code between services. This enforces proper bounded contexts.
- **Handler-layer validation**: input validation (e.g. `amount > 0`) lives in the HTTP handler so it never reaches the usecase.
- **Async debit**: the `createpayment` service only creates the payment record and publishes a `payment.created` Kafka event. Balance debiting is handled asynchronously by a separate consumer (not part of this repo).
- **GORM query builder**: all queries use GORM for type-safe, composable SQL.
- **OTel tracing**: every layer (handler → usecase → repository) carries a span, propagating the trace context end-to-end.
- **JWT auth**: middleware extracts `sub` claim from a Bearer JWT and stores it as `user_id` in the request context.

---

## Prerequisites

- [Docker](https://www.docker.com/) and [Docker Compose](https://docs.docker.com/compose/) v2+
- `curl` and `python3` (for the test script)

---

## Running with Docker Compose

### 1. Clone and enter the project

```bash
git clone <repo-url>
cd draftea-challenge
```

### 2. Start all services

```bash
docker compose up --build
```

This will:
1. Start a PostgreSQL 16 container and apply the migration in `migrations/001_init.sql`
2. Build and start the `createpayment` Lambda container on port `8080`

Seed data included in the migration:

| user_id | initial balance | currency |
|---------|----------------|----------|
| `b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11` | 10,000.00 | ARS |
| `b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a11` | 5,000.00 | ARS |

### 3. Run additional services individually (optional)

The `getbalances` and `gettransactions` services share the same database. You can run them locally against the Docker Postgres instance:

```bash
# Terminal 1 — getbalances
DB_HOST=localhost DB_PORT=5432 DB_USER=draftea DB_PASSWORD=draftea DB_NAME=payments DB_SSLMODE=disable \
  go run ./getbalances/cmd/...

# Terminal 2 — gettransactions  
DB_HOST=localhost DB_PORT=5432 DB_USER=draftea DB_PASSWORD=draftea DB_NAME=payments DB_SSLMODE=disable \
  go run ./gettransactions/cmd/...
```

---

## API Reference

All endpoints require a `Authorization: Bearer <JWT>` header. The JWT must contain a `sub` claim with the user ID.

### Create Payment

```
POST /payments
```

**Request body:**
```json
{
  "amount": 150.00
}
```

**Response** `202 Accepted`:
```json
{
  "status": "PENDING"
}
```

**Error codes:**

| HTTP | Condition |
|------|-----------|
| `400` | Amount is zero or negative, malformed JSON |
| `401` | Missing or invalid JWT |
| `404` | Wallet not found for user |
| `422` | Insufficient funds |
| `500` | Internal error |

---

### Get Balance

```
GET /balances
```

**Response** `200 OK`:
```json
{
  "user_id": "b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11",
  "balance": 10000.00,
  "currency": "ARS"
}
```

---

### Get Transactions

```
GET /transactions
```

**Response** `200 OK`:
```json
{
  "user_id": "b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11",
  "transactions": [
    {
      "id": "...",
      "type": "CREDIT",
      "amount": 10000.00,
      "reference_id": null,
      "description": "Initial deposit",
      "created_at": "2026-03-15T00:00:00Z"
    }
  ]
}
```

---

## Calling the Lambda emulator

The Docker Compose setup runs the Lambda Runtime Interface Emulator (RIE). Requests must be sent as Lambda proxy events to the emulator endpoint:

```
POST http://localhost:8080/2015-03-31/functions/function/invocations
```

### Example — Create Payment

```bash
USER_ID="b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"

# Build a fake JWT with sub=<user_id>
HEADER=$(echo -n '{"alg":"none"}' | base64 | tr -d '=' | tr '+/' '-_')
PAYLOAD=$(echo -n "{\"sub\":\"$USER_ID\"}" | base64 | tr -d '=' | tr '+/' '-_')
TOKEN="$HEADER.$PAYLOAD.sig"

curl -s -XPOST http://localhost:8080/2015-03-31/functions/function/invocations \
  -H "Content-Type: application/json" \
  -d "{
    \"httpMethod\": \"POST\",
    \"path\": \"/payments\",
    \"headers\": {\"Authorization\": \"Bearer $TOKEN\"},
    \"body\": \"{\\\"amount\\\": 100.50}\"
  }" | python3 -m json.tool
```

### Example — Get Balance

```bash
curl -s -XPOST http://localhost:8080/2015-03-31/functions/function/invocations \
  -H "Content-Type: application/json" \
  -d "{
    \"httpMethod\": \"GET\",
    \"path\": \"/balances\",
    \"headers\": {\"Authorization\": \"Bearer $TOKEN\"},
    \"body\": \"\"
  }" | python3 -m json.tool
```

---

## Running Tests

```bash
# All services
go test ./createpayment/... ./getbalances/... ./gettransactions/... -v

# Single service
go test ./getbalances/... -v

# With coverage
go test ./createpayment/... ./getbalances/... ./gettransactions/... -cover
```

---

## Project Structure

```
draftea-challenge/
├── migrations/
│   └── 001_init.sql          # Schema + seed data
├── createpayment/
│   ├── Dockerfile
│   ├── cmd/
│   │   ├── main.go           # Lambda entrypoint
│   │   └── setup.go          # Wiring (tracer, db, repos, usecase, handler)
│   └── internal/
│       ├── domain/           # Payment, Wallet, errors
│       ├── usecase/          # CreatePayment logic + interfaces
│       └── infra/
│           ├── handler/      # HTTP handler, middleware, router, DTOs, mocks, tests
│           ├── publisher/kafka/  # Kafka publisher (simulated)
│           └── repository/postgres/  # GORM repos
├── getbalances/
│   ├── Dockerfile
│   ├── cmd/
│   └── internal/             # Same structure as createpayment
├── gettransactions/
│   ├── Dockerfile
│   ├── cmd/
│   └── internal/             # Same structure as createpayment
├── docker-compose.yml
├── go.mod
└── test.sh                   # Manual smoke test script
```
