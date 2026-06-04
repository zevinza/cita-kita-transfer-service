# Cita Kita Transfer Service

Mini financial transfer API built in Go. It supports single and batch transfers with **atomic balance updates**, **idempotency**, in-process **account locking**, and **retry with backoff** for transient failures.

Balances are stored in **Redis** using a Lua script so debit, credit, and idempotency recording happen in one atomic step.

## Assumptions

Critical design decisions for correctness and scope:

1. **Redis Lua is the source of truth** — Debit, credit, and idempotency are applied in one atomic script. Correctness across instances depends on Redis, not the app layer.
2. **Client-owned idempotency keys** — Callers must send a unique `idempotency_key` per transfer. A duplicate key returns the stored result and must not debit twice.
3. **In-process lock is not distributed** — `AccountLocker` only reduces races inside one process. Multiple replicas rely on Redis atomicity, not shared mutexes.
4. **Optimistic balance check** — The service checks balance before apply; under concurrency, Lua may still reject with `insufficient balance`.

## Features

- **Single transfer** — Move funds between two accounts with balance checks
- **Batch transfer** — Process multiple transfers concurrently; per-item success/failure summary
- **Idempotency** — Re-submitting the same `idempotency_key` returns the cached result without double-spending
- **Atomic updates** — Redis Lua script ensures consistent balance and idempotency state
- **Account locking** — In-process mutexes per account pair to reduce race conditions before Redis applies the transfer
- **Retry** — Configurable retries with backoff when `ApplyTransfer` fails (non-retryable on insufficient balance)
- **Seeded accounts** — Five demo users (A1–A5) with initial balances on startup
- **OpenAPI** — Swagger UI at `/swagger/index.html`

## Tech Stack

