# Architecture Patterns

Diagrams showing how each pattern would look if applied to this system.
The current implementation uses none of these — they are reference designs for future scaling decisions.

---

## Target Architecture

Full production architecture combining API Gateway, BFFs, service mesh with sidecars, and mixed communication patterns.

```mermaid
graph TD
    ExtClient["External Client\n(web / mobile)"]
    AdminClient["Admin Client\n(internal tooling)"]

    subgraph "Inbound Layer"
        GW["API Gateway\n(Kong / Traefik)\nTLS termination, rate limiting, auth"]
    end

    subgraph "BFF Layer"
        RESTBFF["REST BFF\n(cmd/restbff)\nJSON shaping, pagination\nfor external clients"]
        GRPCBFF["gRPC BFF\n(cmd/grpcbff)\nProtobuf\nfor internal tooling"]
    end

    subgraph "Service Mesh (Istio — Envoy sidecars)"
        subgraph "Subscription Pod"
            SUBSC["Envoy Sidecar\n(mTLS, tracing, retries)"]
            SUB["Subscription Service\n(use cases + DB access)"]
            SUBSC <-->|localhost :8080| SUB
        end

        subgraph "Monitoring Pod"
            MONSC["Envoy Sidecar\n(mTLS, tracing, retries)"]
            MON["Monitoring Service\n(GitHub scanner)"]
            MONSC <-->|localhost :8080| MON
        end

        subgraph "Notification Pod"
            NOTIFSC["Envoy Sidecar\n(mTLS, tracing, retries)"]
            NOTIF["Notification Service\n(email sender)"]
            NOTIFSC <-->|localhost :8080| NOTIF
        end
    end

    CP["Istio Control Plane\n(certificate rotation,\nrouting config via xDS)"]

    NATS[("NATS JetStream\n(async events)")]
    DB[(PostgreSQL\nshared)]
    MAIL["SMTP\n(email delivery)"]

    ExtClient -->|HTTPS| GW
    AdminClient -->|HTTPS| GW

    GW -->|HTTP JSON| RESTBFF
    GW -->|HTTP Protobuf| GRPCBFF

    RESTBFF -->|gRPC internal| SUBSC
    GRPCBFF -->|gRPC internal| SUBSC

    MONSC -->|gRPC P2P\nListTrackedRepos| SUBSC

    SUB -->|publish: SubscriptionConfirmed\nReleaseFound| NATS
    MON -->|publish: ReleaseFound| NATS
    NATS -->|consume: SubscriptionConfirmed\nReleaseFound| NOTIFSC

    SUB --> DB
    MON --> DB

    NOTIF --> MAIL

    CP -.->|configures sidecars\nvia xDS API| SUBSC
    CP -.->|configures sidecars\nvia xDS API| MONSC
    CP -.->|configures sidecars\nvia xDS API| NOTIFSC
```

### Communication types

| Path | Protocol | Pattern |
|---|---|---|
| External client → API Gateway | HTTPS | inbound |
| API Gateway → BFFs | HTTP/gRPC | inbound routing |
| BFFs → Subscription sidecar | gRPC | synchronous P2P |
| Monitoring → Subscription sidecar | gRPC | synchronous P2P |
| Subscription → NATS | publish | async event |
| Monitoring → NATS | publish | async event |
| NATS → Notification | consume | async event |
| All internal traffic | mTLS via Envoy | service mesh |

### Responsibilities per layer

**API Gateway** — TLS termination, rate limiting, auth validation, routing to the correct BFF.

**BFFs** — protocol and response shaping per client type. No business logic.

**Envoy sidecars** — mTLS between all internal services, distributed tracing headers, retries, circuit breaking. Transparent to application code.

**Istio control plane** — manages all sidecar config and certificate rotation centrally via xDS.

**NATS JetStream** — decouples subscription and monitoring from notification. Async, durable, guaranteed delivery.

---

## Current Architecture

```mermaid
graph TD
    Client["External Client"]
    MON["Monitoring Service"]

    subgraph "Subscription Binary"
        REST["REST Handler\n(chi router, :8080)"]
        GRPC["gRPC Server\n(plain grpc, :50051)\nCatalogService"]
        UC["Use Cases\n(subscribe, confirm, unsubscribe, list)"]
        CatalogUC["Catalog Use Case\n(listTracked)"]
    end

    DB[(PostgreSQL)]

    Client -->|HTTP JSON| REST
    MON -->|gRPC plaintext\nListTrackedRepos| GRPC
    REST --> UC
    GRPC --> CatalogUC
    UC --> DB
    CatalogUC --> DB
```

One binary, two ports. REST on :8080 serves external clients. Internal-only CatalogService gRPC on :50051 serves monitoring with a single RPC. gRPC reaches only the catalog use case — no access to subscription operations.

---

## API Gateway

A single entry point handles cross-cutting concerns. The subscription service exposes both REST and gRPC interfaces internally — the gateway routes to each.

