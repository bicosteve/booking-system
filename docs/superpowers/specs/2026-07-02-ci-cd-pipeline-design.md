# CI/CD Pipeline Design — booking-system

Date: 2026-07-02
Status: Approved

## Goal

A single GitHub Actions workflow that runs the full CI/CD pipeline for the Go
`booking-system` service: tests with a formatted pass/fail summary, security
scanning, code quality + coverage reporting, a performance placeholder, Docker
image creation, publishing to Docker Hub (main branch only), and a manual EC2
deployment over SSH (`workflow_dispatch` only) that runs the app alongside a
Grafana Alloy sidecar shipping logs to Grafana Loki. A final job summarizes all
results.

## Constraints & Context

- Module `github.com/bicosteve/booking-system`, Go `1.23.0`.
- Imports `confluentinc/confluent-kafka-go/v2`, which requires **CGO** and
  `librdkafka`. Therefore `CGO_ENABLED=1` is mandatory for build and test, and
  `librdkafka-dev` must be installed on runners; the runtime image needs
  `librdkafka1`.
- Prod config is read from `os.Getenv` (not a file) when `ENV=prod`. The
  container must receive these vars at runtime.
- App listens on `7001` (user server) and `7002` (admin server).
- Docker image repository: `bixoloo/booking-system`.

## Triggers

```yaml
on:
  push:
    branches: ["main", "fix/**", "feat/**"]
  pull_request:
    branches: ["main"]
  workflow_dispatch:
    inputs:
      image_tag:
        description: "Image tag to deploy to EC2"
        required: false
        default: "latest"
```

- CI jobs (`test`, `quality`, `security`, `performance`, `summary`) run on
  every `push` to `main`, `fix/**`, `feat/**`, and on `pull_request` targeting
  `main`.
- `build-and-publish` runs on the `main` line: a `push` to `main` OR a
  `pull_request` targeting `main`. The image is always **built** (validating the
  Dockerfile), but it is **pushed to Docker Hub only on `push` to `main`** — PRs
  build without pushing, so an unmerged PR never clobbers the `latest` tag
  (best practice).
- `deploy-ec2` runs only when the event is `workflow_dispatch` AND the ref is
  `main`.
- The workflow uses plain text status words (PASS / FAIL / SKIPPED) in
  summaries; no emojis.

## Job Graph

```
test ──┬─> quality ──┐
       ├─> security ─┤
       └─> performance ─> summary  (if: always)
                     │
build-and-publish (needs: test, quality, security;
                   builds on main line -> push to main OR PR to main;
                   pushes to Docker Hub only on push to main)
deploy-ec2 (if: workflow_dispatch AND ref == main)
```

## Jobs

### 1. test
- Runner: `ubuntu-latest`; `actions/setup-go@v5` with Go `1.23`.
- Install `librdkafka-dev` via apt.
- Run `go test -race -coverprofile=coverage.out -covermode=atomic -json ./...`
  piped through `gotestsum` (or a JSON parser) to render a formatted test table
  into `$GITHUB_STEP_SUMMARY` showing per-package pass/fail counts. The job
  **fails if any test fails** (satisfies "100% pass rate or the test fails").
- Compute total coverage with `go tool cover -func=coverage.out`, print the
  percentage to the summary. **Report only — no coverage gate / no hard fail.**
- Generate `coverage.html` and upload `coverage.out` + `coverage.html` as
  artifacts.

### 2. quality
- Needs: `test`.
- `gofmt -l .` — fail if any file is unformatted.
- `go vet ./...` — fail on suspicious constructs.
- `golangci/golangci-lint-action` — fail on lint violations.
- Config file: `.golangci.yml`.

### 3. security
- Needs: `test`.
- `govulncheck ./...` — **hard fail** on known CVEs in dependencies/stdlib.
- `gosec` (securego/gosec action) — **report only**; upload SARIF, non-blocking
  (`continue-on-error: true`).
- Trivy filesystem scan (`aquasecurity/trivy-action`, `scan-type: fs`) —
  **report only**, non-blocking.
- (Image-level Trivy scan happens in `build-and-publish`.)

### 4. performance
- Needs: `test`.
- Placeholder: run `go test -bench=. -benchmem -run=^$ ./...` if benchmarks
  exist, otherwise write "No benchmarks yet — performance placeholder" to the
  summary. **Never fails.** Structured so real benchmarks can be added later.

### 5. build-and-publish (main line only)
- Condition: runs on the `main` line only, i.e.
  `(github.event_name == 'push' && github.ref == 'refs/heads/main') ||
   (github.event_name == 'pull_request' && github.base_ref == 'main')`.
- Needs: `test`, `quality`, `security`.
- Multi-stage `Dockerfile`:
  - Stage 1 (builder): `golang:1.23-bookworm`, install `librdkafka-dev`,
    `CGO_ENABLED=1 GOOS=linux go build -o /out/bookingapp ./cmd`.
  - Stage 2 (runtime): `debian:bookworm-slim`, install `librdkafka1`,
    `ca-certificates`; copy binary and `docs/`; `EXPOSE 7001 7002`;
    `ENV ENV=prod`; `ENTRYPOINT ["/app/bookingapp"]`.
- `docker/setup-buildx-action`; `docker/login-action` (Docker Hub with
  `DOCKERHUB_USERNAME` / `DOCKERHUB_TOKEN`) runs only when pushing;
  `docker/build-push-action` with Buildx layer cache.
