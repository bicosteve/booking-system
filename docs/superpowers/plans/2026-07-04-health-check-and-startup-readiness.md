# Health Check Endpoint & Startup Readiness Gate — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the `/api/health/test` endpoint report per-dependency reachability for MySQL, Redis, RabbitMQ, and Kafka, and gate app startup on enabled dependencies being reachable.

**Architecture:** A new testable `pkg/health` package (a `Checker` with a `Ping` closure plus `Check`/`Await`) is the shared core. Startup uses config-only probe checkers; the HTTP endpoint uses live-handle checkers built from `Base`. Tests inject synthetic checkers, so no real brokers or DBs are required to test the endpoint.

**Tech Stack:** Go 1.25, `database/sql`, `redis/go-redis/v9`, `streadway/amqp`, `confluent-kafka-go/v2`, `chi/v5`, `stretchr/testify`, `sqlmock`, `redismock`.

## Global Constraints

- Go toolchain: `go 1.25.0` with `toolchain go1.25.11` (already in `go.mod`). All code must compile with this toolchain.
- No new third-party dependencies. Use only modules already in `go.mod`.
- Honor the existing `On` flags: `KafkaStatus`/`RabbitMQStatus` (`1` = enabled). MySQL and Redis are always required.
- Disabled dependencies are reported as `disabled` and never cause `unhealthy`.
- **Formatting:** After creating or editing ANY `.go` file in a task, you MUST run `gofmt -w <file>` (or `gofmt -w .` from the repo root) before running tests or committing. The plan's Go snippets may render with flattened indentation in this doc; `gofmt -w` normalizes them so they compile and pass CI. The code-quality job runs golangci-lint with the `gofmt` linter enabled, so un-formatted files fail CI.
- Confluent Kafka cgo build needs `librdkafka-dev` (installed in CI; present on the dev machine — `go build ./...` already passes).
- Do not change publish/consume logic, routes, services, or repo layers.
- Commit after every task.

## File Structure

- Create `pkg/health/health.go` — `Checker`, `Result`, `Report`, `Check`, `Await`.
- Create `pkg/health/health_test.go` — tests for `Check` and `Await`.
- Create `pkg/health/probes.go` — `MySQLProbe`, `RedisProbe`, `RabbitProbe`, `KafkaProbe`.
- Create `pkg/health/probes_test.go` — probe tests against closed ports (expect errors).
- Modify `pkg/utils/queue.go` — remove `log.Fatalf` from `NewRabbitMQConnection`.
- Create `pkg/utils/queue_test.go` — prove it returns an error instead of killing the process.
- Modify `controllers/base.go` — add `rabbitURL` and `checkersProvider` fields; capture `rabbitURL`; call the startup gate.
- Create `controllers/startup.go` — `waitForDependencies`, `startupTimeout`, `buildStartupProbes`, `rabbitURL`.
- Create `controllers/health.go` — `defaultLiveCheckers`, `healthCheckers`.
- Modify `controllers/userhandler.go` — rewrite `HealthCheck` to use `health.Check`.
- Create `controllers/health_test.go` — endpoint tests with injected checkers.
- Modify `Dockerfile` — install `curl` via `apt-get` (Debian base).
- Modify `docker-compose.yml` — fix `${PORT:-7001}` default syntax.

## Confluent Kafka API reference (installed v2.6.1)

```go
func NewAdminClient(conf *kafka.ConfigMap) (*kafka.AdminClient, error)
func (a *kafka.AdminClient) GetMetadata(topic *string, allTopics bool, timeoutMs int) (*kafka.Metadata, error)
func (a *kafka.AdminClient) Close()
func NewAdminClientFromProducer(p *kafka.Producer) (a *AdminClient, err error)
```

## amqp API reference (installed v1.1.0)

```go
func amqp.Dial(url string) (*amqp.Connection, error)
func (c *amqp.Connection) Close() error
```

---

### Task 1: `pkg/health` core — `Check`

**Files:**
- Create: `pkg/health/health.go`
- Create: `pkg/health/health_test.go`

**Interfaces:**
- Produces: `type Checker struct { Name string; Disabled bool; Ping func(context.Context) error }`, `type Result struct { Name, Status, Error string }`, `type Report struct { Status string; Checks []Result }`, `func Check(ctx context.Context, checkers []Checker) Report`.

- [ ] **Step 1: Write the failing tests**

Create `pkg/health/health_test.go`:

