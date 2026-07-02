# CI/CD Pipeline Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a single GitHub Actions workflow plus supporting Docker/Alloy files that test, scan, quality-check, build, publish, and manually deploy the booking-system service with a Grafana Alloy sidecar shipping logs to Loki.

**Architecture:** One workflow file (`.github/workflows/ci-cd.yml`) with seven jobs (test, quality, security, performance, build-and-publish, deploy-ec2, summary). A multi-stage CGO Dockerfile produces a debian-slim runtime image published to Docker Hub. EC2 deploy uses docker compose to run the app alongside a Grafana Alloy sidecar that discovers the app container over the Docker socket and pushes logs to Loki.

**Tech Stack:** GitHub Actions, Go 1.23 (CGO + librdkafka), Docker/Buildx, Docker Hub, golangci-lint, govulncheck, gosec, Trivy, gotestsum, Grafana Alloy, Grafana Loki, docker compose.

## Global Constraints

- Go version: `1.23` (module declares `go 1.23.0`).
- `CGO_ENABLED=1` is mandatory everywhere Go is built or tested (confluent-kafka-go needs librdkafka). Install `librdkafka-dev` on build/test runners; `librdkafka1` on the runtime image.
- Docker image repository: `bixoloo/booking-system`.
- App ports: `7001` (user), `7002` (admin).
- Prod config comes from environment variables (`ENV=prod`), not a file.
- No emojis anywhere in workflow/config files. Use plain-text status words: `PASS`, `FAIL`, `SKIPPED`.
- Triggers: push to `main`, `fix/**`, `feat/**`; pull_request to `main`; workflow_dispatch.
- build-and-publish: builds on the main line (push to main OR PR to main); pushes to Docker Hub only on push to main.
- deploy-ec2: only when `workflow_dispatch` AND ref is `main`.
- Required secrets: `DOCKERHUB_USERNAME` (=bixoloo), `DOCKERHUB_TOKEN`, `EC2_HOST`, `EC2_USER`, `EC2_SSH_KEY`, `EC2_ENV_FILE`, `LOKI_URL`, `LOKI_USERNAME`, `LOKI_PASSWORD`.

---

### Task 1: Docker ignore + linter config + Dockerfile

**Files:**
- Create: `.dockerignore`
- Create: `.golangci.yml`
- Create: `Dockerfile`

**Interfaces:**
- Consumes: nothing.
- Produces: a buildable image tagged locally as `booking-system:test`; ENTRYPOINT `/app/bookingapp`; exposes 7001/7002. Later tasks (build-and-publish, compose) rely on this Dockerfile at repo root and the binary path `/app/bookingapp`.

- [ ] **Step 1: Create `.dockerignore`**

```
.git
.github
docs/superpowers
*_test.go
coverage.out
coverage.html
bin/
.env
.env-example
.env.*
*.md
.idea
```

- [ ] **Step 2: Create `.golangci.yml`**

```yaml
run:
  timeout: 5m
  tests: true

linters:
  disable-all: true
  enable:
    - errcheck
    - gosimple
    - govet
    - ineffassign
    - staticcheck
    - unused
    - gofmt
    - misspell

issues:
  exclude-rules:
    - path: _test\.go
      linters:
        - errcheck
  max-issues-per-linter: 0
  max-same-issues: 0
```

- [ ] **Step 3: Create `Dockerfile` (multi-stage, CGO)**

```dockerfile
# syntax=docker/dockerfile:1

# ---- Build stage ----
FROM golang:1.23-bookworm AS builder

RUN apt-get update \
    && apt-get install -y --no-install-recommends librdkafka-dev pkg-config \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=linux go build -o /out/bookingapp ./cmd

# ---- Runtime stage ----
FROM debian:bookworm-slim

RUN apt-get update \
    && apt-get install -y --no-install-recommends librdkafka1 ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /out/bookingapp /app/bookingapp
COPY --from=builder /src/docs /app/docs
COPY --from=builder /src/env.dev.toml /app/env.dev.toml

ENV ENV=prod

EXPOSE 7001 7002

ENTRYPOINT ["/app/bookingapp"]
```

- [ ] **Step 4: Build the image locally to verify it compiles**

Run: `docker build -t booking-system:test .`
Expected: build completes successfully; final line shows the image was written / tagged `booking-system:test`.

- [ ] **Step 5: Commit**

```bash
git add .dockerignore .golangci.yml Dockerfile
git commit -m "build: add Dockerfile, .dockerignore and golangci config"
```

---