- `push` set to `true` only on `push` to `main`
  (`github.event_name == 'push' && github.ref == 'refs/heads/main'`);
  on PRs the image is built with `push: false` and loaded locally for scanning.
- Tags: `latest` and short git SHA (`${GITHUB_SHA::7}`) — applied when pushing.
- Trivy **image** scan (report only) into the summary; on PRs it scans the
  locally-built image, on main it scans the pushed image.

### 6. deploy-ec2 (workflow_dispatch on main only)
- Condition: `github.event_name == 'workflow_dispatch' &&
  github.ref == 'refs/heads/main'`.
- Deploys via **docker compose** so the app runs with a **Grafana Alloy
  sidecar** that ships logs to Loki.
- `appleboy/scp-action` copies `docker-compose.yml` and `alloy/config.alloy`
  to the server; `appleboy/ssh-action` runs the deploy using `EC2_HOST`,
  `EC2_USER`, `EC2_SSH_KEY`.
- Steps on the server (`~/booking/` working dir):
  1. Receive `docker-compose.yml` and `config.alloy` (via scp).
  2. Write `EC2_ENV_FILE` secret to `~/booking/booking.env`.
  3. Write Loki secrets (`LOKI_URL`, `LOKI_USERNAME`, `LOKI_PASSWORD`) to
     `~/booking/alloy.env`.
  4. `docker login` with Docker Hub creds.
  5. `IMAGE_TAG=${{ inputs.image_tag }} docker compose pull`.
  6. `IMAGE_TAG=${{ inputs.image_tag }} docker compose up -d`.
  7. `docker image prune -f`.
- Report deployment result to the summary.

#### docker-compose.yml (deployed to EC2)
Two services on a shared user-defined network:
- `booking-system`: image
  `bixoloo/booking-system:${IMAGE_TAG:-latest}`, `env_file:
  booking.env`, `environment: ENV=prod`, ports `7001:7001` and `7002:7002`,
  `restart: unless-stopped`.
- `alloy`: image `grafana/alloy:latest`, command runs
  `/etc/alloy/config.alloy`, `env_file: alloy.env`, volumes mount
  `/var/run/docker.sock:/var/run/docker.sock:ro` and
  `./config.alloy:/etc/alloy/config.alloy:ro`, `restart: unless-stopped`.

#### alloy/config.alloy (committed to repo)
- `discovery.docker` targeting the local Docker socket; relabel to keep only
  the `booking-system` container.
- `loki.source.docker` reads the discovered container logs and forwards to
  `loki.write`.
- Relabel rules add labels: `service="booking-system"`, `env="prod"`,
  `container` name.
- `loki.write` endpoint `url = env("LOKI_URL")` with basic auth
  `username = env("LOKI_USERNAME")`, `password = env("LOKI_PASSWORD")`.
  Compatible with Grafana Cloud Loki or a self-hosted Loki.

### 7. summary
- Needs: `[test, quality, security, performance]`; `if: always()`.
- Writes a single table to `$GITHUB_STEP_SUMMARY` with plain-text status
  (PASS / FAIL / SKIPPED) per job (test, quality, security, performance) plus
  the coverage percentage, giving one consolidated "summary of all the tests".
  No emojis.

## Required GitHub Secrets

| Secret | Purpose |
|---|---|
| `DOCKERHUB_USERNAME` | Docker Hub login / image namespace (value: `bixoloo`) |
| `DOCKERHUB_TOKEN` | Docker Hub access token |
| `EC2_HOST` | EC2 public host/IP |
| `EC2_USER` | SSH user (e.g. `ubuntu`) |
| `EC2_SSH_KEY` | Private SSH key for the EC2 user |
| `EC2_ENV_FILE` | Full prod env file contents (DB/Redis/RabbitMQ/secrets, etc.) |
| `LOKI_URL` | Loki push endpoint (e.g. Grafana Cloud `.../loki/api/v1/push`) |
| `LOKI_USERNAME` | Loki basic-auth user (e.g. Grafana Cloud instance/user ID) |
| `LOKI_PASSWORD` | Loki basic-auth token/API key |

## New Files

- `.github/workflows/ci-cd.yml` — the single workflow.
- `Dockerfile` — multi-stage CGO build on debian-slim runtime.
- `.dockerignore` — exclude tests, docs artifacts, git, local env files.
- `.golangci.yml` — linter configuration.
- `docker-compose.yml` — EC2 orchestration: booking-system + Alloy sidecar.
- `alloy/config.alloy` — Alloy Docker log discovery → Loki pipeline.

## Non-Goals / Deferred

- No hard coverage gate yet (report only; add later once coverage rises).
- No real performance benchmarks yet (placeholder job).
- gosec/Trivy findings are non-blocking (only `govulncheck` blocks).
- No automatic (push-triggered) deployment; EC2 deploy is manual only.

## Success Criteria

- On push/PR: tests run with a formatted summary; the job fails if any test
  fails; quality and `govulncheck` gate the build; coverage + gosec + Trivy +
  performance report into the summary; a final summary job aggregates results.
- On push to `main`: a CGO image builds and publishes to Docker Hub as `latest`
  and the git SHA.
- On manual dispatch: the chosen image tag is pulled and run on EC2 via docker
  compose with prod env vars and ports 7001/7002 exposed, alongside a Grafana
  Alloy sidecar that discovers the app container over the Docker socket and
  ships its stdout/stderr logs to Grafana Loki.
