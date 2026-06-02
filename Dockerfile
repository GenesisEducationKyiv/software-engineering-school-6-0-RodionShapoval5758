FROM golang:1.26 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o api ./cmd/api && \
    CGO_ENABLED=0 GOOS=linux go build -o healthcheck ./cmd/healthcheck

FROM debian:bookworm-slim
WORKDIR /app
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/* \
 && addgroup --system app && adduser --system --ingroup app app

COPY --from=builder /app/api ./api
COPY --from=builder /app/healthcheck ./healthcheck
COPY --from=builder /app/migrations ./migrations

USER app
EXPOSE 8080
CMD ["./api"]