```go
package health

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCheck_AllUp(t *testing.T) {
	checkers := []Checker{
		{Name: "mysql", Ping: func(context.Context) error { return nil }},
		{Name: "redis", Ping: func(context.Context) error { return nil }},
		{Name: "rabbitmq", Ping: func(context.Context) error { return nil }},
		{Name: "kafka", Ping: func(context.Context) error { return nil }},
	}
	r := Check(context.Background(), checkers)
	assert.Equal(t, "healthy", r.Status)
	for _, c := range r.Checks {
		assert.Equal(t, "up", c.Status, c.Name)
		assert.Empty(t, c.Error)
	}
}

func TestCheck_OneDown(t *testing.T) {
	checkers := []Checker{
		{Name: "mysql", Ping: func(context.Context) error { return nil }},
		{Name: "redis", Ping: func(context.Context) error { return errors.New("redis down") }},
		{Name: "rabbitmq", Ping: func(context.Context) error { return nil }},
		{Name: "kafka", Ping: func(context.Context) error { return nil }},
	}
	r := Check(context.Background(), checkers)
	assert.Equal(t, "unhealthy", r.Status)
	var redisResult *Result
	for i := range r.Checks {
		if r.Checks[i].Name == "redis" {
			redisResult = &r.Checks[i]
		}
	}
	if assert.NotNil(t, redisResult) {
		assert.Equal(t, "down", redisResult.Status)
		assert.Contains(t, redisResult.Error, "redis down")
	}
}

func TestCheck_Disabled(t *testing.T) {
	called := false
	checkers := []Checker{
		{Name: "mysql", Ping: func(context.Context) error { return nil }},
		{Name: "redis", Ping: func(context.Context) error { return nil }},
		{Name: "rabbitmq", Disabled: true, Ping: func(context.Context) error { called = true; return nil }},
		{Name: "kafka", Disabled: true, Ping: func(context.Context) error { called = true; return nil }},
	}
	r := Check(context.Background(), checkers)
	assert.Equal(t, "healthy", r.Status)
	assert.False(t, called, "disabled checkers must not be executed")
	for _, c := range r.Checks {
		if c.Name == "rabbitmq" || c.Name == "kafka" {
			assert.Equal(t, "disabled", c.Status, c.Name)
		} else {
			assert.Equal(t, "up", c.Status, c.Name)
		}
	}
}

func TestCheck_NilPing(t *testing.T) {
	checkers := []Checker{{Name: "mysql", Ping: nil}}
	r := Check(context.Background(), checkers)
	assert.Equal(t, "unhealthy", r.Status)
	assert.Equal(t, "down", r.Checks[0].Status)
	assert.Contains(t, r.Checks[0].Error, "ping function not configured")
}

// used by Await tests; keep here to avoid unused warnings before Await exists
var _ = sync.WaitGroup{}

func unusedTime() time.Duration { return time.Millisecond }
```

> Note: the `sync` and `time` imports are referenced by `unusedTime`/`var _` so this file compiles in Task 1 (Await + its tests arrive in Task 2). They will be used properly in Task 2.

- [ ] **Step 2: Run the tests to verify they fail**

Run: `GOTOOLCHAIN=auto go test ./pkg/health/ -run TestCheck -v`
Expected: build failure / FAIL — package `health` has no `Checker`/`Check` symbols.

- [ ] **Step 3: Write the minimal implementation**

Create `pkg/health/health.go`:

```go
package health

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Checker performs a single dependency reachability probe.
type Checker struct {
	// Name identifies the dependency in the report (e.g. "mysql").
	Name string
	// Disabled marks the dependency as intentionally off. Disabled checkers
	// are reported as "disabled" and never affect overall health.
	Disabled bool
	// Ping is invoked with a context carrying a per-checker timeout. A nil
	// Ping is treated as an unreachable dependency.
	Ping func(context.Context) error
}

// Result is the outcome of a single checker.
type Result struct {
	Name   string `json:"name"`
	Status string `json:"status"` // "up" | "down" | "disabled"
	Error  string `json:"error,omitempty"`
}

// Report aggregates checker results.
type Report struct {
	Status string   `json:"status"` // "healthy" | "unhealthy"
	Checks []Result `json:"checks"`
}

// Check runs each checker's Ping (bounded by a per-checker 2s timeout derived
// from ctx) and returns a Report. Overall Status is "healthy" iff every
// enabled checker is "up". Disabled checkers are not executed.
func Check(ctx context.Context, checkers []Checker) Report {
	report := Report{Checks: make([]Result, 0, len(checkers))}
	var mu sync.Mutex // guard report.Checks if Pings run concurrently
	healthy := true
	_ = mu

	for _, c := range checkers {
		if c.Disabled {
			report.Checks = append(report.Checks, Result{Name: c.Name, Status: "disabled"})
			continue
		}
		status := "up"
		errMsg := ""
		if c.Ping == nil {
			status = "down"
			errMsg = "ping function not configured"
			healthy = false
		} else {
			pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
			err := c.Ping(pingCtx)
			cancel()
			if err != nil {
				status = "down"
				errMsg = err.Error()
				healthy = false
			}
		}
		report.Checks = append(report.Checks, Result{Name: c.Name, Status: status, Error: errMsg})
	}

	if healthy {
		report.Status = "healthy"
	} else {
		report.Status = "unhealthy"
	}
	return report
}

// Await is documented here for completeness and implemented in Task 2.
var _ = fmt.Sprintf
var _ = strings.Join
```

