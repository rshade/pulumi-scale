# Implementation Plan: IaC Autoscaler (PulumiScale)

**Branch**: `001-iac-autoscaler` | **Date**: 2026-01-07 | **Spec**: [specs/001-iac-autoscaler/spec.md](spec.md)
**Input**: Feature specification from `specs/001-iac-autoscaler/spec.md`

## Summary

This feature implements a "sidecar" autoscaler for Pulumi infrastructure. It runs as a long-lived process alongside the infrastructure, listening for webhook events (CloudWatch, Prometheus) and executing "Targeted Updates" (`pulumi up -t`) via the Automation API to scale resources dynamically while maintaining state consistency.

## Technical Context

**Language/Version**: Go 1.25.5
**Primary Dependencies**: 
- `github.com/pulumi/pulumi/sdk/v3/go/auto` (Automation API)
- `net/http` (Standard lib) with `chi` (Router)
**Storage**: Pulumi Backend (via Automation API)
**Testing**: `testing` (Go stdlib) for unit tests, Integration tests with local Pulumi stacks.
**Target Platform**: Linux container (Docker/K8s sidecar)
**Project Type**: Standalone CLI/Service
**Performance Goals**: <60s end-to-end scaling latency (excluding cloud provider provisioning).
**Constraints**: Must run in the same context as the Pulumi program (needs `Pulumi.yaml`, auth credentials).
**Scale/Scope**: Single process managing one stack.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- [x] **Code Quality**: Go 1.25.5 with standard formatting (`gofmt`).
- [x] **Testing Strategy**: TDD for calculation logic; Integration tests for Automation API wrappers.
- [x] **UX Consistency**: CLI flags and logging must match Pulumi standards.
- [x] **Performance**: "Targeted Updates" selected specifically to meet latency goals.
- [x] **Rigorous Planning**: Detailed spec and research phase included.
- [x] **Documentation Sync**: Plan includes tasks to update README.md and docs/ (Principle VI).

## Project Structure

### Documentation (this feature)

```text
specs/001-iac-autoscaler/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output
└── checklists/          # Validation checklists
```

### Source Code (repository root)

```text
cmd/pulumiscale/        # Main entry point
    ├── main.go         # Bootstrapping
    └── server.go       # HTTP server setup & wiring

internal/
    ├── autoscaler/     # Core logic
    │   ├── engine.go   # Calculation & Orchestration
    │   ├── config.go   # Stack Output parsing
    │   └── state.go    # Automation API wrapper (Up/Config)
    └── webhooks/       # Webhook Integrations
        ├── types.go    # Shared types and interfaces
        └── routers/    # One file per integration for easy extensibility
            ├── cloudwatch.go
            ├── prometheus.go
            ├── delta.go
            └── count.go
```

**Structure Decision**: Standard Go CLI structure. `internal/webhooks/routers/` is used to isolate each integration's handler logic, making it easy for contributors to add new webhook providers without modifying core server code.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| None | N/A | N/A |