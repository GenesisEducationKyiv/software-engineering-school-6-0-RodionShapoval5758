FROM golang:1.26 AS builder
WORKDIR /app

COPY services/contract/go.mod services/contract/go.sum ./services/contract/
COPY services/notification/go.mod services/notification/go.sum ./services/notification/

RUN cd services/contract && go mod download
RUN cd services/notification && GOWORK=off go mod download

COPY services/contract/ ./services/contract/
COPY services/notification/ ./services/notification/

RUN GOWORK=off CGO_ENABLED=0 GOOS=linux go build -C ./services/notification -o /app/notification ./cmd/notification

FROM debian:bookworm-slim
WORKDIR /app
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/* \
 && addgroup --system app && adduser --system --ingroup app app

COPY --from=builder /app/notification ./notification

USER app
CMD ["./notification"]
