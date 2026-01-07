---
description: "Task list for IaC Autoscaler (PulumiScale) implementation"
---

# Tasks: IaC Autoscaler (PulumiScale)

**Input**: Design documents from `specs/001-iac-autoscaler/`
**Prerequisites**: plan.md, spec.md, data-model.md, contracts/openapi.yaml
**Feature Branch**: `001-iac-autoscaler`

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel
- **[Story]**: [US1], [US2], etc.
- **Path**: Explicit file paths required

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and structure.

- [x] T001 Initialize Go module `github.com/rshade/pulumi-scale`
- [x] T002 Create directory structure (`cmd/pulumiscale`, `internal/autoscaler`, `internal/webhooks/routers`, `internal/api`)
- [x] T003 [P] Add dependencies (`github.com/pulumi/pulumi/sdk/v3/go/auto`, `github.com/go-chi/chi/v5`)

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core data models and server infrastructure.

- [x] T004 Define `ScalingRule` and `ScalingStrategy` types in `internal/autoscaler/types.go`
- [x] T005 Define `ScalingIntent` and `IntentAction` types in `internal/webhooks/types.go`
- [x] T006 Implement `ConfigLoader` to read Stack Outputs in `internal/autoscaler/config.go`
- [x] T007 Implement basic HTTP server with Chi router in `cmd/pulumiscale/server.go`
- [x] T008 Implement Authentication Middleware (Bearer Token) in `internal/api/middleware.go`
- [x] T009 Create `main.go` entrypoint that loads config and starts server in `cmd/pulumiscale/main.go`

**Checkpoint**: Server starts, reads (mock) stack outputs, and enforces auth.

## Phase 3: User Story 1 - Define Autoscaling Rules (Priority: P1)

**Goal**: Autoscaler reads and validates scaling rules from Pulumi Stack Outputs on startup.
**Independent Test**: Run binary in a dir with `Pulumi.yaml`, verify it logs loaded rules.

- [x] T010 [US1] Implement validation logic for `ScalingRule` (min/max/cooldown) in `internal/autoscaler/config.go`
- [x] T011 [US1] Add unit tests for config parsing and validation in `internal/autoscaler/config_test.go`
- [x] T012 [US1] Integrate `ConfigLoader` into `main.go` startup sequence

## Phase 4: User Story 2 - Trigger Scaling via Webhook (Priority: P1)

**Goal**: System accepts and parses webhook events (CW, Prometheus, Direct).
**Independent Test**: POST to endpoints, verify correct `ScalingIntent` is generated/logged.

- [x] T013 [P] [US2] Implement `CloudWatch` payload parser/router in `internal/webhooks/routers/cloudwatch.go`
- [x] T014 [P] [US2] Implement `Prometheus` payload parser/router (with `pool` label check) in `internal/webhooks/routers/prometheus.go`
- [x] T015 [P] [US2] Implement `Count` (Absolute) handler in `internal/webhooks/routers/count.go`
- [x] T016 [P] [US2] Implement `Delta` (Incremental) handler in `internal/webhooks/routers/delta.go`
- [x] T017 [US2] Create `Engine` struct to receive `ScalingIntent` and calculate target in `internal/autoscaler/engine.go`
- [x] T018 [US2] Implement Cooldown/Debounce logic in `internal/autoscaler/engine.go`
- [x] T019 [P] [US2] Add unit tests for each webhook parser in `internal/webhooks/routers/*_test.go`
- [x] T034 [P] [US2] Add specific unit test for CloudWatch Router in `internal/webhooks/routers/cloudwatch_test.go`
- [x] T035 [P] [US2] Add specific unit test for Prometheus Router in `internal/webhooks/routers/prometheus_test.go`

## Phase 5: User Story 3 - Persist Scaled State (Priority: P1)

**Goal**: Engine executes `pulumi up` to apply changes and persist state.
**Independent Test**: Trigger scale event, verify `pulumi config` updates and resource scales.

- [x] T020 [US3] Implement `Automation API` wrapper for `SetConfig` in `internal/autoscaler/state.go`
- [x] T021 [US3] Implement `Automation API` wrapper for `Up` (Targeted) in `internal/autoscaler/state.go`
- [x] T022 [US3] Implement `RetryOnConcurrency` logic (5 retries, exp backoff) in `internal/autoscaler/state.go`
- [x] T023 [US3] Wire `Engine` to call `state.Up` on valid intent in `internal/autoscaler/engine.go`
- [x] T024 [US3] Add integration test (using local stack) for full scaling flow in `tests/integration/scale_test.go` (Verify execution time < 60s per SC-003)
- [x] T036 [US3] Add verification test for `SetConfig` logic (Config Set) in `internal/autoscaler/state_test.go`

## Phase 6: User Story 4 - Recover State after Restart (Priority: P2)

**Goal**: Ensure state consistency on reboot.
**Independent Test**: Restart service, trigger scale, verify it builds on persisted state.

- [x] T025 [US4] Verify `ConfigLoader` correctly reads *persisted* config (not just code defaults) in `internal/autoscaler/config.go`
- [x] T026 [US4] Add test case: Kill process, restart, verify state retention in `tests/integration/recovery_test.go`

## Phase 7: User Story 5 - Dry Run Preview (Priority: P3)

**Goal**: Support `?dryRun=true` to preview changes.
**Independent Test**: Call endpoint with dryRun, verify no actual infra changes.

- [x] T027 [US5] Implement `Preview` method in `internal/autoscaler/state.go`
- [x] T028 [US5] Update `Engine` to handle `DryRun` flag from intent in `internal/autoscaler/engine.go`
- [x] T029 [US5] Update Webhook routers to parse `dryRun` query param in `internal/webhooks/routers/*.go`

## Phase 8: Polish & Cross-Cutting Concerns

- [x] T030 [P] Documentation updates (README.md, docs/) matching Constitution Principle VI
- [x] T031 [P] Standardize logging (JSON/Text) across all components
- [x] T032 Verify OpenAPI spec compliance
- [x] T033 Final code review and linting (`golangci-lint`)
- [x] T037 [P] Create `renovate.json` for dependency updates
- [x] T038 [P] Create `.golangci-lint.yml` configuration

## Dependencies & Execution Order

1. **Setup & Foundational** (T001-T009) must complete first.
2. **US1 & US2** (T010-T019) can run in parallel.
3. **US3** (T020-T024) depends on US2 (Engine) and US1 (Config).
4. **US4** (T025-T026) depends on US3.
5. **US5** (T027-T029) can run anytime after US3.

## Implementation Strategy

1. **MVP**: Setup + Foundational + US1 + US2 + US3 (Manual & Webhook scaling working).
2. **Resilience**: Add US4 (Recovery).
3. **Feature Complete**: Add US5 (Dry Run) + Polish.
