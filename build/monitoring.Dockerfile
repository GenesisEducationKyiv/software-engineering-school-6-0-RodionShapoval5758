FROM golang:1.26 AS builder
WORKDIR /app

COPY services/contract/go.mod services/contract/go.sum ./services/contract/
RUN cd services/contract && GOWORK=off go mod download

COPY go.mod go.sum ./
RUN GOWORK=off go mod download

COPY services/auth/go.mod services/auth/go.sum ./services/auth/
COPY services/subscription/go.mod services/subscription/go.sum ./services/subscription/
RUN cd services/subscription && GOWORK=off go mod download

COPY services/monitoring/go.mod services/monitoring/go.sum ./services/monitoring/
RUN cd services/monitoring && GOWORK=off go mod download

COPY . .
RUN GOWORK=off CGO_ENABLED=0 GOOS=linux go build -o healthcheck ./cmd/healthcheck && \
    GOWORK=off CGO_ENABLED=0 GOOS=linux go build -C ./services/monitoring -o /app/monitoring ./cmd/monitoring

FROM debian:bookworm-slim
WORKDIR /app
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/* \
 && addgroup --system app && adduser --system --ingroup app app

COPY --from=builder /app/monitoring ./monitoring
COPY --from=builder /app/healthcheck ./healthcheck
COPY --from=builder /app/services/monitoring/migrations ./migrations

USER app
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --retries=3 CMD ["./healthcheck"]
CMD ["./monitoring"]