### Task 2: Alloy config + docker-compose for EC2

**Files:**
- Create: `alloy/config.alloy`
- Create: `docker-compose.yml`

**Interfaces:**
- Consumes: the image `bixoloo/booking-system` (built in Task 1 / published in Task 4).
- Produces: `docker-compose.yml` at repo root defining services `booking-system` and `alloy`; the deploy job (Task 6) copies both files to EC2 and runs `docker compose`. Compose reads `IMAGE_TAG` env var (default `latest`), `booking.env`, and `alloy.env`.

- [ ] **Step 1: Create `alloy/config.alloy`**

```alloy
// Grafana Alloy config: discover the booking-system container via the Docker
// socket, read its logs, and forward them to Grafana Loki.

discovery.docker "containers" {
  host = "unix:///var/run/docker.sock"
}

discovery.relabel "booking" {
  targets = discovery.docker.containers.targets

  // Keep only the booking-system container.
  rule {
    source_labels = ["__meta_docker_container_name"]
    regex         = "/?booking-system"
    action        = "keep"
  }

  // Expose the container name as a label.
  rule {
    source_labels = ["__meta_docker_container_name"]
    regex         = "/?(.*)"
    target_label  = "container"
  }

  // Static service + environment labels.
  rule {
    target_label = "service"
    replacement  = "booking-system"
  }

  rule {
    target_label = "env"
    replacement  = "prod"
  }
}

loki.source.docker "booking" {
  host       = "unix:///var/run/docker.sock"
  targets    = discovery.relabel.booking.output
  forward_to = [loki.write.default.receiver]
}

loki.write "default" {
  endpoint {
    url = sys.env("LOKI_URL")

    basic_auth {
      username = sys.env("LOKI_USERNAME")
      password = sys.env("LOKI_PASSWORD")
    }
  }
}
```

- [ ] **Step 2: Create `docker-compose.yml`**

```yaml
services:
  booking-system:
    image: bixoloo/booking-system:${IMAGE_TAG:-latest}
    container_name: booking-system
    restart: unless-stopped
    env_file:
      - booking.env
    environment:
      ENV: "prod"
    ports:
      - "7001:7001"
      - "7002:7002"
    networks:
      - bookingnet

  alloy:
    image: grafana/alloy:latest
    container_name: alloy
    restart: unless-stopped
    env_file:
      - alloy.env
    command:
      - "run"
      - "/etc/alloy/config.alloy"
      - "--server.http.listen-addr=0.0.0.0:12345"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - ./config.alloy:/etc/alloy/config.alloy:ro
    depends_on:
      - booking-system
    networks:
      - bookingnet

networks:
  bookingnet:
    driver: bridge
```

- [ ] **Step 3: Validate compose file syntax**

Run: `docker compose -f docker-compose.yml config --quiet && echo COMPOSE_OK`
Expected: prints `COMPOSE_OK` with no errors (a warning that `IMAGE_TAG`/env files are unset is acceptable; if `config` errors on missing env_file, run `touch booking.env alloy.env` first, validate, then `rm booking.env alloy.env`).

- [ ] **Step 4: Commit**

```bash
git add alloy/config.alloy docker-compose.yml
git commit -m "build: add Alloy sidecar config and docker-compose for EC2"
```

---

### Task 3: Workflow skeleton — triggers + test + quality + performance jobs

**Files:**
- Create: `.github/workflows/ci-cd.yml`

**Interfaces:**
- Consumes: repo Go module; `librdkafka-dev`.
- Produces: jobs named `test`, `quality`, `performance`. `test` uploads coverage artifacts and writes `coverage_pct` to `$GITHUB_OUTPUT` (id `cov`) as output `pct`. Later `summary` job (Task 7) reads `needs.test.outputs.pct` and job results. `quality`, `security`, `performance` all `needs: test`.

- [ ] **Step 1: Create `.github/workflows/ci-cd.yml` with triggers and the test job**

