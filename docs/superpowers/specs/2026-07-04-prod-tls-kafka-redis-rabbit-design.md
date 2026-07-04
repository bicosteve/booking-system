# Production TLS for Kafka (Aiven) and Redis (Upstash), RabbitMQ Off

Date: 2026-07-04
Status: Approved (pending implementation)

## Problem

The booking system is moving to managed services for production:
- **Kafka** -> Aiven (SASL over TLS / SCRAM).
- **Redis** -> Upstash (TLS required).
- **RabbitMQ** -> switched off in production (`RABBITMQ_STATUS=0`).

The current code cannot do this:

1. `utils.ProducerConnect`/`ConsumerConnect` (`pkg/utils/queue.go`) build a librdkafka
   `ConfigMap` with only `bootstrap.servers` — no TLS/SASL — so they cannot reach Aiven.
2. `utils.QPublishMessage` (`pkg/utils/queue.go:71`) mints its own plaintext producer and is
   the actual publish path used by `controllers/bookinghandler.go:240` — so the prod publish
   path also cannot reach Aiven, even if the persistent producer were TLS-aware.
3. `connections/redis.go` enables TLS only via a "password is set" heuristic and does not set
   `ServerName` (needed for SNI against Upstash). The heuristic is fragile for local dev that
   has a password on a plaintext Redis.
4. `controllers/base.go` RabbitMQ connect block (`base.go:220-236`) connects **unconditionally**
   in its `else` branch, so `RABBITMQ_STATUS=0` does not actually skip RabbitMQ — the app would
   still dial Rabbit and `os.Exit(1)` on failure. (Pre-existing bug flagged during the
   health-check work.)
5. The startup gate / health endpoint Kafka and Rabbit probes (`pkg/health/probes.go`,
   `controllers/health.go`) build their own clients without TLS/SASL, so they would
   false-negative against Aiven (Kafka) or a future TLS broker (RabbitMQ).

There is no way to switch RabbitMQ back on with AMQP**S** later — the connect path is
plaintext `amqp.Dial` only.

## Goals

