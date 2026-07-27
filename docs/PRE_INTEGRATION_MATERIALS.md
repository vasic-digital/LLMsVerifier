# LLMsVerifier — Pre-Integration Materials

**Revision:** 1
**Last modified:** 2026-07-15T16:05:00Z
**Purpose:** Consolidated pre-integration materials (the gate that must be satisfied BEFORE any integration/deployment work starts). Every claim below is grounded in this repository's real files with `file:line` citations (§11.4.6 — no guessing; genuine unknowns are marked `UNKNOWN:`).

> Authoring note: this file was assembled from bounded `grep -n` / `sed -n` probes of the repository (the large `ACP_*.md` design docs, ~7–29 KB each, are cross-referenced, not re-read in full). It consolidates existing in-tree materials; it does not restate them.

## 1. Purpose / What it is

LLMsVerifier ("LLM Verifier") is a Go framework for **verifying and benchmarking LLM models/providers** and serving that verification via an API + CLI + TUI. Per `helix-deps.yaml`, it is "the single source of truth for LLM model / provider / verification metadata." The root module is `llmsverifier` (`go.mod:1`, `go 1.25.3`), and the runnable application lives in the inner Go module `digital.vasic.llmsverifier` under `llm-verifier/`.

The extensive `README_IMPLEMENTATION_PLANS.md` + `ACP_*.md` set are **implementation-plan / design documents** (e.g. `README_IMPLEMENTATION_PLANS.md:1-20` describes a "36-week implementation plan" toward "100% project completion"); they describe roadmap and completion targets, not solely shipped state. Treat feature claims in those planning docs as intent unless corroborated by source.

## 2. Architecture overview

- **Language/stack:** Go. Root module `llmsverifier` (`go.mod:1`); inner app module `digital.vasic.llmsverifier` in `llm-verifier/` (its own `go.mod`).
- **Top-level layout (real dirs):** `llm-verifier/` (the app), `internal/` (incl. `internal/benchmark/`), `challenges/`, `sdk/`, `mobile/`, `website/` + `Website/`, `helm/`, `k8s/`, `monitoring/`, `configs/`, `specs/`, `examples/`, `video-course/`, `Assets/`, `tests/`, `test_results/`, `reports/`, `scripts/`, `upstreams/`.
- **API server:** `llm-verifier/api/server.go` — registers `/api/health` → `HealthHandler` (`server.go:37,50`) and serves via `ListenAndServe()` (`server.go:65`). Swagger docs at `llm-verifier/api/docs/docs.go` advertise a `/health` path (`docs.go:346`).
- **CLI/TUI:** `llm-verifier/cmd/main.go` (cobra-style root command; default `--server http://localhost:8080`, `cmd/main.go:42,138,188`) and `llm-verifier/cmd/tui/main.go` (`--server http://localhost:8080`, `cmd/tui/main.go:15`).
- **Additional served surfaces** (separate servers/ports within the app):
  - `llm-verifier/enhanced/enterprise/api.go` — HTTP server default `Addr: ":8080"` (`api.go:150,161`), health at `/api/enterprise/health` (`api.go:215`).
  - `llm-verifier/enhanced/analytics/api.go` — health at `/health` and `/health/analytics` (`api.go:496,526,527`).
  - `llm-verifier/events/websocket_server.go` — WebSocket server, `/health` handler (`websocket_server.go:129`), `ListenAndServe()` (`websocket_server.go:235`).
  - `llm-verifier/events/grpc_server.go` — gRPC server via `net.Listen("tcp", address)` (`grpc_server.go:24,76`); the bind address is caller-supplied (no fixed default found).
- **Local-model / HTTP-provider consumption:** `internal/benchmark/http_provider.go` benchmarks an OpenAI-compatible HTTP endpoint; its doc comment names `http://localhost:8080/v1` as a "llama.cpp HTTP server" example (`http_provider.go:87`), and the tests bind `Endpoint: "http://localhost:8080/v1"` (`http_provider_test.go:58,103,110`). This is the seam through which an external local model (e.g. an OpenAI-compatible `:8080/v1` server) is verified. `UNKNOWN:` a provider literally named `helixllm` was NOT found in source — a repo-wide grep for `helixllm` returned only planning-doc / partial-investigation references, none in `.go` source. The integration path for a local HelixLLM model is the generic OpenAI-compatible HTTP provider above, not a bespoke `helixllm` provider adapter (as of this revision).