```yaml
name: CI-CD

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

permissions:
  contents: read

env:
  GO_VERSION: "1.23"
  IMAGE_NAME: bixoloo/booking-system

jobs:
  test:
    name: Test
    runs-on: ubuntu-latest
    outputs:
      pct: ${{ steps.cov.outputs.pct }}
    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Install librdkafka
        run: |
          sudo apt-get update
          sudo apt-get install -y --no-install-recommends librdkafka-dev

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: ${{ env.GO_VERSION }}
          cache: true

      - name: Install gotestsum
        run: go install gotest.tools/gotestsum@latest

      - name: Run tests
        env:
          CGO_ENABLED: "1"
        run: |
          set -o pipefail
          "$(go env GOPATH)/bin/gotestsum" \
            --format testname \
            --jsonfile test-output.json \
            --junitfile test-report.xml \
            -- -race -covermode=atomic -coverprofile=coverage.out ./...

      - name: Build coverage report
        if: always()
        run: |
          go tool cover -func=coverage.out > coverage-func.txt || true
          go tool cover -html=coverage.out -o coverage.html || true

      - name: Compute total coverage
        id: cov
        if: always()
        run: |
          PCT=$(go tool cover -func=coverage.out 2>/dev/null | awk '/^total:/ {print $3}')
          if [ -z "$PCT" ]; then PCT="0.0%"; fi
          echo "pct=$PCT" >> "$GITHUB_OUTPUT"
          echo "Total coverage: $PCT"

      - name: Write test summary
        if: always()
        run: |
          {
            echo "## Test Results"
            echo ""
            echo "Total coverage: ${{ steps.cov.outputs.pct }}"
            echo ""
            echo "Per-package coverage:"
            echo ""
            echo '```'
            cat coverage-func.txt 2>/dev/null || echo "no coverage data"
            echo '```'
          } >> "$GITHUB_STEP_SUMMARY"

      - name: Upload coverage artifacts
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: coverage
          path: |
            coverage.out
            coverage.html
            coverage-func.txt
            test-report.xml
```

- [ ] **Step 2: Append the quality job**

```yaml
  quality:
    name: Code Quality
    runs-on: ubuntu-latest
    needs: test
    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Install librdkafka
        run: |
          sudo apt-get update
          sudo apt-get install -y --no-install-recommends librdkafka-dev

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: ${{ env.GO_VERSION }}
          cache: true

      - name: Check formatting
        run: |
          unformatted=$(gofmt -l .)
          if [ -n "$unformatted" ]; then
            echo "The following files are not gofmt-formatted:"
            echo "$unformatted"
            exit 1
          fi
          echo "All files are gofmt-formatted"

      - name: Go vet
        env:
          CGO_ENABLED: "1"
        run: go vet ./...

      - name: golangci-lint
        uses: golangci/golangci-lint-action@v6
        with:
          version: latest
          args: --timeout=5m
```

- [ ] **Step 3: Append the performance placeholder job**

```yaml
  performance:
    name: Performance
    runs-on: ubuntu-latest
    needs: test
    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Install librdkafka
        run: |
          sudo apt-get update
          sudo apt-get install -y --no-install-recommends librdkafka-dev

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: ${{ env.GO_VERSION }}
          cache: true

      - name: Run benchmarks (placeholder)
        env:
          CGO_ENABLED: "1"
        run: |
          if grep -rlq "^func Benchmark" --include="*_test.go" .; then
            echo "Running benchmarks"
            go test -run '^$' -bench=. -benchmem ./... | tee bench.txt
          else
            echo "No benchmarks yet - performance placeholder" | tee bench.txt
          fi

      - name: Write performance summary
        if: always()
        run: |
          {
            echo "## Performance"
            echo ""
            echo '```'
            cat bench.txt 2>/dev/null || echo "no benchmark output"
            echo '```'
          } >> "$GITHUB_STEP_SUMMARY"
```

- [ ] **Step 4: Validate YAML syntax**

Run: `python3 -c "import yaml,sys; yaml.safe_load(open('.github/workflows/ci-cd.yml')); print('YAML_OK')"`
Expected: prints `YAML_OK`.

- [ ] **Step 5: Commit**

```bash
git add .github/workflows/ci-cd.yml
git commit -m "ci: add test, quality and performance jobs"
```

---

### Task 4: Security job

**Files:**
- Modify: `.github/workflows/ci-cd.yml` (append `security` job under `jobs:`)

**Interfaces:**
- Consumes: repo Go module; `librdkafka-dev`.
- Produces: job named `security`; `needs: test`. govulncheck blocks; gosec + Trivy are non-blocking.

- [ ] **Step 1: Append the security job (after the performance job, before end of file)**

```yaml
  security:
    name: Security
    runs-on: ubuntu-latest
    needs: test
    permissions:
      contents: read
      security-events: write
    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Install librdkafka
        run: |
          sudo apt-get update
          sudo apt-get install -y --no-install-recommends librdkafka-dev

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: ${{ env.GO_VERSION }}
          cache: true

      - name: govulncheck (blocking)
        env:
          CGO_ENABLED: "1"
        run: |
          go install golang.org/x/vuln/cmd/govulncheck@latest
          "$(go env GOPATH)/bin/govulncheck" ./...

      - name: gosec (report only)
        continue-on-error: true
        uses: securego/gosec@master
        with:
          args: "-no-fail -fmt sarif -out gosec.sarif ./..."

      - name: Upload gosec SARIF
        if: always()
        continue-on-error: true
        uses: actions/upload-artifact@v4
        with:
          name: gosec-sarif
          path: gosec.sarif

      - name: Trivy filesystem scan (report only)
        continue-on-error: true
        uses: aquasecurity/trivy-action@0.24.0
        with:
          scan-type: fs
          scan-ref: .
          severity: HIGH,CRITICAL
          exit-code: "0"
          format: table