> The `fmt`/`strings`/`sync` imports are referenced so Task 1 compiles; `Await` (Task 2) consumes `fmt`/`strings`. Keep `sync` too — it's fine for now.

- [ ] **Step 4: Run the tests to verify they pass**

Run: `GOTOOLCHAIN=auto go test ./pkg/health/ -run TestCheck -v`
Expected: PASS — `TestCheck_AllUp`, `TestCheck_OneDown`, `TestCheck_Disabled`, `TestCheck_NilPing` all pass.

- [ ] **Step 5: Commit**

```bash
git add pkg/health/health.go pkg/health/health_test.go
git commit -m "feat(health): add Checker/Check with per-dependency status reporting"
```

---

### Task 2: `pkg/health` — `Await` (startup gate core)

**Files:**
- Modify: `pkg/health/health.go`
- Modify: `pkg/health/health_test.go`

**Interfaces:**
- Consumes: `Checker`, `Check` from Task 1.
- Produces: `func Await(ctx context.Context, checkers []Checker, interval, timeout time.Duration) error`.

- [ ] **Step 1: Write the failing tests**

Append to `pkg/health/health_test.go` (and remove the `unusedTime` placeholder from Task 1):

```go
func TestAwait_SuccessAfterRetries(t *testing.T) {
	var mu sync.Mutex
	calls := 0
	c := Checker{Name: "mysql", Ping: func(context.Context) error {
		mu.Lock()
		defer mu.Unlock()
		calls++
		if calls < 3 {
			return errors.New("not ready")
		}
		return nil
	}}
	err := Await(context.Background(), []Checker{c}, 10*time.Millisecond, 2*time.Second)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, calls, 3)
}

func TestAwait_Timeout(t *testing.T) {
	c := Checker{Name: "mysql", Ping: func(context.Context) error { return errors.New("nope") }}
	err := Await(context.Background(), []Checker{c}, 10*time.Millisecond, 50*time.Millisecond)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not ready after")
	assert.Contains(t, err.Error(), "mysql")
}

func TestAwait_CancelledContext(t *testing.T) {
	c := Checker{Name: "mysql", Ping: func(context.Context) error { return errors.New("nope") }}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := Await(ctx, []Checker{c}, 10*time.Millisecond, 200*time.Millisecond)
	assert.Error(t, err)
}

func TestAwait_EmptyCheckers(t *testing.T) {
	err := Await(context.Background(), nil, 10*time.Millisecond, 100*time.Millisecond)
	assert.NoError(t, err)
}
```

Also delete the Task-1 placeholders `var _ = sync.WaitGroup{}` and `func unusedTime ...` now that `sync`/`time` are used by real tests.

- [ ] **Step 2: Run the tests to verify they fail**

Run: `GOTOOLCHAIN=auto go test ./pkg/health/ -run TestAwait -v`
Expected: FAIL — `Await` undefined.

- [ ] **Step 3: Write the implementation**

Replace the tail of `pkg/health/health.go` (the `// Await is documented...` placeholder block and its `var _` lines) with:

```go
// Await calls Check every interval until all enabled checkers are up or
// timeout elapses. Returns nil once healthy; on timeout returns an error
// summarizing the failing checkers. Respects ctx cancellation.
func Await(ctx context.Context, checkers []Checker, interval, timeout time.Duration) error {
	if len(checkers) == 0 {
		return nil
	}
	deadline := time.Now().Add(timeout)
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		last := Check(ctx, checkers)
		if last.Status == "healthy" {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("dependencies not ready after %s: %s", timeout, summarize(last))
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(interval):
		}
	}
}

func summarize(r Report) string {
	var downs []string
	for _, c := range r.Checks {
		if c.Status == "down" {
			downs = append(downs, c.Name)
		}
	}
	if len(downs) == 0 {
		return "no details"
	}
	return "down: " + strings.Join(downs, ", ")
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `GOTOOLCHAIN=auto go test ./pkg/health/ -v`
Expected: PASS — all `TestCheck_*` and `TestAwait_*` tests pass.

- [ ] **Step 5: Commit**

```bash
git add pkg/health/health.go pkg/health/health_test.go
git commit -m "feat(health): add Await retry loop for startup readiness"
```

---

### Task 3: `pkg/health` — config-only probe builders

**Files:**
- Create: `pkg/health/probes.go`
- Create: `pkg/health/probes_test.go`

**Interfaces:**
- Consumes: `Checker` from Task 1.
- Produces: `func MySQLProbe(dsn string) Checker`, `func RedisProbe(addr string) Checker`, `func RabbitProbe(url string) Checker`, `func KafkaProbe(broker string) Checker`.

- [ ] **Step 1: Write the failing tests**

Create `pkg/health/probes_test.go`:

```go
package health

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func probeDown(t *testing.T, c Checker) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := c.Ping(ctx)
	assert.Error(t, err, c.Name)
}