```mermaid
graph TD
    Client["External Client"]
    MON["Monitoring Service"]

    GW["API Gateway\n(auth, rate limiting, routing)"]

    subgraph "Subscription Service"
        REST["REST Interface\n(chi router)"]
        GRPC["gRPC Interface\n(plain grpc)"]
        UC["Use Cases"]
    end

    DB[(PostgreSQL\nshared)]
    NOTIF["Notification Service"]

    Client -->|HTTP| GW
    MON -->|gRPC plaintext| GW
    GW -->|routes /api/*| REST
    GW -->|routes /catalog.v1.*| GRPC
    REST --> UC
    GRPC --> UC
    GW -->|routes internal| NOTIF
    UC --> DB
    NOTIF --> DB
    MON --> DB
```

Auth middleware (`ApiKey` check) and rate limiting move out of the subscription service into the gateway. Both interfaces trust all inbound traffic as already authenticated.

---

## Backend for Frontend (BFF)

Each client type gets a dedicated interface binary shaped for its needs. Both forward to the same subscription service which owns the business logic.

```mermaid
graph TD
    WebClient["External Client\n(web / mobile)"]
    MON["Monitoring Service\n(internal)"]

    RESTBFF["REST BFF\n(chi router)\nJSON, pagination, external auth"]
    GRPCBFF["gRPC BFF\n(plain grpc)\nProtobuf, internal auth"]

    subgraph "Subscription Service"
        UC["Use Cases\n(subscribe, confirm, unsubscribe, list)"]
    end

    DB[(PostgreSQL\nshared)]

    WebClient -->|HTTP JSON| RESTBFF
    MON -->|HTTP Protobuf| GRPCBFF
    RESTBFF -->|internal gRPC| UC
    GRPCBFF -->|internal gRPC| UC
    UC --> DB
```

The REST BFF handles pagination and response shaping for external consumers. The gRPC BFF is a thin pass-through optimised for monitoring's `ListTrackedRepos` call pattern. Neither contains business logic.

---

## Layered Decomposition

Interface, business logic, and persistence are independently deployable binaries. Each scales on its own bottleneck.

```mermaid
graph TD
    Client["External Client"]
    MON["Monitoring Service"]

    subgraph "Interface Layer"
        RESTL["REST Binary\ncmd/rest\n(chi router only)"]
        GRPCL["gRPC Binary\ncmd/grpc\n(plain grpc only)"]
    end

    SUBL["Subscription Binary\ncmd/subscription\n(use cases + DB)"]
    DB[(PostgreSQL\nshared)]

    Client -->|HTTP JSON| RESTL
    MON -->|HTTP Protobuf| GRPCL
    RESTL -->|gRPC internal| SUBL
    GRPCL -->|gRPC internal| SUBL
    SUBL --> DB
    MON --> DB
```

Interface binaries are stateless — scale horizontally freely. The subscription binary is also stateless (all state in PostgreSQL) so it scales independently. REST and gRPC interfaces deploy and scale separately from each other and from the business logic.

---

## Sidecar

A helper process runs alongside each service in the same pod. It handles infrastructure concerns so the service code stays pure business logic.

```mermaid
graph TD
    subgraph "Subscription Pod"
        SC["Sidecar\n(TLS termination, auth,\nmetrics scraping, retries)"]
        subgraph "Subscription Service"
            REST["REST Interface\n(:8080)"]
            GRPC["gRPC Interface\n(plain grpc, :50051)"]
            UC["Use Cases"]
        end
        SC -->|localhost :8080| REST
        SC -->|localhost :50051| GRPC
        REST --> UC
        GRPC --> UC
    end

    subgraph "Monitoring Pod"
        MON["Monitoring Service\n(scan logic only)"]
        MSC["Sidecar\n(TLS, metrics)"]
        MSC -->|localhost| MON
    end

    Client["External Client"]
    DB[(PostgreSQL\nshared)]

    Client -->|HTTPS| SC
    UC --> DB
    MON --> DB
```

Auth and Prometheus metrics collection move into the sidecar. Both REST (:8080) and gRPC (:50051) interfaces are reachable through it. The subscription service has no knowledge of auth or observability infrastructure.

---

## Ambassador

A specific sidecar that proxies outbound calls from monitoring to the subscription gRPC interface. Handles retries, circuit breaking, and service discovery.

```mermaid
graph TD
    subgraph "Monitoring Pod"
        MON["Monitoring Service\ncalls localhost:50051"]
        AMB["Ambassador Sidecar\n(retries, circuit breaker,\nTLS, service discovery)"]
        MON -->|localhost:50051| AMB
    end

    subgraph "Subscription Service"
        GRPC["gRPC Interface\n(plain grpc, :50051)\nCatalogService"]
        UC["Use Cases"]
        GRPC --> UC
    end

    DB[(PostgreSQL\nshared)]

    AMB -->|resolves + forwards| GRPC
    UC --> DB
    MON --> DB
```

Instead of monitoring hardcoding `cfg.SubscriptionGRPCAddr` and managing retries itself, it dials `localhost:50051`. The ambassador resolves the real address of the subscription gRPC interface and handles all resilience concerns. If the service moves or is rebalanced, only the ambassador config changes — monitoring code is untouched.