| Layer        | Choice              |
| ------------ | ------------------- |
| HTTP         | [Fiber](https://gofiber.io/) v2 |
| Storage      | [Redis](https://redis.io/) go-redis v9 |
| API docs     | [swaggo/swag](https://github.com/swaggo/swag) |
| Testing      | testify, gomock     |
| Go version   | 1.26.3              |

## Prerequisites

- Go 1.26+
- Redis 7+ (local or via Docker)
- Optional: Docker Compose for Redis only

## Quick Start

### 1. Start Redis

```bash
docker compose up -d
```

### 2. Configure environment

```bash
cp .env.example .env
```

| Variable              | Default     | Description                                      |
| --------------------- | ----------- | ------------------------------------------------ |
| `PORT`                | `8000`      | HTTP listen port                                 |
| `REDIS_HOST`          | `localhost` | Redis host                                       |
| `REDIS_PORT`          | `6379`      | Redis port                                       |
| `REDIS_PASSWORD`      | *(empty)*   | Redis password (if set)                          |
| `REDIS_INDEX`         | `0`         | Redis database index                             |
| `MAX_RETRIES`         | `3`         | Max retries after a failed `ApplyTransfer`       |
| `RETRY_BASE_BACKOFF`  | `1s`        | Backoff between retries (duration or seconds)    |
| `CACHE_TTL_SECONDS`   | `3600`      | Cache TTL (reserved for future use)              |
| `BATCH_CONCURRENT`    | `false`     | Reserved flag in config                          |

### 3. Run the service

```bash
go run .
```

On startup, seed balances are written to Redis with `SETNX` (existing keys are not overwritten).

### 4. Verify

```bash
curl http://localhost:8000/health
# OK

curl http://localhost:8000/accounts
```

Swagger UI: [http://localhost:8000/swagger/index.html](http://localhost:8000/swagger/index.html)

## API Reference

### Health

```http
GET /health
```

Returns `OK` when the process is running.

### List accounts

```http
GET /accounts
```

**Response (200)**

```json
{
  "users": [
    { "id": "A1", "name": "John Doe", "balance": 1000 },
    { "id": "A2", "name": "Jane Doe", "balance": 2000 }
  ]
}
```

### Transfer

```http
POST /transfer
Content-Type: application/json
```

**Request body**

```json
{
  "from_id": "A1",
  "to_id": "A2",
  "amount": 100,
  "idempotency_key": "unique-key-001"
}
```

**Response (200)**

```json
{
  "from_id": "A1",
  "to_id": "A2",
  "amount": 100,
  "idempotency_key": "unique-key-001"
}
```

**Common errors**

| Error message                         | Cause                                      |
| ------------------------------------- | ------------------------------------------ |
| `from_id and to_id cannot be the same`| Source and destination are identical       |
| `account not found`                   | Unknown account or missing balance key     |
| `insufficient balance`                | Source balance lower than `amount`         |

### Batch transfer

```http
POST /batch-transfer
Content-Type: application/json
```

**Request body** — array of transfer requests (same shape as single transfer). Items without `idempotency_key` get an auto-generated key (`batch-{index}-{uuid}`).

```json
[
  {
    "from_id": "A1",
    "to_id": "A2",
    "amount": 50,
    "idempotency_key": "batch-1"
  },
  {
    "from_id": "A3",
    "to_id": "A4",
    "amount": 100,
    "idempotency_key": "batch-2"
  }
]
```

**Response (200)**

```json
{
  "success": 2,
  "failed": 0,
  "details": [
    {
      "idempotency_key": "batch-1",
      "status": "success",
      "message": "transfer OK"
    },
    {
      "idempotency_key": "batch-2",
      "status": "success",
      "message": "transfer OK"
    }
  ]
}
```

`status` is either `success` or `failed`; `message` contains `transfer OK` or the error text.

## Example: cURL

```bash
# List balances
curl -s http://localhost:8000/accounts | jq

# Transfer 100 from A1 to A2
curl -s -X POST http://localhost:8000/transfer \
  -H "Content-Type: application/json" \
  -d '{
    "from_id": "A1",
    "to_id": "A2",
    "amount": 100,
    "idempotency_key": "demo-transfer-1"
  }' | jq

# Repeat the same key — idempotent, no double debit
curl -s -X POST http://localhost:8000/transfer \
  -H "Content-Type: application/json" \
  -d '{
    "from_id": "A1",
    "to_id": "A2",
    "amount": 100,
    "idempotency_key": "demo-transfer-1"
  }' | jq
```

## Seeded accounts

| ID  | Name      | Initial balance |
| --- | --------- | --------------- |
| A1  | John Doe  | 1000            |
| A2  | Jane Doe  | 2000            |
| A3  | Jim Doe   | 3000            |
| A4  | Jill Doe  | 4000            |
| A5  | Jack Doe  | 5000            |

Balances in Redis may differ after transfers. Keys use the pattern `balance:{account_id}`; idempotency results use `idempotency:{key}`.

## Architecture

```
HTTP (Fiber)
    └── controller
            └── service
                    ├── lock.AccountLocker   (in-process per account)
                    └── repository
                            └── Redis (Lua: atomic transfer + idempotency)
```

**Transfer flow (simplified)**

1. Reject if `from_id == to_id`
2. Verify destination exists; load source balance
3. Fail fast if balance &lt; amount
4. Return cached response if idempotency key already exists
5. Lock both accounts (sorted order to avoid deadlock)
6. Run `ApplyTransfer` with retries on transient errors
7. Unlock accounts

## Project structure

```
.
├── controller/          # HTTP handlers
├── service/             # Business logic (transfer, batch, accounts)
├── repository/          # Redis access + mocks
├── model/               # Request/response DTOs
├── internal/
│   ├── cache/           # Redis client
│   ├── config/          # Environment loading
│   ├── lock/            # Account mutex locker
│   └── logging/         # Structured logger
├── docs/                # Generated Swagger (swag)
├── main.go
└── docker-compose.yml   # Redis only
```

## Testing

```bash
# All packages
go test ./...

# Verbose
go test ./... -v

# Service tests only
go test ./service/... -v
```

Tests use **testify** for assertions and **gomock** for repository/logger interfaces. Regenerate mocks after interface changes:

```bash
go run go.uber.org/mock/mockgen@latest \
  -source=repository/transaction_repository.go \
  -destination=repository/transaction_repository_mock.go \
  -package=repository TransactionRepository

go run go.uber.org/mock/mockgen@latest \
  -source=repository/seed_user_repository.go \
  -destination=repository/seed_user_repository_mock.go \
  -package=repository SeedUserRepository

go run go.uber.org/mock/mockgen@latest \
  -source=internal/logging/logger.go \
  -destination=internal/logging/logger_mock.go \
  -package=logging Logger
```

## Regenerate Swagger

After changing handler comments in `main.go` or `controller/`:

```bash
go install github.com/swaggo/swag/cmd/swag@latest
swag init
```

## License

Apache 2.0 — see [LICENSE](LICENSE).

## Contact

Armada Muhammad — armadamuhammads@gmail.com
