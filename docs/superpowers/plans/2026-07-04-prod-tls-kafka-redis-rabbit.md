# Production TLS (Kafka/Aiven, Redis/Upstash) & RabbitMQ Off — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the booking app connect to Aiven Kafka (SASL_SSL+SCRAM) and Upstash Redis (TLS) in production, switch RabbitMQ off cleanly, and keep the startup gate / health endpoint probes consistent with that TLS config.

**Architecture:** Add TLS/SASL fields to the existing `entities` config structs; add pure config builders in `pkg/utils` (`KafkaConfigMap`, `RabbitTLSConfig`) and a `redisOptions` split in `connections`; `pkg/health` probes take low-level handles (`*kafka.ConfigMap`, `*tls.Config`) so it stays free of `utils`/`entities`; `controllers/base.go` wires prod env vars into the structs and passes them to connect/publish.

**Tech Stack:** Go 1.25, `confluent-kafka-go/v2`, `streadway/amqp`, `redis/go-redis/v9`, `crypto/tls`, `crypto/x509`, `encoding/pem`, `stretchr/testify`.

## Global Constraints

- Go toolchain: `go 1.25.0` / `toolchain go1.25.11` (already in `go.mod`). Build with `GOTOOLCHAIN=auto`.
- No new third-party dependencies.
- Honor existing `On` flags: Kafka/RabbitMQ only when `On==1`; MySQL+Redis always required.
- `pkg/health` must NOT import `pkg/utils` or `entities` — it takes pre-built `*kafka.ConfigMap` / `*tls.Config`.
- **Formatting:** After creating or editing ANY `.go` file, run `gofmt -w <file>` before tests/commit. The code-quality job enables the `gofmt` and `errcheck` linters — never ignore a returned error without `_ = `; use `_ = cm.Set(...)` for `kafka.ConfigMap`.
- Commit after every task.

## File Structure

- Modify `entities/entities.go` — add TLS/SASL/bool fields to `KakfaConfig`, `RedisConfig`, `RabbitMQConfig`.
- Modify `pkg/utils/queue.go` — add `KafkaConfigMap`, `RabbitTLSConfig`; change `ProducerConnect`/`ConsumerConnect`/`QPublishMessage`/`NewRabbitMQConnection` signatures.
- Create `pkg/utils/kafka_test.go` — `KafkaConfigMap` tests.
- Modify `pkg/utils/queue_test.go` — update `NewRabbitMQConnection` call site.
- Create `pkg/utils/rabbit_tls_test.go` — `RabbitTLSConfig` tests.
- Modify `connections/redis.go` — split `redisOptions` + explicit TLS + `ServerName`.
- Modify `connections/redis_test.go` — add `redisOptions` tests.
- Modify `pkg/health/probes.go` — `KafkaProbe(cm *kafka.ConfigMap)`, `RabbitProbe(url string, tls *tls.Config)`.
- Modify `pkg/health/probes_test.go` — update probe tests.
- Modify `controllers/base.go` — prod env wiring, `b.kafkaCfg`/`b.rabbitCfg` fields, gate Rabbit by `On` (remove `else`), TLS connect calls.
- Modify `controllers/startup.go` — `rabbitURL` scheme; probes use `utils.KafkaConfigMap`/`utils.RabbitTLSConfig`; add `envBool` helper.
- Modify `controllers/health.go` — rabbit checker uses `utils.RabbitTLSConfig(b.rabbitCfg)`.
- Modify `controllers/bookinghandler.go` — `QPublishMessage(b.kafkaCfg, ...)`.
- Modify `.env-example` and `deploy.md` — document new env vars.

## API reference (installed versions)

- `kafka.NewProducer(cm *kafka.ConfigMap) (*kafka.Producer, error)`; `cm.Set(k, v) error`; `cm.Get(k, def) (interface{}, error)`.
- `kafka.NewAdminClient(cm *kafka.ConfigMap) (*kafka.AdminClient, error)`; `ac.GetMetadata(topic *string, allTopics bool, timeoutMs int) (*kafka.Metadata, error)`.
- `amqp.Dial(url string)` / `amqp.DialTLS(url string, cfg *tls.Config)`; `(*amqp.Connection).Close() error`.

---

### Task 1: entities — add TLS/SASL/bool fields

**Files:**
- Modify: `entities/entities.go` (structs `KakfaConfig` ~113, `RedisConfig` ~105, `RabbitMQConfig` ~121)

**Interfaces:**
- Produces: new fields on existing structs consumed by later tasks.

- [ ] **Step 1: Add fields**

