# Build stage
FROM golang:1.24-bullseye AS builder
WORKDIR /app

# Install build dependencies
RUN apt-get update && apt-get install -y \
    pkg-config \
    librdkafka-dev \
    && rm -rf /var/lib/apt/lists/*

COPY . .
RUN go build -o validator ./cmd/function

# Runtime stage
FROM debian:bullseye-slim

# Install runtime dependencies
RUN apt-get update && apt-get install -y \
    librdkafka1 \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/validator /validator
ENTRYPOINT ["/validator"]