FROM golang:1.26 AS builder
WORKDIR /app
COPY go.work go.work.sum ./
COPY go.mod go.sum ./
COPY services/contract/go.mod services/contract/go.sum ./services/contract/
COPY services/subscription/go.mod services/subscription/go.sum ./services/subscription/
RUN go work download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o subscription ./services/subscription/cmd/subscription && \
    CGO_ENABLED=0 GOOS=linux go build -o healthcheck ./cmd/healthcheck

FROM debian:bookworm-slim
WORKDIR /app
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/* \
 && addgroup --system app && adduser --system --ingroup app app

COPY --from=builder /app/subscription ./subscription
COPY --from=builder /app/healthcheck ./healthcheck
COPY --from=builder /app/services/subscription/migrations ./migrations

USER app
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --retries=3 CMD ["./healthcheck"]
CMD ["./subscription"]
