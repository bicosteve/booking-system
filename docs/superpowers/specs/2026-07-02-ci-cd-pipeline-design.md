# CI/CD Pipeline Design — booking-system

Date: 2026-07-02
Status: Approved

## Goal

A single GitHub Actions workflow that runs the full CI/CD pipeline for the Go
`booking-system` service: tests with a formatted pass/fail summary, security
scanning, code quality + coverage reporting, a performance placeholder, Docker
image creation, publishing to Docker Hub (main branch only), and a manual EC2
deployment over SSH (`workflow_dispatch` only). A final job summarizes all
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
- Docker image repository: `<DOCKERHUB_USERNAME>/booking-system`.

## Triggers

```yaml
on:
  push:
    branches: ["**"]
  pull_request:
  workflow_dispatch:
    inputs:
      image_tag:
        description: "Image tag to deploy to EC2"
        required: false
        default: "latest"
```

- CI jobs (`test`, `quality`, `security`, `performance`, `summary`) run on
  `push` and `pull_request`.
- `build-and-publish` runs only on `push` to `main`.
- `deploy-ec2` runs only on `workflow_dispatch`.

## Job Graph

```
test ──┬─> quality ──┐
       ├─> security ─┤
       └─> performance ─> summary  (if: always)
                     │
build-and-publish (needs: test, quality, security; if: push && main)
deploy-ec2 (if: workflow_dispatch)
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

### 5. build-and-publish (main only)
- Condition: `github.event_name == 'push' && github.ref == 'refs/heads/main'`.
- Needs: `test`, `quality`, `security`.
- Multi-stage `Dockerfile`:
  - Stage 1 (builder): `golang:1.23-bookworm`, install `librdkafka-dev`,
    `CGO_ENABLED=1 GOOS=linux go build -o /out/bookingapp ./cmd`.
  - Stage 2 (runtime): `debian:bookworm-slim`, install `librdkafka1`,
    `ca-certificates`; copy binary and `docs/`; `EXPOSE 7001 7002`;
    `ENV ENV=prod`; `ENTRYPOINT ["/app/bookingapp"]`.
- `docker/setup-buildx-action`, `docker/login-action` (Docker Hub with
  `DOCKERHUB_USERNAME` / `DOCKERHUB_TOKEN`), `docker/build-push-action` with
  Buildx layer cache.
- Tags pushed: `latest` and short git SHA (`${GITHUB_SHA::7}`).
- After push: Trivy **image** scan (report only) into the summary.

### 6. deploy-ec2 (workflow_dispatch only)
- Condition: `github.event_name == 'workflow_dispatch'`.
- `appleboy/ssh-action` using `EC2_HOST`, `EC2_USER`, `EC2_SSH_KEY`.
- Steps on the server:
  1. Write `EC2_ENV_FILE` secret to `~/booking.env`.
  2. `docker login` with Docker Hub creds.
  3. `docker pull <user>/booking-system:${{ inputs.image_tag }}` (default
     `latest`).
  4. `docker rm -f booking-system || true`.
  5. `docker run -d --name booking-system --restart unless-stopped
     --env-file ~/booking.env -e ENV=prod -p 7001:7001 -p 7002:7002 <image>`.
  6. `docker image prune -f`.
- Report deployment result to the summary.

### 7. summary
- Needs: `[test, quality, security, performance]`; `if: always()`.
- Writes a single table to `$GITHUB_STEP_SUMMARY` with ✅/❌ per job (test,
  quality, security, performance) plus the coverage percentage, giving one
  consolidated "summary of all the tests".

## Required GitHub Secrets

| Secret | Purpose |
|---|---|
| `DOCKERHUB_USERNAME` | Docker Hub login / image namespace |
| `DOCKERHUB_TOKEN` | Docker Hub access token |
| `EC2_HOST` | EC2 public host/IP |
| `EC2_USER` | SSH user (e.g. `ubuntu`) |
| `EC2_SSH_KEY` | Private SSH key for the EC2 user |
| `EC2_ENV_FILE` | Full prod env file contents (DB/Redis/RabbitMQ/secrets, etc.) |

## New Files

- `.github/workflows/ci-cd.yml` — the single workflow.
- `Dockerfile` — multi-stage CGO build on debian-slim runtime.
- `.dockerignore` — exclude tests, docs artifacts, git, local env files.
- `.golangci.yml` — linter configuration.

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
- On manual dispatch: the chosen image tag is pulled and run on EC2 over SSH
  with prod env vars and ports 7001/7002 exposed.
