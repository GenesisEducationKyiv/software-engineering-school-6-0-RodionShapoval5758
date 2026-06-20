FROM golang:1.26 AS builder
WORKDIR /app
COPY go.work go.work.sum ./
COPY go.mod go.sum ./
COPY services/contract/go.mod services/contract/go.sum ./services/contract/
COPY services/monitoring/go.mod services/monitoring/go.sum ./services/monitoring/
RUN go work download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o monitoring ./services/monitoring/cmd/monitoring

FROM debian:bookworm-slim
WORKDIR /app
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/* \
 && addgroup --system app && adduser --system --ingroup app app

COPY --from=builder /app/monitoring ./monitoring
COPY --from=builder /app/services/monitoring/migrations ./migrations

USER app
CMD ["./monitoring"]
