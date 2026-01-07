# Research: IaC Autoscaler (PulumiScale)

## 1. Web Framework Selection

**Task**: Research {Go web framework} for {lightweight sidecar service}.

**Options**:
1. **net/http (Stdlib)**: Zero dependencies, fast, sufficient for simple routing.
2. **Gin**: Popular, high performance, good middleware support, but heavier.
3. **Echo**: Easy to use, good binding support.
4. **Chi**: Lightweight, idomatic, just a router.

**Decision**: **net/http (Standard Library)** + **chi** (optional, if routing gets complex).

**Rationale**: 
- The service has very few endpoints (`/webhook/...`, `/scale/...`).
- Keeping dependencies minimal reduces binary size and security surface area.
- Standard library `http.ServeMux` in Go 1.22+ is powerful enough for method-based routing.
- If we need slightly better routing (e.g., path parameters `{pool}`), `go-chi/chi` is a tiny, standard-compatible dependency.

**Action**: Use `net/http` with Go 1.25.5 `ServeMux`. No external framework needed.

## 2. Automation API Concurrency

**Task**: Find best practices for {handling concurrent updates} in {Pulumi Automation API}.

**Challenge**: `pulumi up` fails with HTTP 409 if another update is in progress. The Automation API does not auto-retry this.

**Findings**:
- We must implement an "Optimistic Locking" retry loop.
- **Pattern**:
  1. Call `stack.Up()`
  2. Catch error.
  3. Check if error message contains "conflict" or "concurrent update".
  4. If yes, sleep (exponential backoff) and retry.
  5. Max retries: ~5-10 times.

**Decision**: Implement a `RetryOnConcurrency` wrapper around the `stack.Up` call in `internal/autoscaler/state.go`.

## 3. Webhook Payloads

**Task**: Research {payload formats} for {CloudWatch and Prometheus}.

**CloudWatch (SNS)**:
- **Header**: `x-amz-sns-message-type: Notification`
- **Body**: JSON with `Message` field. The `Message` often contains the alarm details JSON stringified (double-encoded).
- **Validation**: Should verify signature (URL in `SigningCertURL`), but for V1 MVP, shared secret authentication on the webhook URL or header is a simpler "good enough" start if running inside a VPC. *Refinement: Spec requires shared secret/token auth.*

**Prometheus (Alertmanager)**:
- **Body**: JSON list of alerts.
- **Fields**: `alerts[].labels`, `alerts[].annotations`.
- **Logic**: Need to map a specific label (e.g., `pool_name`) to the autoscaler's target pool.

**Decision**: 
- **Adapters**: Create specific structs in `internal/webhooks/types.go` to unmarshal these vendor formats.
- **Normalization**: All adapters must return a common `ScalingIntent` struct `{ TargetPool: string, Action: "set"|"delta", Value: int }`.

## 4. Security & Auth

**Task**: Research {webhook authentication} for {sidecar pattern}.

**Context**: The sidecar runs inside the cluster. Traffic might be internal (Prometheus) or external (SNS).

**Decision**:
- **Shared Secret**: Store a random token in Pulumi Config (`pulumiscale:webhookToken`).
- **Mechanism**: 
  - Clients must send `Authorization: Bearer <token>` OR
  - append `?token=<token>` (easier for some simple webhooks, though less secure).
- **Middleware**: Implement a simple middleware in `internal/api/middleware.go` to check this against the config value loaded at startup.