func TestMySQLProbe_BadAddress(t *testing.T) {
	probeDown(t, MySQLProbe("root:root@tcp(127.0.0.1:1)/nodb?timeout=200ms"))
	assert.Equal(t, "mysql", MySQLProbe("").Name)
}

func TestRedisProbe_BadAddress(t *testing.T) {
	probeDown(t, RedisProbe("127.0.0.1:1"))
	assert.Equal(t, "redis", RedisProbe("").Name)
}

func TestRabbitProbe_BadAddress(t *testing.T) {
	probeDown(t, RabbitProbe("amqp://guest:guest@127.0.0.1:1/"))
	assert.Equal(t, "rabbitmq", RabbitProbe("").Name)
}

func TestKafkaProbe_BadAddress(t *testing.T) {
	probeDown(t, KafkaProbe("127.0.0.1:1"))
	assert.Equal(t, "kafka", KafkaProbe("").Name)
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `GOTOOLCHAIN=auto go test ./pkg/health/ -run Probe -v`
Expected: FAIL — `MySQLProbe`/`RedisProbe`/`RabbitProbe`/`KafkaProbe` undefined.

- [ ] **Step 3: Write the implementation**

Create `pkg/health/probes.go`:

```go
package health

import (
	"context"
	"database/sql"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/redis/go-redis/v9"
	"github.com/streadway/amqp"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// MySQLProbe opens a throwaway *sql.DB against dsn and pings it.
func MySQLProbe(dsn string) Checker {
	return Checker{
		Name: "mysql",
		Ping: func(ctx context.Context) error {
			db, err := sql.Open("mysql", dsn)
			if err != nil {
				return err
			}
			defer db.Close()
			return db.PingContext(ctx)
		},
	}
}

// RedisProbe creates a throwaway redis client against addr (host:port) and pings it.
func RedisProbe(addr string) Checker {
	return Checker{
		Name: "redis",
		Ping: func(ctx context.Context) error {
			cli := redis.NewClient(&redis.Options{Addr: addr})
			defer cli.Close()
			return cli.Ping(ctx).Err()
		},
	}
}

// RabbitProbe dials the provided amqp URL once and closes the connection.
// Honors ctx cancellation so it never blocks longer than the caller's deadline.
func RabbitProbe(url string) Checker {
	return Checker{
		Name: "rabbitmq",
		Ping: func(ctx context.Context) error {
			done := make(chan error, 1)
			go func() {
				conn, err := amqp.Dial(url)
				if err != nil {
					done <- err
					return
				}
				done <- conn.Close()
			}()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case err := <-done:
				return err
			}
		},
	}
}

// KafkaProbe uses an ephemeral AdminClient to fetch cluster metadata, a real
// broker round-trip. The metadata timeout is capped by ctx's deadline.
func KafkaProbe(broker string) Checker {
	return Checker{
		Name: "kafka",
		Ping: func(ctx context.Context) error {
			ac, err := kafka.NewAdminClient(&kafka.ConfigMap{"bootstrap.servers": broker})
			if err != nil {
				return err
			}
			defer ac.Close()
			timeoutMs := 1000
			if dl, ok := ctx.Deadline(); ok {
				if d := time.Until(dl).Milliseconds(); d > 0 && d < int64(timeoutMs) {
					timeoutMs = int(d)
				}
			}
			_, err = ac.GetMetadata(nil, true, timeoutMs)
			return err
		},
	}
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `GOTOOLCHAIN=auto go test ./pkg/health/ -run Probe -v`
Expected: PASS — each probe returns a non-nil error for a closed port.

- [ ] **Step 5: Commit**

```bash
git add pkg/health/probes.go pkg/health/probes_test.go
git commit -m "feat(health): add config-only probe builders for mysql/redis/rabbit/kafka"
```

---

### Task 4: Remove `log.Fatalf` from `NewRabbitMQConnection`

**Files:**
- Modify: `pkg/utils/queue.go:2-16,136-148`
- Create: `pkg/utils/queue_test.go`

**Interfaces:**
- Consumes: nothing new.
- Produces: `NewRabbitMQConnection` returns an error on failure instead of calling `log.Fatalf` (process-exit). Required so the startup retry loop (Task 7) can observe failures.

- [ ] **Step 1: Write the failing test**

Create `pkg/utils/queue_test.go`:

```go
package utils

import "testing"

// NewRabbitMQConnection must return an error for an unreachable broker, not
// call log.Fatalf (which would os.Exit and abort the test binary).
func TestNewRabbitMQConnection_InvalidURL(t *testing.T) {
	conn, err := NewRabbitMQConnection("amqp://guest:guest@127.0.0.1:1/")
	if conn != nil {
		_ = conn.Close()
	}
	if err == nil {
		t.Fatal("expected error for invalid rabbitmq url, got nil")
	}
}
```

- [ ] **Step 2: Run the test to verify the current behavior fails**

Run: `GOTOOLCHAIN=auto go test ./pkg/utils/ -run TestNewRabbitMQConnection -v`
Expected: either the test binary exits via `log.Fatalf` (proving the bug), or a hang/race. Either way this is the failing baseline.

- [ ] **Step 3: Remove the `log.Fatalf` calls**

In `pkg/utils/queue.go`:

Change the import block (lines 3-16) to remove `"log"`:

```go
import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/bicosteve/booking-system/entities"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/streadway/amqp"
)
```

In `NewRabbitMQConnection` (around lines 135-148), replace the body so it returns the error without `log.Fatalf`:

```go
	conn, err := amqp.Dial(qURI)
	if err != nil {
		LogError("RABBITMQ: Failed to connect due to: %s", entities.ErrorLog, err)
		return nil, err
	}

	// 2. Open a rabbitmq channel
	ch, err := conn.Channel()
	if err != nil {
		LogError("RABBITMQ: Failed to open a channel : %s", entities.ErrorLog, err)
		return nil, err
	}