Cross-reference: the `ACP_*.md` design set (`ACP_IMPLEMENTATION_DESIGN.md`, `ACP_IMPLEMENTATION_GUIDE.md`, `ACP_API_DOCUMENTATION.md`, etc.) documents the "ACP" surface. `UNKNOWN:` whether "ACP" is a served protocol endpoint vs a capability/design layer was not resolved from source in this bounded pass — read the `ACP_*.md` docs (chunked) before wiring any ACP integration.

## 3. Dependencies

- **Own-org (external) Go module — exactly one:** `Challenges` — root `go.mod require digital.vasic.challenges` + `replace => ../challenges`, org `vasic-digital` (`helix-deps.yaml` `deps:` block, lines ~37-45; `why:` = "Anti-bluff Challenge Test infrastructure used by the LLMsVerifier test suites"). Layout `flat` — the consuming project must expose `challenges/` at its root per §11.4.31.
- **Inner subtree (NOT an external dep):** `digital.vasic.llmsverifier => ./llm-verifier` is an inner subtree of THIS submodule, per `helix-deps.yaml` header notes; it is intentionally NOT a helix-deps entry, and the inner `llm-verifier/go.mod` carries no own-org `replace`.
- **`.gitmodules`:** none (grep returned empty) — dependencies are managed via go.mod `replace` + `upstreams/` recipes, not git submodules.
- **Infra services (from `docker-compose.prod.yml` + `docker-compose.messaging.yml`, evidenced):** PostgreSQL (`postgres:15-alpine`, `prod.yml:82`), Redis (`redis:7-alpine`, `prod.yml:121`), Prometheus (`prod.yml:155`), Grafana (`prod.yml:191`), Nginx (`prod.yml:225`); and the optional messaging stack: RabbitMQ (`messaging.yml:9`), ZooKeeper (`messaging.yml:42`), Kafka (`messaging.yml:74`), Schema Registry (`messaging.yml:121`), Kafka-UI (`messaging.yml:153`).
- **Credentials:** the prod/messaging compose reference env-var NAMES such as `REDIS_PASSWORD` (`prod.yml:128`) and standard postgres user/db names. Env-var NAMES only are documented here; no values are read or echoed (§11.4.10). `UNKNOWN:` the exhaustive per-provider LLM-API-key env-var set (the repo ships an `.env.example`-class config surface; enumerate it from `configs/`/`.env*` chunked before deploy).

## 4. Deploy / Distribution design

Three distribution shapes, all in-tree:

1. **Container image + single-service compose** — `Dockerfile` (multi-stage: `FROM golang:1.25-alpine AS builder` → `FROM gcr.io/distroless/static-debian12:latest`; `EXPOSE 8080`; `HEALTHCHECK` `CMD ["/llm-verifier","health"]`; `ENTRYPOINT ["/llm-verifier"]`; `CMD ["server","-port","8080"]` — `Dockerfile:3,55,77,86,89,90`). `docker-compose.yml` runs the single `llm-verifier` service (`8080:8080`, healthcheck `http://localhost:8080/api/health` — `docker-compose.yml:4,13,22`).
2. **Production stack** — `docker-compose.prod.yml` composes `llm-verifier` (`image: llm-verifier:${TAG:-v1.0.0}`, `8080:8080`) + postgres + redis + prometheus (`9090`) + grafana + nginx (`prod.yml:4,13,21,81,120,154,190,224`).
3. **Optional messaging stack** — `docker-compose.messaging.yml` adds RabbitMQ/ZooKeeper/Kafka/Schema-Registry/Kafka-UI for the event-streaming surface.

Kubernetes/Helm manifests also ship (`k8s/`, `helm/`), plus an SDK (`sdk/`), a mobile client (`mobile/`), and a website (`website/`/`Website/`). **Distribution slice for an integrator:** the distroless `llm-verifier` image (built from the multi-stage Dockerfile) is the shippable artifact; the CLI/TUI binaries and the Go SDK are the library/consumer surface; the compose/helm/k8s files are the deployment recipes.

Rootless note: the runtime image is `gcr.io/distroless/static-debian12` (non-root-friendly static binary); align container execution with the project rootless-podman mandate (§11.4.161) at deploy time. `UNKNOWN:` the Dockerfile does not itself set a `USER` line in the lines probed — confirm the runtime UID before deploy.

## 5. Ports

Grounded, from compose + Dockerfile + source:

