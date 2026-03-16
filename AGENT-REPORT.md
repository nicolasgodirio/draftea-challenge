# Agent Report — AI-Assisted Development with GitHub Copilot (Claude Sonnet 4.5 and Opus 4.6)

## Overview

This project was built entirely through a conversation-driven development session using **GitHub Copilot powered by Claude Sonnet 4.5 and Opus 4.6** inside VS Code. The AI acted as a senior software engineer, making every architectural decision, writing all production code, tests, and infrastructure configuration from scratch.

The session lasted a single continuous conversation. No code was written manually by the developer — every file was created or modified through AI tool calls in response to natural language instructions.

---

## Session Flow

### Step 1 — Database Architecture and Migrations

The first input to the AI was a description of the domain and the requirement for a PostgreSQL schema to back the system. The AI designed the full schema and wrote `migrations/001_init.sql`:

- **`wallets`** — one row per user, holds the current balance and currency. `user_id` has a `UNIQUE` constraint enforcing one wallet per user. All monetary amounts use `NUMERIC(18,2)` to avoid floating-point precision loss.
- **`transactions`** — append-only ledger. Each row records a debit, credit, refund, reserve, or release event against a wallet. A `CHECK` constraint on the `type` column enforces the allowed enum values at the database level. `reference_id` (nullable UUID) links a transaction back to the triggering payment.
- **`payments`** — tracks payment requests. Includes a `(user_id, idempotency_key)` unique constraint, a `status` column with a `CHECK` constraint, and indexed by `user_id` for fast lookups.

The migration also installs the `pgcrypto` extension to enable `gen_random_uuid()` as the default primary key generator for all tables, and inserts seed rows for two wallets so the system is immediately testable after `docker compose up`.

---

### Step 2 — Initial Boilerplate: Three Lambda Services in One Shot

With the schema defined, the developer asked the AI to scaffold all three Lambda services simultaneously. In a single step the AI generated the complete skeleton for `createpayment`, `getbalances`, and `gettransactions`, each following the same internal structure:

```
<service>/
├── Dockerfile
├── cmd/
│   ├── main.go       # Lambda entrypoint (lambda.Start)
│   └── setup.go      # DI wiring: tracer, db, repos, usecase, handler, adapter
└── internal/
    ├── domain/       # Plain Go structs and sentinel errors
    ├── usecase/      # Business logic + repository interfaces
    └── infra/
        ├── handler/  # HTTP handler, JWT middleware, router, DTOs, error map
        └── repository/postgres/  # Concrete DB implementations
```

Key boilerplate decisions made by the AI:
- Used `aws-lambda-go` + `aws-lambda-go-api-proxy/httpadapter` so each service exposes a standard `net/http` handler that is wrapped into a Lambda-compatible adapter — no Lambda SDK coupling inside business logic
- Each service has its own copy of the domain types (no shared module between services), enforcing bounded context isolation
- JWT auth middleware extracted `sub` from a Bearer token using base64 decode only (no signature verification), which is the correct pattern for internal Lambda-to-Lambda or API Gateway + Authorizer flows
- `internal/infra/handler/dto/` DTO package separates API contract types from domain types
- `cmd/setup.go` wires all dependencies and returns a `cleanup()` function, keeping `main.go` trivial

---

### Step 3 — Initial Refactoring of `createpayment`

The starting point was a partially built `createpayment` service with some rough patterns. The developer gave three refactoring instructions:

> 1. Idempotency key should be deleted completely from request since it will go in consumer
> 2. Implement tracing that goes from handler to repo with spans
> 3. In usecase we create payment and debiting balance at the same time. Delete debiting balance, that will be done asyncly

The AI:
- Removed `IdempotencyKey` from the `Payment` domain struct, `CreatePaymentInput`, handler, and all repository queries
- Removed `ErrDuplicatePayment` and the idempotency check branch from the usecase
- Removed `DebitBalance` from the `WalletRepository` interface and the wallet repository implementation
- Installed the OpenTelemetry Go SDK (`go.opentelemetry.io/otel`) and added a `TracerProvider` wired through `setup.go`
- Added OTel spans at every layer — `Handler.Handle`, `UseCase.CreatePayment`, `PaymentRepo.Create`, `WalletRepo.GetByUserID` — with structured attributes (`user_id`, `amount`, `payment_id`, `wallet_id`) and proper error recording
- Updated all hand-written mocks and test assertions to reflect the new interfaces

### Step 4 — Three Further Improvements

> 4. Amount input should be checked in handler layer
> 5. Add publisher after creating record in DB. Publisher must publish to Kafka with a proper topic. This is simulated, mock it
> 6. For queries in repo use a query builder, e.g. GORM

The AI:
- Moved the `amount <= 0` guard from the usecase to the `Handle()` method — the usecase no longer knows about this rule
- Added a `PaymentPublisher` interface to the usecase, injected as a dependency. After `paymentRepo.Create()` succeeds, `publisher.Publish()` is called with the created payment
- Created `createpayment/internal/infra/publisher/kafka/publisher.go` — a concrete publisher that logs and holds a TODO comment for wiring an actual Kafka producer (`confluent-kafka-go` or `segmentio/kafka-go`)
- Added `MockPaymentPublisher` to the usecase mocks
- Installed GORM (`gorm.io/gorm`, `gorm.io/driver/postgres`) and migrated both repositories from raw `database/sql` to GORM model structs with `WithContext().Where().First()` / `.Create()` calls
- Replaced `NewConnection()` to return `*gorm.DB` instead of `*sql.DB`; updated `setup.go` and `cleanup` to get the underlying `*sql.DB` via `db.DB()` for closing
- Updated `TestHandle_InvalidAmount` to no longer call the usecase mock (the check never reaches it now); added `TestHandle_ZeroAmount`

