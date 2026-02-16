# PulumiScale Strategic Roadmap

This document tracks the feature roadmap, planned enhancements, and long-term
vision for PulumiScale.

## Vision

PulumiScale is an infrastructure-as-code autoscaler that bridges cloud
monitoring signals (CloudWatch, Prometheus, etc.) with Pulumi stack
configuration updates. It enables automated, event-driven scaling of
cloud resources through declarative IaC workflows.

## Immediate Focus (v1.1)

*Focus: Correctness and robust cloud integration.*

- [ ] **fix(webhooks): CloudWatch Directionality** ([#6](../../issues/6))
  - Support query params (`?action=down`) and parsing alarm descriptions.
- [ ] **fix(autoscaler): Accurate DryRuns** ([#7](../../issues/7))
  - Inject temporary config into `pulumi preview` so changes are visible.
- [ ] **chore(engine): Graceful Shutdown** ([#8](../../issues/8))
  - Finish in-flight `pulumi up` operations before exiting.

## Near-Term Vision (v1.2)

*Focus: Running in production with confidence.*

- [ ] **feat(observability): Prometheus Metrics** ([#9](../../issues/9))
  - `/metrics` endpoint with scaling counters and duration histograms.
- [ ] **feat(api): Health Probes** ([#10](../../issues/10))
  - `/healthz` and `/readyz` for container orchestration.
- [ ] **sec(webhooks): Security Verification** ([#11](../../issues/11))
  - Verify AWS SNS signatures and Prometheus Bearer tokens.

## Future Vision (Long-Term)

*Focus: Smarter scaling strategies and enterprise resilience.*

### Intelligence and Logic (v2.0)

- [ ] **feat(cron): Scheduled Scaling** ([#12](../../issues/12))
  - Built-in cron scheduler for time-based scaling (e.g., "Scale to 0 at
    8 PM").
- [ ] **feat(strategy): Step Scaling** ([#13](../../issues/13))
  - Granular rules: "If metric > 90%, add 5. If > 70%, add 2."
- [ ] **feat(notifications): ChatOps Webhooks** ([#14](../../issues/14))
  - Outbound notifications to Slack/Teams/Discord on scaling events.

### Enterprise and Resilience

- [ ] **feat(config): Composite Scaling** ([#15](../../issues/15))
  - Ability to update multiple config keys in a single atomic transaction
    (e.g., scale app + DB connections).
- [ ] **feat(ha): Leader Election** ([#16](../../issues/16))
  - Support running multiple `pulumiscale` replicas with a locking mechanism
    to prevent race conditions.
- [ ] **feat(drift): Auto-Refresh** ([#17](../../issues/17))
  - Periodically run `pulumi refresh` to detect and reconcile manual cloud
    changes before scaling.

## Completed Milestones

### 2026-Q1

- [x] **feat(autoscaler): Implement PulumiScale MVP** ([#1](../../issues/1))
  - Initial sidecar implementation with CloudWatch webhook integration,
    Pulumi Automation API engine, and basic scaling loop.

## Boundary Safeguards

- PulumiScale does **not** manage cloud resources directly; it only updates
  Pulumi stack configuration and triggers `pulumi up`.
- PulumiScale does **not** store state; Pulumi's backend is the source of
  truth.
- PulumiScale does **not** implement cloud provider SDKs; it relies on
  webhook payloads for signal ingestion.
