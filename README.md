# Cita Kita Transfer Service

Go API for single and batch transfers with **atomic Redis Lua updates**, **client idempotency keys**, in-process **account locking**, and **retry with backoff**.

## Assumptions

Critical design decisions for correctness and scope:

1. **Redis Lua is the source of truth** — Debit, credit, and idempotency are applied in one atomic script. Correctness across instances depends on Redis, not the app layer.
2. **Client-owned idempotency keys** — Callers must send a unique `idempotency_key` per transfer. A duplicate key returns the stored result and must not debit twice.
3. **In-process lock is not distributed** — `AccountLocker` only reduces races inside one process. Multiple replicas rely on Redis atomicity, not shared mutexes.
4. **Optimistic balance check** — The service checks balance before apply; under concurrency, Lua may still reject with `insufficient balance`.

## What's inside

| Layer        | Choice |
| ------------ | ------ |
| HTTP         | Fiber v2 |
| Storage      | Redis (go-redis v9), Lua script for transfers |
| API docs     | Swagger at `/swagger/index.html` |
| Go           | 1.26+ |

```
HTTP (Fiber) → controller → service → lock.AccountLocker
                              └── repository → Redis (Lua)
```

```
.
├── controller/     # HTTP handlers
├── service/        # Transfer, batch, accounts
├── repository/     # Redis + mocks
├── model/          # DTOs
├── internal/       # cache, config, lock, logging
├── docs/           # Generated Swagger
├── main.go
└── docker-compose.yml   # Redis only
```

**Transfer flow:** validate accounts → balance check → idempotency cache hit → lock accounts (sorted) → `ApplyTransfer` with retries → unlock.

## Prerequisites

- Go 1.26+
- Redis 7+ (`docker compose up -d` for local)

## Quick start

```bash
cp .env.example .env   # see file for PORT, REDIS_*, MAX_RETRIES, etc.
docker compose up -d
go run .
```

Startup seeds five demo accounts (A1–A5) via `SETNX`; existing Redis keys are not overwritten.

| ID  | Name      | Initial balance |
| --- | --------- | --------------- |
| A1  | John Doe  | 1000            |
| A2  | Jane Doe  | 2000            |
| A3  | Jim Doe   | 3000            |
| A4  | Jill Doe  | 4000            |
| A5  | Jack Doe  | 5000            |

Redis keys: `balance:{account_id}`, `idempotency:{key}`.

## API

| Method | Path              | Description |
| ------ | ----------------- | ----------- |
| GET    | `/health`         | Liveness (`OK`) |
| GET    | `/accounts`       | List users and balances |
| POST   | `/transfer`       | Single transfer; requires `idempotency_key` |
| POST   | `/batch-transfer` | Array of transfers; missing keys get `batch-{index}-{uuid}` |

**Transfer body:** `from_id`, `to_id`, `amount`, `idempotency_key`. Common errors: same account, `account not found`, `insufficient balance`.

```bash
curl http://localhost:8000/health
curl -s http://localhost:8000/accounts | jq
curl -s -X POST http://localhost:8000/transfer \
  -H "Content-Type: application/json" \
  -d '{"from_id":"A1","to_id":"A2","amount":100,"idempotency_key":"demo-1"}' | jq
```

Full request/response schemas: [Swagger UI](http://localhost:8000/swagger/index.html).

## Testing

```bash
go test ./...
```

Uses testify and gomock; regenerate mocks with `go generate ./...`. Swagger: `swag init` after handler comment changes.

## License

Apache 2.0 — see [LICENSE](LICENSE).