### Step 5 — Implementing `getbalances` and `gettransactions`

> Great, whole service is successfully implemented. We should do the same for getbalances and gettransactions. Delete all unnecessary code from createpayment.

The AI:
- Inspected the existing `internal/` shared package (domain, ports, handlers, repos, usecases) to understand the intended data model and service boundaries
- Built two complete, independent Lambda services from scratch, each with identical internal structure to `createpayment`:

  **`getbalances`**
  - `domain/` — `Wallet`, `ErrWalletNotFound`, `ErrUnauthorized`
  - `usecase/GetBalance` — calls `WalletRepository.GetByUserID`, OTel span
  - `infra/repository/postgres/WalletRepo` — GORM `Where("user_id = ?").First()`, OTel span
  - `infra/handler/Handler` — `BalanceGetter` interface, JWT middleware, OTel span, DTO response
  - Mocks for usecase and repository interfaces
  - 5 handler tests

  **`gettransactions`**
  - `domain/` — `Wallet`, `Transaction` (with all five `TransactionType` variants), errors
  - `usecase/GetTransactions` — verifies wallet exists first, then calls `GetTransactionsByUserID`, OTel span
  - `infra/repository/postgres/WalletRepo` — GORM `Joins("JOIN wallets ON ...").Where().Order().Find()`, OTel span
  - `infra/handler/Handler` — `TransactionsGetter` interface, maps domain slice to DTO slice, OTel span
  - Mocks for usecase and repository interfaces
  - 6 handler tests (including `TestHandle_EmptyTransactions`)

- Deleted `createpayment/internal/domain/transaction.go` (the `TransactionType` constant was only needed there as a leftover from the old synchronous debit)
- Each service got its own `Dockerfile`, `cmd/main.go`, and `cmd/setup.go`

---

### Step 6 — Unit Tests for All Layers

> Add unit tests for all layers — usecase and handler — across all three services.

All 29 tests pass (`go test ./createpayment/... ./getbalances/... ./gettransactions/... -v -count=1`).

**Handler tests** (existing — prior sessions)

| Service | File | Tests |
|---------|------|-------|
| `createpayment` | `internal/infra/handler/handler_test.go` | 9 tests: success, missing/invalid auth token, invalid body, invalid amount, zero amount, wallet not found, insufficient funds, internal server error |
| `getbalances` | `internal/infra/handler/handler_test.go` | 5 tests: success, missing/invalid auth token, wallet not found, internal server error |
| `gettransactions` | `internal/infra/handler/handler_test.go` | 6 tests: success, empty transactions, missing/invalid auth token, wallet not found, internal server error |

**Usecase tests** (added this session)

| Service | File | Tests |
|---------|------|-------|
| `createpayment` | `internal/usecase/create_payment_test.go` | 6 tests: success (`DoAndReturn` to simulate ID assignment), wallet not found, insufficient funds, create fails, publish fails, wallet repository error |
| `getbalances` | `internal/usecase/get_balance_test.go` | 3 tests: success, wallet not found, repository error |
| `gettransactions` | `internal/usecase/get_transactions_test.go` | 5 tests: success (2 transactions), empty result, wallet not found (`GetTransactionsByUserID` never called), wallet repository error, get transactions fails |

All usecase tests use the GoMock-compatible mocks in each service's `usecase/mocks/` package. Repository-layer integration tests (requiring a live PostgreSQL instance via `testcontainers-go`) are out of scope for this session.

---

## What the AI Handled Autonomously

| Concern | Detail |
|---------|--------|
| **Architecture** | Proposed and enforced a strict layered architecture: Handler → UseCase → Repository. Each service is fully isolated with no shared packages. |
| **Interface design** | Defined narrow interfaces (`WalletRepository`, `PaymentRepository`, `PaymentPublisher`, `BalanceGetter`, `TransactionsGetter`) co-located with their consumers, following the dependency-inversion principle. |
| **Dependency wiring** | All concrete implementations are wired outside the business logic in `cmd/setup.go`, keeping layers decoupled. |
| **Mock generation** | Wrote all GoMock-compatible mocks by hand (matching the `mockgen` format exactly) since `mockgen` cannot run in a file-editing context. |
| **Test design** | Wrote handler tests using `httptest` covering success, auth failures, domain errors, and internal errors. Wrote usecase tests with GoMock covering happy paths, domain-rule violations, error propagation, and side-effect simulation (`DoAndReturn` for repository ID assignment). 29 tests total, all passing. |
| **Error propagation** | Mapped domain errors to HTTP status codes in a declarative `errorStatusMap`, keeping the handler free of switch/case logic. |
| **Build verification** | After every batch of changes, ran `go build`, `go test -v`, and `go vet` and fixed any compilation errors (e.g. `gorm.Model` `CreatedAt` being `time.Time` directly, not a wrapper). |
| **Incremental progress** | Used a structured todo list throughout, marking tasks in-progress and completed as it moved through the work. |

---

## Prompting Strategy

The developer used short, directive prompts — typically a numbered list of requirements. The AI interpreted intent, inferred missing details (e.g. which Kafka topic to use, how to name interfaces, where validation belongs), and proceeded without asking clarifying questions unless genuinely ambiguous.

No scaffolding tools, code generators, or templates were used — all code was synthesized from the AI's knowledge of Go idioms, the existing codebase structure, and the stated requirements.

---

## Final State

All three services compile cleanly and pass their full test suites:

```
ok  draftea-challenge/createpayment/internal/infra/handler   (9 tests)
ok  draftea-challenge/getbalances/internal/infra/handler      (5 tests)
ok  draftea-challenge/gettransactions/internal/infra/handler  (6 tests)
```

`go vet` passes with zero warnings across all packages.