```

- [ ] **Step 2: Validate YAML syntax**

Run: `python3 -c "import yaml,sys; yaml.safe_load(open('.github/workflows/ci-cd.yml')); print('YAML_OK')"`
Expected: prints `YAML_OK`.

- [ ] **Step 3: Commit**

```bash
git add .github/workflows/ci-cd.yml
git commit -m "ci: add security job (govulncheck, gosec, trivy)"
```

---

### Task 5: build-and-publish job

**Files:**
- Modify: `.github/workflows/ci-cd.yml` (append `build-and-publish` job)

**Interfaces:**
- Consumes: `Dockerfile` (Task 1); secrets `DOCKERHUB_USERNAME`, `DOCKERHUB_TOKEN`; `needs: [test, quality, security]`.
- Produces: job named `build-and-publish`; on push to main pushes `bixoloo/booking-system:latest` and `bixoloo/booking-system:<sha>`.

- [ ] **Step 1: Append the build-and-publish job**

```yaml
  build-and-publish:
    name: Build and Publish
    runs-on: ubuntu-latest
    needs: [test, quality, security]
    if: >
      (github.event_name == 'push' && github.ref == 'refs/heads/main') ||
      (github.event_name == 'pull_request' && github.base_ref == 'main')
    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Short SHA
        id: vars
        run: echo "sha=${GITHUB_SHA::7}" >> "$GITHUB_OUTPUT"

      - name: Set up Buildx
        uses: docker/setup-buildx-action@v3

      - name: Determine push
        id: push
        run: |
          if [ "${{ github.event_name }}" = "push" ] && [ "${{ github.ref }}" = "refs/heads/main" ]; then
            echo "enabled=true" >> "$GITHUB_OUTPUT"
          else
            echo "enabled=false" >> "$GITHUB_OUTPUT"
          fi

      - name: Log in to Docker Hub
        if: steps.push.outputs.enabled == 'true'
        uses: docker/login-action@v3
        with:
          username: ${{ secrets.DOCKERHUB_USERNAME }}
          password: ${{ secrets.DOCKERHUB_TOKEN }}

      - name: Build and push
        uses: docker/build-push-action@v6
        with:
          context: .
          push: ${{ steps.push.outputs.enabled }}
          load: ${{ steps.push.outputs.enabled == 'false' }}
          tags: |
            ${{ env.IMAGE_NAME }}:latest
            ${{ env.IMAGE_NAME }}:${{ steps.vars.outputs.sha }}
          cache-from: type=gha
          cache-to: type=gha,mode=max

      - name: Trivy image scan (report only)
        continue-on-error: true
        uses: aquasecurity/trivy-action@0.24.0
        with:
          image-ref: ${{ env.IMAGE_NAME }}:${{ steps.vars.outputs.sha }}
          severity: HIGH,CRITICAL
          exit-code: "0"
          format: table

      - name: Write build summary
        if: always()
        run: |
          {
            echo "## Build and Publish"
            echo ""
            echo "Image: ${{ env.IMAGE_NAME }}"
            echo "Tags: latest, ${{ steps.vars.outputs.sha }}"
            echo "Pushed: ${{ steps.push.outputs.enabled }}"
          } >> "$GITHUB_STEP_SUMMARY"