Update the three structs (keep the typo'd `KakfaConfig` name and its `toml` tag):

```go
type KakfaConfig struct {
	Name    string   `toml:"name"`
	Broker  string   `toml:"broker"`
	Topics  []string `toml:"topics"`
	Key     string   `toml:"key"`
	On      int      `toml:"on"`
	SecurityProtocol string `toml:"securityprotocol"` // "SASL_SSL" in prod; "" => plaintext
	SaslMechanism    string `toml:"saslmechanism"`    // default "SCRAM-SHA-256"
	SaslUsername     string `toml:"saslusername"`
	SaslPassword     string `toml:"saslpassword"`
	CaPem            string `toml:"capem"`      // inline CA PEM (ssl.ca.pem)
	CaLocation       string `toml:"calocation"` // optional file path; precedence
}

type RedisConfig struct {
	Name     string `toml:"name"`
	Address  string `toml:"address"`
	Password string `toml:"password"`
	Port     string `toml:"port"`
	Database int    `toml:"database"`
	TLS      bool   `toml:"tls"`
}

type RabbitMQConfig struct {
	Name      string `toml:"name"`
	Host      string `toml:"host"`
	Password  string `toml:"password"`
	User      string `toml:"user"`
	Queue     string `toml:"queue"`
	On        int    `toml:"on"`
	Port      string `toml:"port"`
	Vhost     string `toml:"vhost"`
	TLS       bool   `toml:"tls"`
	CaPem     string `toml:"capem"`
	CaLocation string `toml:"calocation"`
}
```

- [ ] **Step 2: Verify it builds and existing toml still parses**

Run: `GOTOOLCHAIN=auto go build ./... && GOTOOLCHAIN=auto go test ./pkg/app/... -run TestLoadConfigs -v`
Expected: pass (adding fields without a default is a no-op for existing toml).

- [ ] **Step 3: Commit**

```bash
gofmt -w entities/entities.go
git add entities/entities.go
git commit -m "feat(entities): add TLS/SASL fields for kafka/redis/rabbit configs"
```

---

### Task 2: `pkg/utils` — `KafkaConfigMap` + `RabbitTLSConfig` builders

**Files:**
- Modify: `pkg/utils/queue.go`
- Create: `pkg/utils/kafka_test.go`
- Create: `pkg/utils/rabbit_tls_test.go`

**Interfaces:**
- Produces: `func KafkaConfigMap(cfg entities.KakfaConfig) *kafka.ConfigMap`; `func RabbitTLSConfig(rb entities.RabbitMQConfig) *tls.Config`.

- [ ] **Step 1: Write failing tests**

Create `pkg/utils/kafka_test.go`:

```go
package utils

import (
	"testing"

	"github.com/bicosteve/booking-system/entities"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/stretchr/testify/assert"
)

func TestKafkaConfigMap_Plaintext(t *testing.T) {
	cm := KafkaConfigMap(entities.KakfaConfig{Broker: "localhost:19092"})
	v, err := cm.Get("bootstrap.servers", 0)
	assert.NoError(t, err)
	assert.Equal(t, "localhost:19092", v)
	_, err = cm.Get("security.protocol", 0)
	assert.Error(t, err, "plaintext map must not set security.protocol")
}

func TestKafkaConfigMap_SASL_SSL(t *testing.T) {
	cm := KafkaConfigMap(entities.KakfaConfig{
		Broker: "a.example:9092", SecurityProtocol: "SASL_SSL",
		SaslUsername: "u", SaslPassword: "p", CaPem: "PEM",
	})
	v, _ := cm.Get("security.protocol", 0)
	assert.Equal(t, "SASL_SSL", v)
	v, _ = cm.Get("sasl.mechanisms", 0)
	assert.Equal(t, "SCRAM-SHA-256", v, "default mechanism")
	v, _ = cm.Get("sasl.username", 0)
	assert.Equal(t, "u", v)
	v, _ = cm.Get("sasl.password", 0)
	assert.Equal(t, "p", v)
	v, _ = cm.Get("ssl.ca.pem", 0)
	assert.Equal(t, "PEM", v)
	_, err := cm.Get("ssl.ca.location", 0)
	assert.Error(t, err)
}

func TestKafkaConfigMap_CaLocationPrecedence(t *testing.T) {
	cm := KafkaConfigMap(entities.KakfaConfig{
		Broker: "a", SecurityProtocol: "SASL_SSL",
		CaPem: "P", CaLocation: "/etc/ca.pem",
	})
	_, err := cm.Get("ssl.ca.pem", 0)
	assert.Error(t, err, "CaLocation must take precedence; ca.pem not set")
	v, _ := cm.Get("ssl.ca.location", 0)
	assert.Equal(t, "/etc/ca.pem", v)
}

func TestKafkaConfigMap_CustomMechanism(t *testing.T) {
	cm := KafkaConfigMap(entities.KakfaConfig{Broker: "a", SecurityProtocol: "SASL_SSL", SaslMechanism: "SCRAM-SHA-512"})
	v, _ := cm.Get("sasl.mechanisms", 0)
	assert.Equal(t, "SCRAM-SHA-512", v)
}
```

Create `pkg/utils/rabbit_tls_test.go`:

```go
package utils

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"

	"github.com/bicosteve/booking-system/entities"
	"github.com/stretchr/testify/assert"
)

func testCAPEM(t *testing.T) string {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("gen key: %v", err)
	}
	tmpl := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "test"}, IsCA: true, KeyUsage: x509.KeyUsageCertSign}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create cert: %v", err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
}

func TestRabbitTLSConfig_Disabled(t *testing.T) {
	assert.Nil(t, RabbitTLSConfig(entities.RabbitMQConfig{}))
}

func TestRabbitTLSConfig_NoCA(t *testing.T) {
	cfg := RabbitTLSConfig(entities.RabbitMQConfig{TLS: true, Host: "b.example.com"})
	if assert.NotNil(t, cfg) {
		assert.Equal(t, "b.example.com", cfg.ServerName)
		assert.Nil(t, cfg.RootCAs, "no CA => system roots expected")
	}
}

func TestRabbitTLSConfig_WithCaPem(t *testing.T) {
	cfg := RabbitTLSConfig(entities.RabbitMQConfig{TLS: true, Host: "b.example.com", CaPem: testCAPEM(t)})
	if assert.NotNil(t, cfg) {
		assert.NotNil(t, cfg.RootCAs, "RootCAs populated from valid PEM")
		assert.Equal(t, "b.example.com", cfg.ServerName)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `GOTOOLCHAIN=auto go test ./pkg/utils/ -run "KafkaConfigMap|RabbitTLSConfig" -v`
Expected: FAIL — `KafkaConfigMap`/`RabbitTLSConfig` undefined.

- [ ] **Step 3: Implement the builders**

Add to `pkg/utils/queue.go` (keep existing imports; the file already imports `crypto/x509`? no — add what's missing). Update the import block to include `crypto/tls`, `crypto/x509`, `encoding/pem`, `os` (os already imported):

```go
import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
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

Append the builders:

```go
// KafkaConfigMap builds a librdkafka ConfigMap from an entity config.
// Plaintext when cfg.SecurityProtocol is empty; SASL_SSL + SCRAM otherwise.
func KafkaConfigMap(cfg entities.KakfaConfig) *kafka.ConfigMap {
	cm := &kafka.ConfigMap{"bootstrap.servers": cfg.Broker}
	if cfg.SecurityProtocol == "" {
		return cm
	}
	_ = cm.Set("security.protocol", cfg.SecurityProtocol)
	mech := cfg.SaslMechanism
	if mech == "" {
		mech = "SCRAM-SHA-256"
	}
	_ = cm.Set("sasl.mechanisms", mech)
	_ = cm.Set("sasl.username", cfg.SaslUsername)
	_ = cm.Set("sasl.password", cfg.SaslPassword)
	if cfg.CaLocation != "" {
		_ = cm.Set("ssl.ca.location", cfg.CaLocation)
	} else if cfg.CaPem != "" {
		_ = cm.Set("ssl.ca.pem", cfg.CaPem)
	}
	return cm
}

// RabbitTLSConfig returns nil when rb.TLS is false; otherwise a *tls.Config with
// ServerName set and RootCAs populated from CaLocation (file) or CaPem (inline)
// when provided. Empty CA => system roots (public-CA brokers).
func RabbitTLSConfig(rb entities.RabbitMQConfig) *tls.Config {
	if !rb.TLS {
		return nil
	}
	cfg := &tls.Config{ServerName: rb.Host}
	pool := x509.NewCertPool()
	switch {
	case rb.CaLocation != "":
		if b, err := os.ReadFile(rb.CaLocation); err == nil && pool.AppendCertsFromPEM(b) {
			cfg.RootCAs = pool
		}
	case rb.CaPem != "":
		if pool.AppendCertsFromPEM([]byte(rb.CaPem)) {
			cfg.RootCAs = pool
		}
	}
	return cfg
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `GOTOOLCHAIN=auto go test ./pkg/utils/ -run "KafkaConfigMap|RabbitTLSConfig" -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
gofmt -w pkg/utils/queue.go pkg/utils/kafka_test.go pkg/utils/rabbit_tls_test.go
git add pkg/utils/queue.go pkg/utils/kafka_test.go pkg/utils/rabbit_tls_test.go
git commit -m "feat(utils): add KafkaConfigMap and RabbitTLSConfig builders"
```

---

### Task 3: `connections` — `redisOptions` split + explicit TLS

**Files:**
- Modify: `connections/redis.go`
- Modify: `connections/redis_test.go`

**Interfaces:**
- Produces (unexported, same package): `func redisOptions(cfg entities.RedisConfig) *redis.Options`. `NewRedisDB` keeps its existing signature but uses it.

- [ ] **Step 1: Write failing tests**

Append to `connections/redis_test.go` (ensure it's `package connections`):

```go
func TestRedisOptions_NoTLS(t *testing.T) {
	opts := redisOptions(entities.RedisConfig{Address: "127.0.0.1", Port: "6379", Password: "x"})
	assert.Nil(t, opts.TLSConfig)
	assert.Equal(t, "127.0.0.1:6379", opts.Addr)
	assert.Equal(t, "x", opts.Password)
}

func TestRedisOptions_TLS(t *testing.T) {
	opts := redisOptions(entities.RedisConfig{Address: "us1-x.upstash.io", Port: "6379", TLS: true})
	if assert.NotNil(t, opts.TLSConfig) {
		assert.Equal(t, "us1-x.upstash.io", opts.TLSConfig.ServerName)
	}
}
```

Add imports if missing (`github.com/bicosteve/booking-system/entities`, `github.com/stretchr/testify/assert`).

- [ ] **Step 2: Run tests to verify they fail**

Run: `GOTOOLCHAIN=auto go test ./connections/ -run TestRedisOptions -v`
Expected: FAIL — `redisOptions` undefined.

- [ ] **Step 3: Refactor `redis.go`**

Replace the body of `connections/redis.go` with:

```go
package connections

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"github.com/bicosteve/booking-system/entities"
	"github.com/bicosteve/booking-system/pkg/utils"
	"github.com/redis/go-redis/v9"
)

