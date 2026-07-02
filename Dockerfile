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