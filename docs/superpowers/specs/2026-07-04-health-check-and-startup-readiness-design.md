# Health Check Endpoint & Startup Readiness Gate

Date: 2026-07-04
Status: Approved (pending implementation)

## Problem

1. The HTTP health endpoint `/api/health/test` (`controllers/userhandler.go:30`) returns a
static `{"status":"ok"}` and does not verify that RabbitMQ, Kafka, MySQL, or Redis are
reachable. Docker Compose uses this endpoint via `curl` to mark the container healthy, so a
green health check does not mean the backing services are up.
2. On startup, `Base.Init()` (`controllers/base.go:61`) calls `os.Exit(1)` on the first
connection failure and never retries. In `docker-compose` the four backing services come up
at different times, so the app frequently exits before its dependencies are ready.
3. `utils.NewRabbitMQConnection` (`pkg/utils/queue.go:123`) calls `log.Fatalf` on failure,
which terminates the whole process and makes any retry loop impossible.
4. Two unrelated infra bugs block the `curl`-based healthcheck:
   - `Dockerfile:28` runs `apk add` on a Debian (`debian:bookworm-slim`) base image.
   - `docker-compose.yml:18` uses `${PORT:7001}`, which Compose parses as a substring offset,
     not a default value. It must be `${PORT:-7001}`.

## Goals

- A health endpoint that probes MySQL, Redis, and (when enabled) RabbitMQ and Kafka, and
  reports their status.
- A startup readiness gate that waits (with retry/backoff) until all *enabled* dependencies
  are reachable before the HTTP servers start.
- Unit tests for both behaviors.
- Restore the `curl`-based Docker Compose healthcheck.

## Non-Goals

- No changes to publish/consume logic, routes, services, or repo layers.
- No new third-party dependencies.
- No persistence/monitoring of health history.

## Decisions

- **Startup behavior when a dependency is down:** Wait with retry/backoff until all enabled
  deps are up; exit only after a total timeout.
- **Optional deps:** Respect the existing `On` flags (`KafkaStatus`, `RabbitMQStatus`,
  `on=1` means enabled). MySQL and Redis are always required. Disabled deps are reported as
  `disabled` by the health endpoint and skipped by the startup gate.

## Architecture

A new `pkg/health` package provides the reusable core. It is used in two places:

1. **Startup readiness** — `health.Await` over *probe* checkers built from config-only
   connections, run inside `Init()` before the heavy/persistent connect steps.
2. **Runtime health endpoint** — `health.Check` over *live-handle* checkers built from the
   persistent handles stored on `Base` (`DB`, `Redis`, `KafkaProducer`, `rabbitURL`).

Both share the same `Checker` shape, so probe and live-handle checkers are interchangeable
with `health.Check`/`health.Await`.

## Components

### `pkg/health` (new package)

```go
type Checker struct {
    Name     string
    Disabled bool
    Ping     func(context.Context) error
}

type Result struct {
    Name   string
    Status string // "up" | "down" | "disabled"
    Error  string // "" when up or disabled
}

type Report struct {
    Status string   // "healthy" | "unhealthy"
    Checks []Result
}

// Check runs each checker's Ping (bounded by ctx) and returns a Report.
// Overall Status is "healthy" iff every enabled checker is "up".
func Check(ctx context.Context, checkers []Checker) Report

// Await calls Check every interval until all enabled checkers are up or
// timeout elapses. Returns nil on success; returns an error wrapping the
// final Report on timeout.
func Await(ctx context.Context, checkers []Checker, interval, timeout time.Duration) error
```

Behavior details:
- `Check` derives a per-checker timeout from `ctx` (default 2s). A `Ping` returning an
  error marks the checker `down`; a panic-free contract is maintained by not sharing mutable
  broker channels.
- `disabled` checkers are not executed and never cause `unhealthy`.
- `Await` samples at `interval` (default 2s), bounded by `timeout` (default 60s). It does not
  sleep past `timeout`; the last failure's `Report` becomes the returned error's text.

### Live-handle checkers (built from `Base`)

The endpoint builds checkers from `Base`'s persistent handles:

- **MySQL** — `func(ctx) error { return b.DB.PingContext(ctx) }`
- **Redis** — `func(ctx) error { return b.Redis.Ping(ctx).Err() }`
- **RabbitMQ** (`RabbitMQStatus == 1`) — dial a fresh `amqp.Dial` against `b.rabbitURL` with a
  short dial timeout, then close. Using `b.rabbitURL` (new `Base` field) avoids touching the
  shared channel concurrently.
- **Kafka** (`KafkaStatus == 1`) — `kafka.NewAdminClientFromProducer(b.KafkaProducer)` then
  `GetMetadata(&kafka.GetMetadataArgs{...}, true, <timeout>)`, which is a real broker round-trip.

Disabled RabbitMQ/Kafka are emitted as `disabled`.

`Base` gains one new unexported field:

```go
rabbitURL string
```

captured in `Init()` (the same URL string already computed at `base.go:213-227`).

### Startup readiness gate

In `Init()`, after config is parsed and **before** the existing heavy connect steps:

- Build probe checkers from config-only connections:
  - **MySQL** — `sql.Open(dsn)` + `PingContext`, close afterward.
  - **Redis** — `redis.NewClient(opts)` + `Ping`, close afterward.
  - **RabbitMQ** (if `on==1`) — `amqp.Dial` against the configured URL with a short dial
    timeout, then close.
  - **Kafka** (if `on==1`) — `kafka.NewAdminClient` from an ephemeral producer-less admin
    client (`kafka.NewAdminClient(&kafka.ConfigMap{"bootstrap.servers": broker})`) +
    `GetMetadata`. Close after.
- `health.Await(ctx, probes, 2*time.Second, startupTimeout)` where `startupTimeout` defaults
  to 60s and is overridable via `STARTUP_DEPENDENCY_TIMEOUT` env var (parsed as a `time.Duration`
  string; invalid/empty falls back to 60s).
- On success: proceed to the existing connect steps (which now succeed quickly).
- On timeout: `utils.LogError` with the final report and `os.Exit(1)`.

### Refactor: `utils.NewRabbitMQConnection`

Remove the two `log.Fatalf` calls (`queue.go:138`, `queue.go:146`); keep returning the error.
The callers already log and decide exit behavior. This is required so the startup retry loop
and any future retry can observe the error rather than have the process killed.

### HTTP endpoint `/api/health/test`

Rewrite the handler in `controllers/userhandler.go:30`:

- Build live-handle checkers from `Base` via an overridable seam: a new unexported field

  ```go
  checkersProvider func() []health.Checker
  ```

  `nil` in production → a default builder that reads `b.DB`, `b.Redis`, `b.KafkaProducer`,
  `b.rabbitURL`, `b.KafkaStatus`, `b.RabbitMQStatus`. In tests, the harness injects synthetic
  checkers (closures with counters) so the endpoint is testable without real brokers.

- Call `health.Check(ctx, checkers)`.
- Response (JSON):
  - all enabled `up`: `200` + `{"status":"healthy","checks":[...]}`
  - any enabled `down`: `503` + `{"status":"unhealthy","checks":[...]}`
- Update the swagger annotation: remove the bogus `@Param payload body ...` (this is a GET).

### Dockerfile

Replace `Dockerfile:28`:

```dockerfile
RUN apt-get update \
 && apt-get install -y --no-install-recommends curl \
 && rm -rf /var/lib/apt/lists/*
```

### docker-compose.yml

Fix `docker-compose.yml:18`:

```yaml
["CMD-SHELL", "curl -fsS http://127.0.0.1:${PORT:-7001}/api/health/test"]
```

## Testing

### `pkg/health/health_test.go`

- `TestCheck_AllUp` — every Ping returns nil → `Status == "healthy"`, all `up`.
- `TestCheck_OneDown` — one Ping returns an error → `Status == "unhealthy"`, that checker
  `down` with `Error` set, others `up`.
- `TestCheck_Disabled` — a `Disabled` checker is reported `disabled`, not executed, and does
  not affect overall status.
- `TestAwait_SuccessAfterRetries` — a Ping flips to nil after N calls (counter-based) → `Await`
  returns nil once all up, before timeout.
- `TestAwait_Timeout` — a Ping always errors with a short timeout → `Await` returns a non-nil
  error whose message references the failing checker(s).

### `controllers/userhandler_test.go`

- Helper: `newHealthBase(t, checkers []health.Checker) *Base` sets `checkersProvider`.
- `TestHealthCheck_AllUp` → 200, JSON `status == "healthy"`.
- `TestHealthCheck_OneDown` → 503, JSON `status == "unhealthy"`, failing checker present.
- `TestHealthCheck_DisabledDeps` → Kafka/Rabbit `disabled`, MySQL/Redis `up`, overall
  `healthy`, status 200.

Tests call the handler directly via `httptest` (consistent with the existing
`bookinghandler_test.go` pattern).

## Risk & Verification

- **Probe vs. live-handle divergence:** Both use the same `Checker`/`Check` core, so behavior
  is identical by construction. Differences are only in *what* the Ping does (ephemeral dial in
  startup probes, live-handle ping in the endpoint).
- **Kafka reachability probe cost:** `GetMetadata` is a single broker round-trip with a 2s
  timeout; acceptable for startup and a 30s-interval healthcheck.
- **No `log.Fatalf` after refactor:** `NewRabbitMQConnection` will only return errors; the only
  process-exit paths remain explicit `os.Exit(1)` calls in `Init()` and `main`, preserving
  current crash semantics except where retry is intended.
- **Verification command:** `go build ./...`, `go test ./pkg/health/... ./controllers/...`,
  and `golangci-lint run`.