- **`8080`** — main `llm-verifier` app HTTP server (`Dockerfile:77,90`; `docker-compose.yml:13`; `docker-compose.prod.yml:21`; `enhanced/enterprise/api.go:150`; CLI default target `cmd/main.go:42`).
- **`9090`** — Prometheus (`docker-compose.prod.yml:162`).
- **`3000`** — Grafana (via `docker-compose.prod.yml:211` healthcheck `http://localhost:3000/api/health`).
- **Messaging stack** (`docker-compose.messaging.yml`): `5672`/`15672` RabbitMQ (`:17,18`), `2181` ZooKeeper (`:50`), `9092`/`29092` Kafka (`:85,86`), `8081` Schema Registry (`:132`), `8082:8080` Kafka-UI (`:166`).
- gRPC + WebSocket servers (`events/`) bind a **caller-supplied** address — `UNKNOWN:` no fixed default port found in source for those.

## 6. Health

- **`/api/health`** on the main app server (`llm-verifier/api/server.go:37,50`) — the endpoint the base `docker-compose.yml` healthcheck probes (`docker-compose.yml:22`).
- **`/api/enterprise/health`** (`enhanced/enterprise/api.go:215`), **`/health`** + **`/health/analytics`** (`enhanced/analytics/api.go:526,527`), **`/health`** on the websocket server (`events/websocket_server.go:129`).
- **Container HEALTHCHECK:** `CMD ["/llm-verifier","health"]` (a CLI subcommand, `Dockerfile:86`).
- **⚠ Reconciliation item (grounded, flag for integration — do NOT silently "fix"):** `docker-compose.prod.yml:44` probes `http://localhost:8080/health`, and the swagger docs advertise `/health` (`api/docs/docs.go:346`), but the main `api/server.go` registers **`/api/health`**, not `/health` (`server.go:37,50`). `/health` is registered on the *analytics*/*enterprise*/*websocket* servers, not the base `api/server.go`. Whether the prod-container's `:8080` is the base server (→ probe would 404) or the enterprise/analytics server (→ probe would pass) is `UNKNOWN:` from static files — confirm the actual `:8080` server binding before relying on the prod healthcheck.

## 7. How it boots

- **Server:** container `ENTRYPOINT ["/llm-verifier"]` + `CMD ["server","-port","8080"]` (`Dockerfile:89,90`) → the app's `server` subcommand starts the HTTP server (`api/server.go:65` `ListenAndServe()`).
- **CLI/TUI:** `llm-verifier <subcommand>` (root `cmd/main.go`) / `tui` (`cmd/tui/main.go`), targeting `--server http://localhost:8080` by default.
- **Compose:** `docker compose -f docker-compose.yml up` (single service) or `-f docker-compose.prod.yml` (full stack) or add `-f docker-compose.messaging.yml` (event stack). (Documented for reference; NOT executed by this materials pass.)

## 8. Materials status (verify pass)

| Gate material | State |
|---|---|
| Purpose | PRESENT (this doc) — cross-refs `README_IMPLEMENTATION_PLANS.md`, `helix-deps.yaml` |
| Architecture | PRESENT (this doc) + existing `ACP_*.md` design set, `docs/`, swagger `api/docs/docs.go` |
| Dependencies | PRESENT (this doc) — grounded in `helix-deps.yaml` + `go.mod` + compose |
| Deploy / Distribution | PRESENT (this doc) — `Dockerfile` + 3 compose files + `helm/` + `k8s/` + `sdk/` |
| Ports | PRESENT (this doc) — grounded in compose/Dockerfile/source |
| Health | PRESENT (this doc) — with the `/health` vs `/api/health` reconciliation flag |
| Boot | PRESENT (this doc) — grounded in Dockerfile ENTRYPOINT/CMD + cmd/ |

**Verdict:** `HAS-VERIFIED` — the repository already carries rich pre-integration material (README/implementation plans, `ACP_*.md` design set, swagger docs, helm/k8s, compose, SDK). This doc consolidates the deploy/ports/health/boot slice into one grounded gate artifact and surfaces two honest reconciliation items (the `/health` vs `/api/health` healthcheck-path mismatch; the absence of a source-level `helixllm` provider — local models are consumed via the generic OpenAI-compatible HTTP provider).

**Open `UNKNOWN:` items for integrators:** (1) `/health` vs `/api/health` prod-healthcheck path (§6); (2) whether "ACP" is a served endpoint vs a design layer (§2); (3) exhaustive per-provider LLM-API-key env-var set (§3); (4) runtime container `USER`/UID (§4); (5) gRPC/WebSocket default bind ports (§5).