func redisOptions(cfg entities.RedisConfig) *redis.Options {
	var tlsConfig *tls.Config
	if cfg.TLS {
		tlsConfig = &tls.Config{ServerName: cfg.Address}
	}
	return &redis.Options{
		Addr:        cfg.Address + ":" + cfg.Port,
		Password:    cfg.Password,
		DB:          cfg.Database,
		ClientName:  cfg.Name,
		PoolSize:    100,
		PoolTimeout: time.Second * 20,
		MinIdleConns: 32,
		TLSConfig:   tlsConfig,
	}
}

func NewRedisDB(ctx context.Context, cfg entities.RedisConfig) (*redis.Client, error) {
	client := redis.NewClient(redisOptions(cfg))
	pong, err := client.Ping(ctx).Result()
	if err != nil {
		return nil, err
	}
	utils.LogInfo(fmt.Sprintf("REDIS: %v", pong), entities.InfoLog)
	return client, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `GOTOOLCHAIN=auto go test ./connections/ -v`
Expected: PASS (new `redisOptions` tests + existing redis/mysql tests).

- [ ] **Step 5: Commit**

```bash
gofmt -w connections/redis.go connections/redis_test.go
git add connections/redis.go connections/redis_test.go
git commit -m "refactor(connections): split redisOptions, explicit TLS with ServerName"
```

---

### Task 4: `pkg/health` — TLS-aware probe signatures + controller callers

**Files:**
- Modify: `pkg/health/probes.go`
- Modify: `pkg/health/probes_test.go`
- Modify: `controllers/base.go` (add `b.rabbitCfg` field)
- Modify: `controllers/startup.go` (`rabbitURL` scheme, use builders, add `envBool`)
- Modify: `controllers/health.go` (rabbit checker uses TLS config)

**Interfaces:**
- Consumes: `utils.KafkaConfigMap`, `utils.RabbitTLSConfig` from Task 2.
- Produces: `func KafkaProbe(cm *kafka.ConfigMap) Checker`; `func RabbitProbe(url string, tlsConfig *tls.Config) Checker`.

- [ ] **Step 1: Update probe tests**

In `pkg/health/probes_test.go`, replace the `TestKafkaProbe_BadAddress` and `TestRabbitProbe_BadAddress` bodies and add a `kafka` import:

```go
import (
	"context"
	"testing"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/stretchr/testify/assert"
)

func TestKafkaProbe_BadAddress(t *testing.T) {
	cm := &kafka.ConfigMap{"bootstrap.servers": "127.0.0.1:1"}
	probeDown(t, KafkaProbe(cm))
	assert.Equal(t, "kafka", KafkaProbe(&kafka.ConfigMap{}).Name)
}

func TestRabbitProbe_BadAddress(t *testing.T) {
	probeDown(t, RabbitProbe("amqp://guest:guest@127.0.0.1:1/", nil))
	assert.Equal(t, "rabbitmq", RabbitProbe("", nil).Name)
}
```

- [ ] **Step 2: Run to see they fail (signature mismatch)**

Run: `GOTOOLCHAIN=auto go test ./pkg/health/ -run Probe -v 2>&1 | head`
Expected: build failure — `KafkaProbe`/`RabbitProbe` arg types changed.

- [ ] **Step 3: Update `probes.go`**

Replace `KafkaProbe` and `RabbitProbe` in `pkg/health/probes.go` (keep MySQL/Redis probes), and add `crypto/tls` to imports:

```go
import (
	"context"
	"crypto/tls"
	"database/sql"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/redis/go-redis/v9"
	"github.com/streadway/amqp"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// KafkaProbe uses an ephemeral AdminClient built from cm to fetch cluster metadata.
func KafkaProbe(cm *kafka.ConfigMap) Checker {
	return Checker{
		Name: "kafka",
		Ping: func(ctx context.Context) error {
			ac, err := kafka.NewAdminClient(cm)
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

// RabbitProbe dials url once; amqp.DialTLS when tlsConfig != nil, else amqp.Dial.
func RabbitProbe(url string, tlsConfig *tls.Config) Checker {
	return Checker{
		Name: "rabbitmq",
		Ping: func(ctx context.Context) error {
			done := make(chan error, 1)
			go func() {
				var conn *amqp.Connection
				var err error
				if tlsConfig != nil {
					conn, err = amqp.DialTLS(url, tlsConfig)
				} else {
					conn, err = amqp.Dial(url)
				}
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
```

- [ ] **Step 4: Add `b.rabbitCfg` field to `Base`**

In `controllers/base.go`, in the `Base` struct, add near `rabbitURL`:

```go
	rabbitURL string
	rabbitCfg entities.RabbitMQConfig
	kafkaCfg  entities.KakfaConfig
```

(Adding `kafkaCfg` now too; Task 6 populates it.)

- [ ] **Step 5: Update `controllers/startup.go`**

Replace `rabbitURL` with a TLS-aware scheme, and update `buildStartupProbes` kafka/rabbit loops, and add an `envBool` helper. The full updated parts:

```go
// rabbitURL builds the amqp(s) URL. In prod the vhost is included; elsewhere it is omitted.
func rabbitURL(rb entities.RabbitMQConfig) string {
	scheme := "amqp"
	if rb.TLS {
		scheme = "amqps"
	}
	if os.Getenv("ENV") == "prod" {
		return fmt.Sprintf("%s://%s:%s@%s:%s/%s", scheme, rb.User, rb.Password, rb.Host, rb.Port, rb.Vhost)
	}
	return fmt.Sprintf("%s://%s:%s@%s:%s", scheme, rb.User, rb.Password, rb.Host, rb.Port)
}

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
			cs = append(cs, health.KafkaProbe(utils.KafkaConfigMap(k)))
		}
	}
	for _, rb := range cfg.Rabbit {
		if rb.On == 1 {
			url := rabbitURL(rb)
			b.rabbitURL = url
			cs = append(cs, health.RabbitProbe(url, utils.RabbitTLSConfig(rb)))
		}
	}
	return cs
}

// envBool reads a boolean env var; returns def when unset/invalid.
func envBool(name string, def bool) bool {
	switch os.Getenv(name) {
	case "true", "1":
		return true
	case "false", "0":
		return false
	default:
		return def
	}
}
```

(The `startupTimeout` and `waitForDependencies` functions stay as-is.)

- [ ] **Step 6: Update `controllers/health.go` rabbit checker**

In the `if b.RabbitMQStatus == 1 {` branch of `defaultLiveCheckers`, replace `health.RabbitProbe(b.rabbitURL)` with:

```go
	health.RabbitProbe(b.rabbitURL, utils.RabbitTLSConfig(b.rabbitCfg))
```

Add `github.com/bicosteve/booking-system/pkg/utils` to the import block of `controllers/health.go`.

- [ ] **Step 7: Run all affected tests**

Run: `GOTOOLCHAIN=auto go build ./... && GOTOOLCHAIN=auto go test ./pkg/health/... ./controllers/... `
Expected: build + tests pass.

- [ ] **Step 8: Commit**

```bash
gofmt -w pkg/health/probes.go pkg/health/probes_test.go controllers/base.go controllers/startup.go controllers/health.go
git add pkg/health/probes.go pkg/health/probes_test.go controllers/base.go controllers/startup.go controllers/health.go
git commit -m "feat(health): TLS-aware kafka/rabbit probes; wire builders into startup and health"
```

---

### Task 5: RabbitMQ — TLS dial + base.go wiring (off truly skips)

**Files:**
- Modify: `pkg/utils/queue.go` (`NewRabbitMQConnection` signature)
- Modify: `pkg/utils/queue_test.go`
- Modify: `controllers/base.go` (prod `RABBIT_*` env, `b.rabbitCfg` store, gate by `On`)

**Interfaces:**
- Produces: `func NewRabbitMQConnection(url string, tlsConfig *tls.Config) (*amqp.Connection, error)`.

- [ ] **Step 1: Update the `NewRabbitMQConnection` test**

In `pkg/utils/queue_test.go`, change the call to the new signature:

```go
	conn, err := NewRabbitMQConnection("amqp://guest:guest@127.0.0.1:1/", nil)
```

- [ ] **Step 2: Run to see it fail**

Run: `GOTOOLCHAIN=auto go test ./pkg/utils/ -run TestNewRabbitMQConnection -v`
Expected: FAIL — arg count/signature mismatch.

- [ ] **Step 3: Change `NewRabbitMQConnection` in `queue.go`**

Replace the dial site (top of `NewRabbitMQConnection`) with branch on `tlsConfig`:

```go
func NewRabbitMQConnection(qURI string, tlsConfig *tls.Config) (*amqp.Connection, error) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		LogError("RABBITMQ: Termination signal. Exiting...", entities.ErrorLog)
		os.Exit(1)
	}()

	var conn *amqp.Connection
	var err error
	if tlsConfig != nil {
		conn, err = amqp.DialTLS(qURI, tlsConfig)
	} else {
		conn, err = amqp.Dial(qURI)
	}
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
	RabbitMQClient = &entities.RabbitMQ{Connection: conn, Channel: ch}
	LogInfo("RABBITMQ: Connected successfully", entities.InfoLog)
	return RabbitMQClient.Connection, nil
}
```

(Adds `crypto/tls` already present from Task 2.)

- [ ] **Step 4: Wire base.go rabbit block**

In `controllers/base.go` prod `Rabbit` config block, set the TLS/CA fields from env (add inside the existing `Rabbit: []entities.RabbitMQConfig{{ ... }}` literal):

```go
		Rabbit: []entities.RabbitMQConfig{
			{
				Host:      os.Getenv("RABBIT_HOST"),
				Port:      os.Getenv("RABBIT_PORT"),
				User:      os.Getenv("RABBIT_USER"),
				Password:  os.Getenv("RABBIT_PASSWORD"),
				Vhost:     os.Getenv("RABBIT_VHOST"),
				Queue:     os.Getenv("RABBIT_QUEUE"),
				On:        rabbitMQStatus,
				TLS:       envBool("RABBIT_TLS", false),
				CaPem:     os.Getenv("RABBIT_CA_PEM"),
				CaLocation: os.Getenv("RABBIT_CA_LOCATION"),
			},
		},
```

In the `for _, rabbitConf := range config.Rabbit` loop, add after `b.RabbitMQStatus = rabbitConf.On`:

```go
		b.rabbitCfg = rabbitConf
```

Replace the rabbit connect block (`if b.RabbitMQStatus == 1 && os.Getenv("ENV") == "prod" { ... } else { ... }`) with the On-gated single block:

```go
if b.RabbitMQStatus == 1 {
	url := rabbitURL(rabbitConf)
	b.rabbitURL = url
	conn, err := utils.NewRabbitMQConnection(url, utils.RabbitTLSConfig(b.rabbitCfg))
	if err != nil {
		utils.LogError(err.Error(), entities.ErrorLog)
		os.Exit(1)
	}
	b.rabbitConn = conn
}
// else: RabbitMQ disabled — no dial, no exit.
```

- [ ] **Step 5: Run tests + build**

Run: `GOTOOLCHAIN=auto go build ./... && GOTOOLCHAIN=auto go test ./pkg/utils/... ./controllers/...`
Expected: pass.

- [ ] **Step 6: Commit**

```bash
gofmt -w pkg/utils/queue.go pkg/utils/queue_test.go controllers/base.go
git add pkg/utils/queue.go pkg/utils/queue_test.go controllers/base.go
git commit -m "feat(rabbit): TLS dial + gate RabbitMQ by On in prod (off truly skips)"
```

---

### Task 6: Kafka — connect/publish signatures + base.go wiring + bookinghandler

**Files:**
- Modify: `pkg/utils/queue.go` (`ProducerConnect`/`ConsumerConnect`/`QPublishMessage`)
- Modify: `controllers/base.go` (prod `KAFKA_*` env, `b.kafkaCfg`, pass cfg)
- Modify: `controllers/bookinghandler.go`

**Interfaces:**
- Produces: `func ProducerConnect(cfg entities.KakfaConfig) (*kafka.Producer, error)`; `func ConsumerConnect(cfg entities.KakfaConfig) (*kafka.Consumer, error)`; `func QPublishMessage(cfg entities.KakfaConfig, topic, key string, data any) error`.

- [ ] **Step 1: Update the three functions in `queue.go`**

```go
func ProducerConnect(cfg entities.KakfaConfig) (*kafka.Producer, error) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		LogError("PRODUCER: Received termination signal. Exiting", entities.ErrorLog)
		os.Exit(1)
	}()
	p, err := kafka.NewProducer(KafkaConfigMap(cfg))
	if err != nil {
		LogError("PRODUCER: Could not connect to broker because: "+err.Error(), entities.ErrorLog)
		return nil, err
	}
	LogInfo("PRODUCER: connected successfully", entities.InfoLog)
	return p, nil
}

func ConsumerConnect(cfg entities.KakfaConfig) (*kafka.Consumer, error) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		LogError("CONSUMER: Received termination signal. Exiting", entities.ErrorLog)
		os.Exit(1)
	}()
	cm := KafkaConfigMap(cfg)
	_ = cm.Set("group.id", "kafka-go-getting-started")
	_ = cm.Set("auto.offset.reset", "earliest")
	c, err := kafka.NewConsumer(cm)
	if err != nil {
		LogError("CONSUMER: Could not connect due to "+err.Error(), entities.ErrorLog)
		return nil, err
	}
	LogInfo("CONSUMER: connected successfully", entities.InfoLog)
	return c, nil
}

func QPublishMessage(cfg entities.KakfaConfig, topic, key string, data any) error {
	wg := &sync.WaitGroup{}
	p, err := kafka.NewProducer(KafkaConfigMap(cfg))
	if err != nil {
		LogError(err.Error(), entities.ErrorLog)
		return errors.New(err.Error())
	}
	defer p.Flush(15 * 100)
	defer p.Close()
	wg.Add(1)
	go func(w *sync.WaitGroup) {
		for e := range p.Events() {
			switch ev := e.(type) {
			case *kafka.Message:
				if ev.TopicPartition.Error != nil {
					LogError("message cannot be delivered because of "+ev.TopicPartition.String(), entities.ErrorLog)
				} else {
					_msg := fmt.Sprintf("Produced events to topic %s key = %-10s value = %s\n", *ev.TopicPartition.Topic, string(ev.Key), string(ev.Value))
					LogInfo(_msg, entities.InfoLog)
				}
			}
		}
		w.Done()
	}(wg)
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	p.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: -1},
		Key:            []byte(key),
		Value:          []byte(string(dataBytes)),
	}, nil)
	return nil
}
```

- [ ] **Step 2: Wire `base.go` kafka block**

In the prod `Kafka` config literal, add the SASL/TLS env fields:

```go
		Kafka: []entities.KakfaConfig{
			{
				Broker: os.Getenv("KAFKA_BROKER"),
				Key: os.Getenv("KAFKA_KEY"),
				Topics: []string{os.Getenv("KAFKA_TOPIC")},
				On: kafkaStatus,
				SecurityProtocol: os.Getenv("KAFKA_SECURITY_PROTOCOL"),
				SaslMechanism: os.Getenv("KAFKA_SASL_MECHANISM"),
				SaslUsername: os.Getenv("KAFKA_SASL_USERNAME"),
				SaslPassword: os.Getenv("KAFKA_SASL_PASSWORD"),
				CaPem: os.Getenv("KAFKA_CA_PEM"),
				CaLocation: os.Getenv("KAFKA_CA_LOCATION"),
			},
		},
```

In the `for _, kafka := range config.Kafka` loop, add `b.kafkaCfg = kafka`, and change the connect calls to pass the struct:

```go
	for _, kafka := range config.Kafka {
		brokerURL = kafka.Broker
		paymentKey = kafka.Key
		paymentTopic = kafka.Topics
		b.KafkaStatus = kafka.On
		b.kafkaCfg = kafka
	}

	if b.KafkaStatus == 1 {
		p, err := utils.ProducerConnect(b.kafkaCfg)
		if err != nil {
			utils.LogError(err.Error(), entities.ErrorLog)
			os.Exit(1)
		}
		c, err := utils.ConsumerConnect(b.kafkaCfg)
		if err != nil {
			utils.LogError(err.Error(), entities.ErrorLog)
			os.Exit(1)
		}
		b.KafkaProducer = p
		b.KafkaConsumer = c
		b.Broker = brokerURL
		b.Topics = paymentTopic
		b.Key = paymentKey
	}
```

- [ ] **Step 3: Update `bookinghandler.go` publish call**

In `controllers/bookinghandler.go:240`, replace:

```go
		err = utils.QPublishMessage(b.Broker, b.Topics[1], b.Key, trx)
```

with:

```go
		err = utils.QPublishMessage(b.kafkaCfg, b.Topics[1], b.Key, trx)
```

- [ ] **Step 4: Run build + tests**

Run: `GOTOOLCHAIN=auto go build ./... && GOTOOLCHAIN=auto go test ./...`
Expected: pass (full suite).

- [ ] **Step 5: Commit**

```bash
gofmt -w pkg/utils/queue.go controllers/base.go controllers/bookinghandler.go
git add pkg/utils/queue.go controllers/base.go controllers/bookinghandler.go
git commit -m "feat(kafka): SASL_SSL connect/publish via KafkaConfigMap; wire prod env"
```

---

### Task 7: Redis — prod TLS env in `base.go`

**Files:**
- Modify: `controllers/base.go` (prod `Redis` config literal)

- [ ] **Step 1: Set `TLS` in the prod Redis config**

In the prod `Redis: []entities.RedisConfig{{ ... }}` literal, add `TLS: envBool("REDIS_TLS", os.Getenv("ENV") == "prod")`:

```go
		Redis: []entities.RedisConfig{
			{
				Name: os.Getenv("REDIS_NAME"),
				Address: os.Getenv("REDIS_ADDRESS"),
				Port: os.Getenv("REDIS_PORT"),
				Password: os.Getenv("REDIS_PASSWORD"),
				Database: redisDB,
				TLS: envBool("REDIS_TLS", os.Getenv("ENV") == "prod"),
			},
		},
```

- [ ] **Step 2: Build + test**

Run: `GOTOOLCHAIN=auto go build ./... && GOTOOLCHAIN=auto go test ./...`
Expected: pass.

- [ ] **Step 3: Commit**

```bash
gofmt -w controllers/base.go
git add controllers/base.go
git commit -m "feat(redis): prod TLS flag (REDIS_TLS, default on in prod)"
```

---

### Task 8: Env docs + full verification

**Files:**
- Modify: `.env-example`
- Modify: `deploy.md`

- [ ] **Step 1: Document new prod env vars**

Add to `.env-example`:

```
# Production managed services
KAFKA_SECURITY_PROTOCOL=SASL_SSL
KAFKA_SASL_MECHANISM=SCRAM-SHA-256
KAFKA_SASL_USERNAME=
KAFKA_SASL_PASSWORD=
KAFKA_CA_PEM=
KAFKA_CA_LOCATION=
REDIS_TLS=true
RABBITMQ_STATUS=0
RABBIT_TLS=false
RABBIT_CA_PEM=
RABBIT_CA_LOCATION=
```

In `deploy.md`, add a short "Production managed services" section stating Aiven Kafka uses SASL_SSL+SCRAM (CA via `KAFKA_CA_PEM`), Upstash Redis uses TLS (auto-on in prod), and RabbitMQ is off (`RABBITMQ_STATUS=0`) with optional AMQPS (`RABBIT_TLS=true`).

- [ ] **Step 2: Full verification**

Run:
```bash
GOTOOLCHAIN=auto go build ./...
GOTOOLCHAIN=auto go test ./...
gofmt -l pkg/ connections/ controllers/ entities/
GOTOOLCHAIN=auto go vet ./...
```
Expected: build pass; all tests pass; `gofmt -l` empty; vet clean.

- [ ] **Step 3: Commit**

```bash
git add .env-example deploy.md
git commit -m "docs: document prod managed-services env (Kafka SASL_SSL, Redis TLS, RabbitMQ off)"
```

---

## Self-Review Notes

- **Spec coverage:** entities fields (Task 1) ✓; `KafkaConfigMap`/`RabbitTLSConfig` (Task 2) ✓; `redisOptions` split + `ServerName` (Task 3) ✓; probe signatures decoupled (Task 4) ✓; rabbit TLS dial + On-gating + remove `else` (Task 5) ✓; kafka connect/publish TLS + bookinghandler (Task 6) ✓; redis prod TLS (Task 7) ✓; env docs (Task 8) ✓. `consumers.go` unchanged (uses persistent consumer) ✓. `pkg/health` never imports `utils`/`entities` (takes `*kafka.ConfigMap`/`*tls.Config`) ✓.
- **Type consistency:** `KafkaConfigMap(cfg entities.KakfaConfig)` used in Task 4 (`health.KafkaProbe(utils.KafkaConfigMap(k))`) and Task 6 (connect/publish). `RabbitTLSConfig(rb entities.RabbitMQConfig)` used in Task 4 and Task 5. `b.kafkaCfg`/`b.rabbitCfg` added in Task 4, populated in Tasks 5/6. `envBool` defined in Task 4, used in Tasks 5/7. Kafka probe test uses `&kafka.ConfigMap{...}` literal consistent with `KafkaProbe(*kafka.ConfigMap)`.
- **Known caveat:** `ProducerConnect`/`ConsumerConnect`/`QPublishMessage` keep their signal/os.Exit goroutines (out of scope); only the config map source changes. Full Aiven/Upstash wiring is not unit-tested with live brokers — verified via build/test/gofmt/vet; real connection validated in the deployed environment.