- Connect Kafka (producer, consumer, publish path) to Aiven via `SASL_SSL` + SCRAM.
- Connect Redis to Upstash via TLS with correct SNI.
- Make `RABBITMQ_STATUS=0` truly skip RabbitMQ (no dial, no exit).
- Optionally support AMQP**S**/TLS for RabbitMQ (in case it's switched back on later).
- Keep the startup gate and health endpoint probes consistent with the real connect config so
  they don't false-negative against managed/TLS brokers.
- Stay decoupled: `pkg/health` must not import `pkg/utils`/`entities`.

## Non-Goals

- No producer delivery-event goroutine redesign.
- No rename of `entities.KakfaConfig` (existing toml tag preserved).
- No change to `controllers/consumers.go` (uses the persistent `b.KafkaConsumer`, which inherits
  TLS).
- No Redis mTLS / client certs; Upstash uses a password + server cert only.

## Decisions

- **Aiven Kafka auth:** SASL over TLS, SCRAM. Default mechanism `SCRAM-SHA-256`, overridable via
  `KAFKA_SASL_MECHANISM` (e.g. `SCRAM-SHA-512`).
- **Aiven CA cert delivery:** inline PEM via env var `KAFKA_CA_PEM`. An optional file path env
  `KAFKA_CA_LOCATION` takes precedence if set (mapped file).
- **Redis TLS:** enabled for production. Default: `true` when `ENV==prod`, overridable via
  `REDIS_TLS` (`true`/`false`/`0`/`1`). `ServerName` = `REDIS_ADDRESS` (the host).
- **RabbitMQ:** off by default in prod (`RABBITMQ_STATUS=0`). When switched on, TLS is controlled
  by `RABBIT_TLS` (default false), with optional `RABBIT_CA_PEM` / `RABBIT_CA_LOCATION` for
  private-CA brokers. Public-CA brokers (CloudAMQP/Aiven RabbitMQ) work with no CA via system
  roots.
- **No new packages.** Thread config through the existing `entities` structs and `pkg/utils` /
  `connections` functions.

## Architecture

A single librdkafka / amqp / redis TLS config is built from the existing `entities` config (populated
from env in `base.go` prod branch) and reused by: (a) the real connect/publish functions, and
(b) the `pkg/health` probes. `pkg/health` receives already-built low-level handles — a
`*kafka.ConfigMap` and a `*tls.Config` — so it stays free of `utils`/`entities` imports.

## Components

### 1. `entities/entities.go`

Add TLS/SASL fields to the Kafka and RabbitMQ config structs, and a `TLS` bool to Redis:

```go
type KakfaConfig struct { // name kept (toml tag) to avoid migration
    Name, Broker, Key string
    Topics []string
    On   int
    SecurityProtocol string // "SASL_SSL" in prod; "" => plaintext local
    SaslMechanism    string // default "SCRAM-SHA-256"
    SaslUsername     string
    SaslPassword     string
    CaPem            string // inline CA PEM (ssl.ca.pem)
    CaLocation       string // optional file path; precedence over CaPem
}

type RedisConfig struct {
    Name, Address, Password, Port string
    Database int
    TLS bool
}

type RabbitMQConfig struct {
    Name, Host, Password, User, Queue, Port, Vhost string
    On int
    TLS   bool
    CaPem      string // inline CA PEM (optional)
    CaLocation string // optional file path; precedence over CaPem
}
```

### 2. `pkg/utils/queue.go` — Kafka connect/publish + a shared configmap builder

```go
// KafkaConfigMap builds the librdkafka ConfigMap. Plaintext when cfg.SecurityProtocol is empty;
// SASL_SSL + SCRAM when set.
func KafkaConfigMap(cfg entities.KakfaConfig) *kafka.ConfigMap

func ProducerConnect(cfg entities.KakfaConfig) (*kafka.Producer, error)
func ConsumerConnect(cfg entities.KakfaConfig) (*kafka.Consumer, error)
func QPublishMessage(cfg entities.KakfaConfig, topic, key string, data any) error
```

`KafkaConfigMap` behavior:
- Set `bootstrap.servers: cfg.Broker`.
- If `cfg.SecurityProtocol != ""`: set `security.protocol`, `sasl.mechanisms` (default
  `SCRAM-SHA-256` when `cfg.SaslMechanism == ""`), `sasl.username`, `sasl.password`.
- CA: `ssl.ca.location` if `cfg.CaLocation != ""`, else `ssl.ca.pem` if `cfg.CaPem != ""`.
- `ProducerConnect`/`ConsumerConnect`/`QPublishMessage` use `KafkaConfigMap(cfg)` instead of the
  inline `bootstrap.servers`-only map. The signal/os.Exit gore in those functions is untouched
  (out of scope).

### 3. `pkg/utils/queue.go` — RabbitMQ TLS dial + TLS config builder

```go
// NewRabbitMQConnection dials url. When tlsConfig != nil it uses amqp.DialTLS, else amqp.Dial.
func NewRabbitMQConnection(url string, tlsConfig *tls.Config) (*amqp.Connection, error)

// RabbitTLSConfig returns nil when rb.TLS is false; otherwise a *tls.Config with ServerName set
// and RootCAs populated from CaLocation (file) or CaPem (inline) when provided.
func RabbitTLSConfig(rb entities.RabbitMQConfig) *tls.Config
```

`RabbitTLSConfig` loads CA via `crypto/x509`:
- `CaLocation != ""`: read file -> `AppendCertsFromPEM` into a `CertPool`.
- else `CaPem != ""`: `AppendCertsFromPEM([]byte(CaPem))`.
- If neither is set: leave `RootCAs` nil so `amqp.DialTLS` uses system roots (public CA brokers).
- `ServerName = rb.Host`.
The existing `TestNewRabbitMQConnection_InvalidURL` updates to pass a `nil` tlsConfig.

### 4. `connections/redis.go` — explicit TLS, `ServerName`, testable options split

Refactor into a pure options builder + the existing connection function:

```go
func redisOptions(cfg entities.RedisConfig) *redis.Options   // testable, no I/O
func NewRedisDB(ctx context.Context, cfg entities.RedisConfig) (*redis.Client, error)
```

`redisOptions`:
- `Addr: cfg.Address + ":" + cfg.Port`, `Password`, `DB`, `ClientName`, pool settings (as today).
- If `cfg.TLS`: `TLSConfig = &tls.Config{ServerName: cfg.Address}`.
- Else: no `TLSConfig`.

Replaces the current "password is set" heuristic. `NewRedisDB` calls `redis.NewClient(redisOptions(cfg))`
then `Ping`.

### 5. `pkg/health/probes.go` — accept low-level TLS handles (decoupled)

```go
func KafkaProbe(cm *kafka.ConfigMap) Checker
func RabbitProbe(url string, tlsConfig *tls.Config) Checker
```

- `KafkaProbe`: builds `kafka.NewAdminClient(cm)` + `GetMetadata`. For local dev the caller passes
  a plaintext configmap; for prod, `utils.KafkaConfigMap(k)` (SASL_SSL).
- `RabbitProbe`: `amqp.DialTLS(url, tlsConfig)` when `tlsConfig != nil`, else `amqp.Dial(url)`.
- `MySQLProbe` / `RedisProbe` unchanged. Existing `TestKafkaProbe_BadAddress` /
  `TestRabbitProbe_BadAddress` updated to build/pass the plaintext inputs.

### 6. `controllers/base.go` — prod env wiring, store configs, gate RabbitMQ by `On`

Prod `Kafka` block reads (existing `KAFKA_BROKER`/`KAFKA_KEY`/`KAFKA_TOPIC`/`KAFKA_STATUS` plus):
- `KAFKA_SECURITY_PROTOCOL` (e.g. `SASL_SSL`)
- `KAFKA_SASL_MECHANISM` (default `SCRAM-SHA-256`)
- `KAFKA_SASL_USERNAME`, `KAFKA_SASL_PASSWORD`
- `KAFKA_CA_PEM`, `KAFKA_CA_LOCATION` (optional)

Prod `Redis` block sets `TLS`:
- Default `TLS = (ENV == "prod")`; if `REDIS_TLS` is set, parse `true`/`false`/`1`/`0` and override.

Prod `Rabbit` block reads (existing `RABBIT_*` plus):
- `RABBIT_TLS` (default false; parse `true`/`false`/`1`/`0`)
- `RABBIT_CA_PEM`, `RABBIT_CA_LOCATION` (optional)

Within the existing `for _, kafka := range config.Kafka` loop, store `b.kafkaCfg = kafka`
(unexported `Base` field). Within `for _, rabbitConf := range config.Rabbit`, store
`b.rabbitCfg = rabbitConf` (unexported `Base` field).

Kafka connect: `utils.ProducerConnect(b.kafkaCfg)`, `utils.ConsumerConnect(b.kafkaCfg)`.

Rabbit block — replaced so off truly skips:
```go
if b.RabbitMQStatus == 1 {
    url := rabbitURL(rabbitConf)                 // amqps:// when rabbitConf.TLS else amqp://
    b.rabbitURL = url
    conn, err := utils.NewRabbitMQConnection(url, utils.RabbitTLSConfig(b.rabbitCfg))
    if err != nil {
        utils.LogError(err.Error(), entities.ErrorLog)
        os.Exit(1)
    }
    b.rabbitConn = conn
}
// else: skipped entirely (no dial, no exit)
```
The unconditional `else` connect is removed.

### 7. `controllers/startup.go` — `rabbitURL` scheme + TLS-aware probes

`rabbitURL(rb)` returns `amqps://…` when `rb.TLS` else `amqp://…`.

`buildStartupProbes`:
- Kafka: `health.KafkaProbe(utils.KafkaConfigMap(k))` for enabled `k`.
- Rabbit: `health.RabbitProbe(rabbitURL(rb), utils.RabbitTLSConfig(rb))` for enabled `rb`.

### 8. `controllers/bookinghandler.go` — publish with TLS config

`utils.QPublishMessage(b.kafkaCfg, b.Topics[1], b.Key, trx)`.

### 9. `controllers/health.go` — TLS-aware rabbit live checker

The Kafka live checker already uses `kafka.NewAdminClientFromProducer(b.KafkaProducer)` and
inherits SASL_SSL — unchanged. The Rabbit live checker becomes:
```go
health.RabbitProbe(b.rabbitURL, utils.RabbitTLSConfig(b.rabbitCfg))
```
(Requires adding `b.rabbitCfg` to `Base`, same as `b.kafkaCfg`.)

### 10. Env documentation

Add the new env vars to `.env-example` and document them in `deploy.md`. Prod `.env` sets
`RABBITMQ_STATUS=0`, `KAFKA_SECURITY_PROTOCOL=SASL_SSL`, `KAFKA_SASL_MECHANISM=SCRAM-SHA-256`,
`KAFKA_SASL_USERNAME`, `KAFKA_SASL_PASSWORD`, `KAFKA_CA_PEM`, and `REDIS_TLS=true` (default).

## Testing

### `pkg/utils` (new tests; existing ones updated)
- `KafkaConfigMap` plaintext: only `bootstrap.servers`; no `security.protocol`.
- `KafkaConfigMap` SASL_SSL: `security.protocol==SASL_SSL`, `sasl.mechanisms==SCRAM-SHA-256`
  when unset, `sasl.username`, `sasl.password`, `ssl.ca.pem` set when `CaPem` provided.
- `KafkaConfigMap` `CaLocation` precedence: when both `CaLocation` and `CaPem` set, only
  `ssl.ca.location` is set.
- `RabbitTLSConfig`: `!TLS` -> nil; `TLS` no CA -> `&tls.Config{ServerName: host}`; `TLS + CaPem`
  -> `RootCAs` non-nil (built from a small valid PEM in the test).
- `TestNewRabbitMQConnection_InvalidURL`: call `NewRabbitMQConnection(url, nil)` and assert an
  error is returned (no `log.Fatalf`).

### `connections` (new tests)
- `redisOptions` `!TLS` -> `TLSConfig == nil`.
- `redisOptions` `TLS` -> `TLSConfig != nil` and `TLSConfig.ServerName == cfg.Address`.
- (Existing `redis_test.go` / `mysql_test.go` unaffected behavior aside from signature touch if
  they call `NewRedisDB`; otherwise unchanged.)

### `pkg/health` (update existing probe tests)
- `TestKafkaProbe_BadAddress`: build `utils.KafkaConfigMap(entities.KakfaConfig{Broker:
  "127.0.0.1:1"})` and pass to `KafkaProbe`; assert error from `Ping` on a closed port.
- `TestRabbitProbe_BadAddress`: pass `("amqp://guest:guest@127.0.0.1:1/", nil)`; assert error.
- `controllers` health-check tests unaffected (use injected `checkersProvider`).

## Risk & Verification

- **Aiven mechanism mismatch:** default `SCRAM-SHA-256`; if an Aiven project uses `SCRAM-SHA-512`,
  set `KAFKA_SASL_MECHANISM`. Mismatch fails the startup gate with a clear `down: kafka` summary.
- **CA PEM inline vs file:** inline (`ssl.ca.pem`) avoids file mounting; `CaLocation` is provided
  as an escape hatch for mounted CA files. Both branches unit-tested.
- **Redis SNI without `ServerName`:** explicitly setting `ServerName` avoids the silent handshake
  hang that motivated the old password heuristic, so the heuristic is no longer needed.
- **Rabbit off now truly off:** removing the unconditional `else` connect is the intended fix;
  prod sets `RABBITMQ_STATUS=0` and nothing dials Rabbit.
- **Verification commands:** `GOTOOLCHAIN=auto go build ./...`,
  `GOTOOLCHAIN=auto go test ./pkg/health/... ./pkg/utils/... ./connections/... ./controllers/...`,
  `gofmt -w .`, `GOTOOLCHAIN=auto go vet ./...`. Full broker wiring is exercised in the deployed
  environment (no live Aiven/Upstash in the test suite).