```

- [ ] **Step 2: Validate YAML syntax**

Run: `python3 -c "import yaml,sys; yaml.safe_load(open('.github/workflows/ci-cd.yml')); print('YAML_OK')"`
Expected: prints `YAML_OK`.

- [ ] **Step 3: Commit**

```bash
git add .github/workflows/ci-cd.yml
git commit -m "ci: add build-and-publish job (build on main line, push on main)"
```

---

### Task 6: deploy-ec2 job

**Files:**
- Modify: `.github/workflows/ci-cd.yml` (append `deploy-ec2` job)

**Interfaces:**
- Consumes: `docker-compose.yml` + `alloy/config.alloy` (Task 2); secrets `EC2_HOST`, `EC2_USER`, `EC2_SSH_KEY`, `EC2_ENV_FILE`, `DOCKERHUB_USERNAME`, `DOCKERHUB_TOKEN`, `LOKI_URL`, `LOKI_USERNAME`, `LOKI_PASSWORD`; input `image_tag`.
- Produces: job named `deploy-ec2`; runs docker compose on EC2.

- [ ] **Step 1: Append the deploy-ec2 job**

```yaml
  deploy-ec2:
    name: Deploy to EC2
    runs-on: ubuntu-latest
    if: github.event_name == 'workflow_dispatch' && github.ref == 'refs/heads/main'
    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Copy deploy files to EC2
        uses: appleboy/scp-action@v0.1.7
        with:
          host: ${{ secrets.EC2_HOST }}
          username: ${{ secrets.EC2_USER }}
          key: ${{ secrets.EC2_SSH_KEY }}
          source: "docker-compose.yml,alloy/config.alloy"
          target: "~/booking"
          overwrite: true

      - name: Deploy on EC2
        uses: appleboy/ssh-action@v1.0.3
        env:
          IMAGE_TAG: ${{ github.event.inputs.image_tag }}
          DOCKERHUB_USERNAME: ${{ secrets.DOCKERHUB_USERNAME }}
          DOCKERHUB_TOKEN: ${{ secrets.DOCKERHUB_TOKEN }}
          EC2_ENV_FILE: ${{ secrets.EC2_ENV_FILE }}
          LOKI_URL: ${{ secrets.LOKI_URL }}
          LOKI_USERNAME: ${{ secrets.LOKI_USERNAME }}
          LOKI_PASSWORD: ${{ secrets.LOKI_PASSWORD }}
        with:
          host: ${{ secrets.EC2_HOST }}
          username: ${{ secrets.EC2_USER }}
          key: ${{ secrets.EC2_SSH_KEY }}
          envs: IMAGE_TAG,DOCKERHUB_USERNAME,DOCKERHUB_TOKEN,EC2_ENV_FILE,LOKI_URL,LOKI_USERNAME,LOKI_PASSWORD
          script: |
            set -e
            cd ~/booking
            # scp nests the source path; move config.alloy up if needed
            if [ -f alloy/config.alloy ]; then cp alloy/config.alloy ./config.alloy; fi
            printf '%s' "$EC2_ENV_FILE" > booking.env
            {
              echo "LOKI_URL=$LOKI_URL"
              echo "LOKI_USERNAME=$LOKI_USERNAME"
              echo "LOKI_PASSWORD=$LOKI_PASSWORD"
            } > alloy.env
            echo "$DOCKERHUB_TOKEN" | docker login -u "$DOCKERHUB_USERNAME" --password-stdin
            export IMAGE_TAG="${IMAGE_TAG:-latest}"
            docker compose pull
            docker compose up -d
            docker image prune -f
            docker logout

      - name: Write deploy summary
        if: always()
        run: |
          {
            echo "## Deploy to EC2"
            echo ""
            echo "Image tag: ${{ github.event.inputs.image_tag }}"
            echo "Result: ${{ job.status }}"
          } >> "$GITHUB_STEP_SUMMARY"
