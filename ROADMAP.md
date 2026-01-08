# PulumiScale Roadmap

This document tracks the feature roadmap, planned enhancements, and long-term vision for PulumiScale.

## Phase 1: Stabilization & Fixes (v1.1)
*Focus: Correctness and robust cloud integration.*

- [ ] **fix(webhooks): CloudWatch Directionality** (Issue #6)
    - Support query params (`?action=down`) and parsing alarm descriptions.
- [ ] **fix(autoscaler): Accurate DryRuns** (Issue #7)
    - Inject temporary config into `pulumi preview` so changes are visible.
- [ ] **chore(engine): Graceful Shutdown** (Issue #8)
    - Finish in-flight `pulumi up` operations before exiting.

## Phase 2: Operational Excellence (v1.2)
*Focus: Running in production with confidence.*

- [ ] **feat(observability): Prometheus Metrics** (Issue #9)
    - `/metrics` endpoint with scaling counters and duration histograms.
- [ ] **feat(api): Health Probes** (Issue #10)
    - `/healthz` and `/readyz` for container orchestration.
- [ ] **sec(webhooks): Security Verification** (Issue #11)
    - Verify AWS SNS signatures and Prometheus Bearer tokens.

## Phase 3: Intelligence & Logic (v2.0)
*Focus: Smarter scaling strategies.*

- [ ] **feat(cron): Scheduled Scaling** (Issue #12)
    - Built-in cron scheduler for time-based scaling (e.g., "Scale to 0 at 8 PM").
- [ ] **feat(strategy): Step Scaling** (Issue #13)
    - Granular rules: "If metric > 90%, add 5. If > 70%, add 2."
- [ ] **feat(notifications): ChatOps Webhooks** (Issue #14)
    - Outbound notifications to Slack/Teams/Discord on scaling events.

## Phase 4: Enterprise & Resilience (Backlog)
*Focus: High scale and complex environments.*

- [ ] **feat(config): Composite Scaling** (Issue #15)
    - Ability to update multiple config keys in a single atomic transaction (e.g., scale app + DB connections).
- [ ] **feat(ha): Leader Election** (Issue #16)
    - Support running multiple `pulumiscale` replicas with a locking mechanism to prevent race conditions.
- [ ] **feat(drift): Auto-Refresh** (Issue #17)
    - Periodically run `pulumi refresh` to detect and reconcile manual cloud changes before scaling.