```

Leave the rest of `NewRabbitMQConnection` unchanged.

- [ ] **Step 4: Run the test to verify it passes**

Run: `GOTOOLCHAIN=auto go test ./pkg/utils/ -v`
Expected: PASS — `TestNewRabbitMQConnection_InvalidURL` returns an error; the test binary is not killed.

- [ ] **Step 5: Commit**

```bash
git add pkg/utils/queue.go pkg/utils/queue_test.go
git commit -m "fix(queue): return error from NewRabbitMQConnection instead of log.Fatalf"
```

---

### Task 5: `Base` fields + startup gate plumbing

**Files:**
- Modify: `controllers/base.go:54-58` (struct), `base.go:61` (Init), `base.go:201-228` (rabbit block)
- Create: `controllers/startup.go`

**Interfaces:**
- Consumes: `health.Await`, probes from Task 3; `entities.Config`, `entities.RabbitMQConfig`; `utils.LogError`, `entities.ErrorLog`.
- Produces: `Base.rabbitURL string`, `Base.checkersProvider func() []health.Checker`; `func (b *Base) waitForDependencies(cfg entities.Config)`; `func startupTimeout() time.Duration`; `func buildStartupProbes(cfg entities.Config) []health.Checker`; `func rabbitURL(rb entities.RabbitMQConfig) string` (package-level helper, also used by `base.go`).

- [ ] **Step 1: Add the new fields to `Base`**

In `controllers/base.go`, add two unexported fields to the `Base` struct (insert after line 58, near `rabbitConn`/`queueName`):

```go
	rabbitConn *amqp.Connection
	queueName string
	rabbitURL string
	// healthCheckers is overridden in tests; nil means use defaultLiveCheckers(). Used by HealthCheck.
	checkersProvider func() []health.Checker
	ctx context.Context
	KafkaStatus int
	RabbitMQStatus int
}
```

(Only the two new lines (`rabbitURL string` and the `checkersProvider` comment+field) are new; the surrounding lines are shown for placement.)

- [ ] **Step 2: Create `controllers/startup.go`**

```go
package controllers

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/bicosteve/booking-system/entities"
	"github.com/bicosteve/booking-system/pkg/health"
	"github.com/bicosteve/booking-system/pkg/utils"
)

// waitForDependencies blocks until all enabled dependencies are reachable or
// the startup timeout expires. On timeout it logs the failure and exits.
func (b *Base) waitForDependencies(cfg entities.Config) {
	checkers := buildStartupProbes(cfg, b)
	if len(checkers) == 0 {
		return
	}
	timeout := startupTimeout()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := health.Await(ctx, checkers, 2*time.Second, timeout); err != nil {
		utils.LogError(err.Error(), entities.ErrorLog)
		os.Exit(1)
	}
}

// startupTimeout reads STARTUP_DEPENDENCY_TIMEOUT; defaults to 60s.
func startupTimeout() time.Duration {
	if v := os.Getenv("STARTUP_DEPENDENCY_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			return d
		}
	}
	return 60 * time.Second
}

// buildStartupProbes returns config-only checkers for enabled dependencies.
// It also captures b.rabbitURL so the health endpoint (Task 6) can reuse it.
func buildStartupProbes(cfg entities.Config, b *Base) []health.Checker {
	var cs []health.Checker
	for _, m := range cfg.Mysql {
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=latin1&parseTime=True&loc=Local",
			m.Username, m.Password, m.Host, m.Port, m.Schema)
		cs = append(cs, health.MySQLProbe(dsn))
	}
	for _, r := range cfg.Redis {
		cs = append(cs, health.RedisProbe(r.Address+":"+r.Port))
	}
	for _, k := range cfg.Kafka {
		if k.On == 1 {
			cs = append(cs, health.KafkaProbe(k.Broker))
		}
	}
	for _, rb := range cfg.Rabbit {
		if rb.On == 1 {
			url := rabbitURL(rb)
			b.rabbitURL = url
			cs = append(cs, health.RabbitProbe(url))
		}
	}
	return cs
}