```

- [ ] **Step 2: Validate YAML syntax**

Run: `python3 -c "import yaml,sys; yaml.safe_load(open('.github/workflows/ci-cd.yml')); print('YAML_OK')"`
Expected: prints `YAML_OK`.

- [ ] **Step 3: Commit**

```bash
git add .github/workflows/ci-cd.yml
git commit -m "cd: add manual EC2 deploy job with Alloy sidecar"
```

---

### Task 7: summary job + secrets documentation

**Files:**
- Modify: `.github/workflows/ci-cd.yml` (append `summary` job)
- Create: `docs/ci-cd-secrets.md`

**Interfaces:**
- Consumes: results/outputs of `test`, `quality`, `security`, `performance`.
- Produces: job named `summary`; a consolidated status table in the run summary.

- [ ] **Step 1: Append the summary job**

```yaml
  summary:
    name: Summary
    runs-on: ubuntu-latest
    needs: [test, quality, security, performance]
    if: always()
    steps:
      - name: Aggregate results
        run: |
          {
            echo "## Pipeline Summary"
            echo ""
            echo "| Job | Status |"
            echo "| --- | --- |"
            echo "| Test | ${{ needs.test.result == 'success' && 'PASS' || (needs.test.result == 'skipped' && 'SKIPPED' || 'FAIL') }} |"
            echo "| Code Quality | ${{ needs.quality.result == 'success' && 'PASS' || (needs.quality.result == 'skipped' && 'SKIPPED' || 'FAIL') }} |"
            echo "| Security | ${{ needs.security.result == 'success' && 'PASS' || (needs.security.result == 'skipped' && 'SKIPPED' || 'FAIL') }} |"
            echo "| Performance | ${{ needs.performance.result == 'success' && 'PASS' || (needs.performance.result == 'skipped' && 'SKIPPED' || 'FAIL') }} |"
            echo ""
            echo "Total coverage: ${{ needs.test.outputs.pct }}"
          } >> "$GITHUB_STEP_SUMMARY"

      - name: Fail if any required job failed
        if: needs.test.result == 'failure' || needs.quality.result == 'failure' || needs.security.result == 'failure'
        run: |
          echo "One or more required jobs failed"
          exit 1
```

- [ ] **Step 2: Create `docs/ci-cd-secrets.md`**

```markdown
# CI/CD Required GitHub Secrets

Set these under Settings -> Secrets and variables -> Actions.

| Secret | Purpose |
| --- | --- |
| DOCKERHUB_USERNAME | Docker Hub login / image namespace (value: bixoloo) |
| DOCKERHUB_TOKEN | Docker Hub access token |
| EC2_HOST | EC2 public host or IP |
| EC2_USER | SSH user (e.g. ubuntu) |
| EC2_SSH_KEY | Private SSH key for the EC2 user |
| EC2_ENV_FILE | Full prod env file contents for the app (DB_HOST, DB_USER, DB_PASSWORD, DB_PORT, DB_SCHEMA, REDIS_ADDRESS, REDIS_PORT, REDIS_DB, REDIS_PASSWORD, REDIS_NAME, RABBIT_HOST, RABBIT_PORT, RABBIT_USER, RABBIT_PASSWORD, RABBIT_VHOST, RABBIT_QUEUE, RABBITMQ_STATUS, KAFKA_STATUS, HTTP_PORT, ADMIN_PORT, CONTENT_TYPE, API_PATH, JWT_SECRET, SENDGRID_KEY, MAIL_FROM, AT_KEY, APP_USERNAME, PP_CLIENT_ID, PP_SECRET, STRIPE_NAME, STRIPE_SECRET, STRIPE_PUB_KEY, STRIPE_SUCCESS_URL, STRIPE_CANCEL_URL, LOGGER_FOLDER) |
| LOKI_URL | Loki push endpoint (e.g. https://<id>.grafana.net/loki/api/v1/push) |
| LOKI_USERNAME | Loki basic-auth user (Grafana Cloud instance/user id) |
| LOKI_PASSWORD | Loki basic-auth token / API key |

## Notes

- The EC2 server must have Docker and the docker compose plugin installed.
- The app image is bixoloo/booking-system, published on push to main as
  `latest` and the short git SHA.
- Manual deploy: Actions -> CI-CD -> Run workflow (must be run from the main
  branch), optionally set image_tag (defaults to latest).
```

- [ ] **Step 3: Validate YAML syntax**

Run: `python3 -c "import yaml,sys; yaml.safe_load(open('.github/workflows/ci-cd.yml')); print('YAML_OK')"`
Expected: prints `YAML_OK`.

- [ ] **Step 4: Commit**

```bash
git add .github/workflows/ci-cd.yml docs/ci-cd-secrets.md
git commit -m "ci: add summary job and document required secrets"
```

---

## Self-Review Notes

- Spec coverage: tests+formatted summary (Task 3), 100%-pass-or-fail via gotestsum + summary fail-gate (Task 3/7), security scanning (Task 4), quality + coverage report (Task 3 coverage, Task 3 quality), performance placeholder (Task 3), image creation (Task 1/5), publish to Docker Hub on main (Task 5), manual EC2 deploy via workflow_dispatch on main (Task 6), summary of all tests (Task 7), Alloy sidecar to Loki (Task 2/6). All covered.
- Trigger/branch rules and no-emoji constraint captured in Global Constraints and enforced in the job `if:` conditions and plain-text summaries.
- Image name `bixoloo/booking-system` consistent across Tasks 1, 2, 5, 6.
