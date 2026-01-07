# Issue #1: Implement PulumiScale (IaC Autoscaler) MVP

## Description

Current Infrastructure as Code (IaC) architectures are predominantly static. To handle dynamic scaling, users often resort to "Split Brain" architectures where scaling is managed by imperative external systems, or they use `ignoreChanges` in Pulumi, losing state consistency.

**Goal:** Build an "IaC Autoscaler" sidecar that retains the security, auditability, and predictability of Pulumi while enabling dynamic, metric-driven scaling.

## Core Requirements

- **True IaC:** The Pulumi state file must eventually reflect the scaled state.
- **Recovery:** If the autoscaler restarts, it should restore the last known scaled state from the Pulumi Backend.
- **Security:** Authenticated webhook endpoints (Bearer Token).
- **Integration:** Support for AWS CloudWatch (SNS) and Prometheus (Alertmanager) triggers.
- **Performance:** End-to-end scaling (config update + targeted up) should complete within 60 seconds.

## Acceptance Criteria

- [x] Sidecar service loads scaling rules from Stack Outputs.
- [x] Exposes `/webhook/{pool}/cloudwatch` for SNS notifications.
- [x] Exposes `/webhook/{pool}/prometheus` for Alertmanager alerts.
- [x] Exposes `/webhook/{pool}/count` and `/webhook/{pool}/delta` for direct control.
- [x] Updates Pulumi Stack Config and executes `pulumi up --target <urn>`.
- [x] Implements exponential backoff retry for concurrent update conflicts (HTTP 409).
- [x] Supports `?dryRun=true` for previewing actions via `pulumi preview`.
- [x] Structured logging using `zerolog`.

## Technical Details

- **Language:** Go 1.25.5
- **SDK:** Pulumi Automation API
- **Framework:** `net/http` + `chi`
- **Auth:** Bearer Token via `pulumiscale:webhookToken` config.