// rabbitURL builds the amqp URL. In prod the vhost is included; elsewhere it is omitted.
func rabbitURL(rb entities.RabbitMQConfig) string {
	if os.Getenv("ENV") == "prod" {
		return fmt.Sprintf("amqp://%s:%s@%s:%s/%s", rb.User, rb.Password, rb.Host, rb.Port, rb.Vhost)
	}
	return fmt.Sprintf("amqp://%s:%s@%s:%s", rb.User, rb.Password, rb.Host, rb.Port)
}
```

- [ ] **Step 3: Call the gate from `Init()` and reuse `rabbitURL`**

In `controllers/base.go`, inside `func (b *Base) Init()`, insert a call to the gate immediately after the logger init (after the existing `utils.InitLogger(...)` block, before the `for _, kafka := range config.Kafka` loop):

```go
	// Wait for backing services to be reachable before connecting, so the app
	// doesn't exit when docker-compose services come up at different times.
	b.waitForDependencies(config)
```

Then replace the rabbit connect block (the `if b.RabbitMQStatus == 1 && os.Getenv("ENV") == "prod" { ... } else { ... }` around lines 212-228) with a version that reuses `rabbitURL` and captures `b.rabbitURL`:

```go
	if b.RabbitMQStatus == 1 {
		url := rabbitURL(rabbitConf)
		b.rabbitURL = url
		conn, err := utils.NewRabbitMQConnection(url)
		if err != nil {
			utils.LogError(err.Error(), entities.ErrorLog)
			os.Exit(1)
		}
		b.rabbitConn = conn
	} else {
		url := rabbitURL(rabbitConf)
		conn, err := utils.NewRabbitMQConnection(url)
		if err != nil {
			utils.LogError(err.Error(), entities.ErrorLog)
			os.Exit(1)
		}
		b.rabbitConn = conn
	}
```

> This keeps existing connect-always behavior for the non-vhost path but removes the duplicated URL formatting. `rabbitConf` is already in scope from the earlier `for _, rabbitConf := range config.Rabbit` loop.

- [ ] **Step 4: Verify the package builds**

Run: `GOTOOLCHAIN=auto go build ./controllers/`
Expected: build succeeds (no test cycle here; tests arrive in Task 6).

- [ ] **Step 5: Commit**

```bash
git add controllers/base.go controllers/startup.go
git commit -m "feat(controllers): add startup dependency readiness gate to Init"
```

---

### Task 6: Health endpoint rewrite + tests

**Files:**
- Create: `controllers/health.go`
- Modify: `controllers/userhandler.go:1-34`
- Create: `controllers/health_test.go`

**Interfaces:**
- Consumes: `health.Check`, `health.Report`, `health.Checker`, `health.RabbitProbe`, `Base.KafkaProducer`, `Base.rabbitURL`, `Base.KafkaStatus`, `Base.RabbitMQStatus`, `Base.DB`, `Base.Redis`, `Base.checkersProvider`.
- Produces: rewrite of `func (b *Base) HealthCheck(w http.ResponseWriter, r *http.Request)` returning `{"status","checks"}` JSON.

- [ ] **Step 1: Write the failing tests**

Create `controllers/health_test.go`:

```go
package controllers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bicosteve/booking-system/pkg/health"
	"github.com/stretchr/testify/assert"
)

func newHealthBase(checkers []health.Checker) *Base {
	return &Base{checkersProvider: func() []health.Checker { return checkers }}
}

