# Kubernetes Manifests — Known Gaps

This document tracks known gaps in the `deploy/k8s/` manifests that are accepted for now and deferred to a later pass. It is not a roadmap of planned features — just a record of tradeoffs made deliberately so they aren't rediscovered as surprises.

---

## A. Shared config values duplicated across per-service ConfigMaps

**What.** Each service (`auth`, `monitoring`, `notification`, `subscription`) has its own `configmap.yaml` under `deploy/k8s/<service>/`. Several values are copy-pasted identically across them instead of defined once:
- `NATS_URL: "nats://nats:4222"` — in `auth`, `monitoring`, `notification`, `subscription`.
- `GRPC_TLS_CA_CERT: "/certs/ca.crt"` — in `auth`, `monitoring`, `subscription`.

**Why.** There is no templating or composition layer (Kustomize base/overlays, Helm values) generating these manifests — each service's YAML is hand-written and self-contained. Changing a shared value (e.g. the NATS URL, if NATS moves or gets TLS) means editing N files by hand and hoping none are missed.

**Decision.** Left as-is for now. Each service is deployed and manifested independently, and with 4 services and a handful of shared keys, the duplication is small enough to manage by hand at this stage.

**Trigger to revisit.** When a shared value needs to change across services (NATS endpoint, cert paths, common labels), when a 5th service is added, or when manifest drift causes an actual incident (one service updated, another missed).

**How (when revisited).** Introduce a Kustomize base with per-service overlays, or move shared keys into a common ConfigMap consumed via `envFrom` by all services, keeping per-service ConfigMaps for values that actually differ.
