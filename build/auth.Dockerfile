FROM golang:1.26 AS builder
WORKDIR /app

COPY services/contract/go.mod services/contract/go.sum ./services/contract/
RUN cd services/contract && GOWORK=off go mod download

COPY go.mod go.sum ./
RUN GOWORK=off go mod download

COPY services/auth/go.mod services/auth/go.sum ./services/auth/
RUN cd services/auth && GOWORK=off go mod download

COPY . .
RUN GOWORK=off CGO_ENABLED=0 GOOS=linux go build -o healthcheck ./cmd/healthcheck && \
    GOWORK=off CGO_ENABLED=0 GOOS=linux go build -C ./services/auth -o /app/auth ./cmd/auth

FROM debian:bookworm-slim
WORKDIR /app
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/* \
 && addgroup --system app && adduser --system --ingroup app app

COPY --from=builder /app/auth ./auth
COPY --from=builder /app/healthcheck ./healthcheck
COPY --from=builder /app/services/auth/migrations ./migrations

USER app
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --retries=3 CMD ["./healthcheck"]
CMD ["./auth"]