func TestHealthCheck_AllUp(t *testing.T) {
	base := newHealthBase([]health.Checker{
		{Name: "mysql", Ping: func(context.Context) error { return nil }},
		{Name: "redis", Ping: func(context.Context) error { return nil }},
		{Name: "rabbitmq", Ping: func(context.Context) error { return nil }},
		{Name: "kafka", Ping: func(context.Context) error { return nil }},
	})
	req := httptest.NewRequest(http.MethodGet, "/api/health/test", nil)
	w := httptest.NewRecorder()
	base.HealthCheck(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var rep health.Report
	assert.NoError(t, json.NewDecoder(w.Body).Decode(&rep))
	assert.Equal(t, "healthy", rep.Status)
	assert.Len(t, rep.Checks, 4)
}

func TestHealthCheck_OneDown(t *testing.T) {
	base := newHealthBase([]health.Checker{
		{Name: "mysql", Ping: func(context.Context) error { return nil }},
		{Name: "redis", Ping: func(context.Context) error { return errors.New("redis down") }},
		{Name: "rabbitmq", Ping: func(context.Context) error { return nil }},
		{Name: "kafka", Ping: func(context.Context) error { return nil }},
	})
	req := httptest.NewRequest(http.MethodGet, "/api/health/test", nil)
	w := httptest.NewRecorder()
	base.HealthCheck(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	var rep health.Report
	assert.NoError(t, json.NewDecoder(w.Body).Decode(&rep))
	assert.Equal(t, "unhealthy", rep.Status)
	var redisResult *health.Result
	for i := range rep.Checks {
		if rep.Checks[i].Name == "redis" {
			redisResult = &rep.Checks[i]
		}
	}
	if assert.NotNil(t, redisResult) {
		assert.Equal(t, "down", redisResult.Status)
	}
}

func TestHealthCheck_DisabledDeps(t *testing.T) {
	base := newHealthBase([]health.Checker{
		{Name: "mysql", Ping: func(context.Context) error { return nil }},
		{Name: "redis", Ping: func(context.Context) error { return nil }},
		{Name: "rabbitmq", Disabled: true},
		{Name: "kafka", Disabled: true},
	})
	req := httptest.NewRequest(http.MethodGet, "/api/health/test", nil)
	w := httptest.NewRecorder()
	base.HealthCheck(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var rep health.Report
	assert.NoError(t, json.NewDecoder(w.Body).Decode(&rep))
	assert.Equal(t, "healthy", rep.Status)
	for _, c := range rep.Checks {
		if c.Name == "rabbitmq" || c.Name == "kafka" {
			assert.Equal(t, "disabled", c.Status, c.Name)
		}
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `GOTOOLCHAIN=auto go test ./controllers/ -run TestHealthCheck -v`
Expected: FAIL — `HealthCheck` still returns the old static body (status code 200 even when a checker is down; no `checks` decoded).

- [ ] **Step 3: Create the live-checker builder**

Create `controllers/health.go`:

```go
package controllers

import (
	"context"
	"errors"

	"github.com/bicosteve/booking-system/pkg/health"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// defaultLiveCheckers builds checkers from Base's persistent handles. Disabled
// message brokers are reported as "disabled".
func (b *Base) defaultLiveCheckers() []health.Checker {
	var cs []health.Checker

	cs = append(cs, health.Checker{
		Name: "mysql",
		Ping: func(ctx context.Context) error {
			if b.DB == nil {
				return errors.New("mysql client not initialized")
			}
			return b.DB.PingContext(ctx)
		},
	})

	cs = append(cs, health.Checker{
		Name: "redis",
		Ping: func(ctx context.Context) error {
			if b.Redis == nil {
				return errors.New("redis client not initialized")
			}
			return b.Redis.Ping(ctx).Err()
		},
	})

	if b.RabbitMQStatus == 1 {
		cs = append(cs, health.RabbitProbe(b.rabbitURL))
	} else {
		cs = append(cs, health.Checker{Name: "rabbitmq", Disabled: true})
	}

	if b.KafkaStatus == 1 && b.KafkaProducer != nil {
		cs = append(cs, health.Checker{
			Name: "kafka",
			Ping: func(ctx context.Context) error {
				ac, err := kafka.NewAdminClientFromProducer(b.KafkaProducer)
				if err != nil {
					return err
				}
				defer ac.Close()
				_, err = ac.GetMetadata(nil, true, 1000)
				return err
			},
		})
	} else {
		cs = append(cs, health.Checker{Name: "kafka", Disabled: true})
	}

	return cs
}

// healthCheckers returns the provider-derived checkers (tests), or the live
// handle checkers (production).
func (b *Base) healthCheckers() []health.Checker {
	if b.checkersProvider != nil {
		return b.checkersProvider()
	}
	return b.defaultLiveCheckers()
}
```

- [ ] **Step 4: Rewrite the `HealthCheck` handler**

Replace the `HealthCheck` handler in `controllers/userhandler.go` (lines 19-34) with:

```go
// TestApp godoc
// @Summary Check status of the app and its dependencies
// @Description Reports whether MySQL, Redis, RabbitMQ, and Kafka are reachable
// @ID check-health
// @Tags health
// @Produce json
// @Success 200 {object} health.Report "All enabled dependencies reachable"
// @Failure 503 {object} health.Report "One or more enabled dependencies down"
// @Router /api/health/test [get]
// @Security []
func (b *Base) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	report := health.Check(r.Context(), b.healthCheckers())

	status := http.StatusOK
	if report.Status != "healthy" {
		status = http.StatusServiceUnavailable
	}

	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(report)
}
```

Update the import block at the top of `controllers/userhandler.go` to add `encoding/json` and the health package, and remove now-unused import(s):

```go
import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/bicosteve/booking-system/entities"
	"github.com/bicosteve/booking-system/pkg/health"
	"github.com/bicosteve/booking-system/pkg/utils"
	_ "github.com/swaggo/http-swagger/v2"
)
```

> Verify no other lines in `userhandler.go` use a removed import; if `errors`, `fmt`, `context`, `time`, or `utils` were only used by the old HealthCheck, check the remainder of the file before removing. Only add `encoding/json` and `health`; keep everything else the file already uses.

- [ ] **Step 5: Run the tests to verify they pass**

Run: `GOTOOLCHAIN=auto go test ./controllers/ -run TestHealthCheck -v`
Expected: PASS — three tests pass (200/healthy, 503/unhealthy, 200 with disabled).

- [ ] **Step 6: Run the full controllers test suite**

Run: `GOTOOLCHAIN=auto go test ./controllers/ -v`
Expected: PASS — existing tests still pass.

- [ ] **Step 7: Commit**

```bash
git add controllers/health.go controllers/health_test.go controllers/userhandler.go
git commit -m "feat(controllers): rewrite HealthCheck to report dependency reachability"
```

---

### Task 7: Fix Dockerfile `curl` install and docker-compose healthcheck URL

**Files:**
- Modify: `Dockerfile:22-28`
- Modify: `docker-compose.yml:16-22`

**Interfaces:** None (infra only).

- [ ] **Step 1: Fix the Dockerfile runtime image**

In `Dockerfile`, replace the runtime stage `apt-get` + invalid `apk` block (lines 22-28):

```dockerfile
RUN apt-get update \
 && apt-get install -y --no-install-recommends librdkafka1 ca-certificates curl \
 && rm -rf /var/lib/apt/lists/*

WORKDIR /app
```

(Remove the standalone `RUN apk add --no-cache curl` line entirely — `apk` is not available on `debian:bookworm-slim`.)

- [ ] **Step 2: Fix the docker-compose healthcheck URL**

In `docker-compose.yml`, change the healthcheck `test` line so the default-port syntax is valid:

```yaml
  healthcheck:
    test:
      ["CMD-SHELL", "curl -fsS http://127.0.0.1:${PORT:-7001}/api/health/test"]
    interval: 30s
    timeout: 5s
    retries: 5
    start_period: 60s
```

- [ ] **Step 3: Verify the compose file parses**

Run: `docker compose config -q 2>&1 || echo "docker not available; static check only"`
Expected: no interpolation/parse error. (If docker is unavailable, visually confirm the `${PORT:-7001}` syntax is correct.)

- [ ] **Step 4: Commit**

```bash
git add Dockerfile docker-compose.yml
git commit -m "fix(docker): install curl via apt-get and fix healthcheck port default"
```

---

### Task 8: Full verification

**Files:** None (verification only).

- [ ] **Step 1: Build everything**

Run: `GOTOOLCHAIN=auto go build ./...`
Expected: exit 0, no output.

- [ ] **Step 2: Run the relevant test packages**

Run: `GOTOOLCHAIN=auto go test ./pkg/health/... ./pkg/utils/... ./controllers/... -v`
Expected: all packages PASS (including the existing `bookinghandler_test.go` and `userhandler_test.go` cases).

- [ ] **Step 3: Run golangci-lint**

Run: `GOTOOLCHAIN=auto golangci-lint run ./...`
Expected: no new findings. (If `golangci-lint` is not installed locally, run `go vet ./...` instead and note the limitation.)

- [ ] **Step 4: Manual smoke check of the endpoint (optional)**

If a local stack is running, curl the endpoint:

Run: `curl -i http://localhost:7001/api/health/test`
Expected: `200` with `{"status":"healthy","checks":[...]}` when deps are up; `503` with `"unhealthy"` when a dep is down.

- [ ] **Step 5: Final commit (only if anything needed a touch-up)**

Only if Step 1-3 required edits:

```bash
git add -A
git commit -m "chore(health): verification touch-ups"
```

---

## Self-Review Notes

- **Spec coverage:** `pkg/health` Checker/Check/Await (Tasks 1-2) ✓; probe builders (Task 3) ✓; remove `log.Fatalf` (Task 4) ✓; startup gate honoring On flags (Task 5) ✓; live-handle checkers + `rabbitURL`/`checkersProvider` (Tasks 5-6) ✓; endpoint 200/503 + per-dep JSON (Task 6) ✓; tests for health core, endpoint, disabled (Tasks 1-2,6) ✓; Dockerfile apt curl (Task 7) ✓; docker-compose `${PORT:-7001}` (Task 7) ✓; STARTUP_DEPENDENCY_TIMEOUT env default 60s (Task 5) ✓; swagger `@Param` fixed (Task 6) ✓.
- **Known caveat (not a code gap):** `base.go` still connects to RabbitMQ in the `else` branch even when `RabbitMQStatus != 1` (pre-existing behavior). The startup gate and endpoint honor `On`; this connect path is intentionally left unchanged to avoid scope creep. If disabled-RabbitMQ-without-a-server must not exit, gate the connect by `On` in a follow-up.
- **Type consistency:** `rabbitURL` (package-level, returns string) is used in both `startup.go` and `base.go`. `health.RabbitProbe` returns `health.Checker`, appended directly. `Base.healthCheckers()` and `Base.defaultLiveCheckers()` names match between Tasks 5 and 6.
