# Feature Specification: IaC Autoscaler (PulumiScale)

**Feature Branch**: `001-iac-autoscaler`  
**Created**: 2026-01-07  
**Status**: Draft  
**Input**: User description: (See original prompt)

## Clarifications

### Session 2026-01-07
- Q: Webhook Authentication Method → A: `Authorization: Bearer <token>` (Option A)
- Q: Concurrency Retry Limits → A: 5 retries with exponential backoff (Option B)
- Q: Prometheus Alert Label Mapping → A: Use the `pool` label (Option A)

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Define Autoscaling Rules (Priority: P1)

As a Platform Engineer, I want to define scaling rules directly in my Pulumi program using Stack Outputs so that the scaling configuration is versioned with the infrastructure code.

**Why this priority**: Fundamental to the "True IaC" goal; defines the contract between infrastructure and the autoscaler.

**Independent Test**: Can be tested by creating a Pulumi stack with specific outputs and verifying the autoscaler reads them correctly on startup.

**Acceptance Scenarios**:

1. **Given** a Pulumi program with a `pulumiscale` stack output defining a "worker-pool", **When** the autoscaler service starts, **Then** it registers a webhook endpoint for that pool.
2. **Given** the stack output defines `min: 1` and `max: 50`, **When** a scaling event requests 100 nodes, **Then** the autoscaler clamps the request to 50.

---

### User Story 2 - Trigger Scaling via Webhook (Priority: P1)

As an Operator or Monitoring System, I want to trigger scaling events via standard webhooks (CloudWatch, Prometheus) so that infrastructure scales dynamically based on metrics.

**Why this priority**: Core functionality; enables the actual "autoscaling" behavior.

**Independent Test**: Can be tested by sending mock HTTP POST requests to the local autoscaler service and observing the log output or state change.

**Acceptance Scenarios**:

1. **Given** the autoscaler is running, **When** a `POST` request is sent to `/webhook/{pool}/cloudwatch` with a valid SNS payload, **Then** the system calculates the new desired count.
2. **Given** the autoscaler is running, **When** a `POST` request is sent to `/webhook/{pool}/prometheus` with a valid Alertmanager payload, **Then** the system calculates the new desired count.
3. **Given** a cooldown period is active, **When** a webhook triggers, **Then** the request is ignored (debounced).

---

### User Story 3 - Persist Scaled State (Priority: P1)

As a Platform Engineer, I want scaling actions to update the Pulumi Config and Backend state so that "True IaC" is maintained and the state file reflects reality.

**Why this priority**: Differentiates this tool from external scalers; prevents "Split Brain" and ensures auditability.

**Independent Test**: Can be tested by triggering a scale event and verifying `pulumi config` shows the new value.

**Acceptance Scenarios**:

1. **Given** a scaling event calculates a new count of 5, **When** the event is processed, **Then** the system updates the stack configuration key (e.g., `workerCount`) to 5.
2. **Given** the configuration is updated, **When** the update completes, **Then** the Pulumi Backend state reflects the new configuration value.

---

### User Story 4 - Recover State after Restart (Priority: P2)

As an Operator, I want the system to respect the last scaled state if the autoscaler service restarts so that the infrastructure doesn't revert to an arbitrary baseline.

**Why this priority**: Ensures resilience and operational stability.

**Independent Test**: Can be tested by scaling the system, killing the autoscaler process, restarting it, and running a standard update.

**Acceptance Scenarios**:

1. **Given** the system scaled to 10 nodes and the autoscaler crashed, **When** the autoscaler restarts and runs an update, **Then** it restores the state to 10 nodes (from persisted config), not the initial default.

---

### User Story 5 - Dry Run Preview (Priority: P3)

As a Platform Engineer, I want to preview scaling actions without applying them so that I can test permissions and alarm configurations safely.

**Why this priority**: Useful for testing and validation but not critical for core operation.

**Independent Test**: Can be tested by calling the endpoint with `?dryRun=true`.

**Acceptance Scenarios**:

1. **Given** the autoscaler is running, **When** a request is sent to `/scale/{pool}?dryRun=true`, **Then** the system returns the calculated diff without changing actual infrastructure.

### Edge Cases

- **Concurrency**: What happens if two scaling events arrive simultaneously? (System must handle locking/retries).
- **Network Failure**: What happens if the Pulumi Backend is unreachable? (System should retry or fail safely).
- **Invalid Config**: What happens if the Stack Output is malformed? (System should log error and fail to start/register route).

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST run as a sidecar/process within the Pulumi Program Directory.
- **FR-002**: System MUST read scaling configuration (targets, limits, strategies) from the Pulumi Stack Outputs on startup.
- **FR-003**: System MUST expose HTTP endpoints to receive scaling triggers from AWS CloudWatch (SNS JSON format).
- **FR-004**: System MUST expose HTTP endpoints to receive scaling triggers from Prometheus (Alertmanager JSON format), mapping the `pool` label in the alert to the target worker pool.
- **FR-005**: System MUST expose generic HTTP endpoints for direct Absolute (`count`) and Incremental (`delta`) scaling.
- **FR-006**: System MUST authenticate webhook requests using a shared secret or token stored in Pulumi Config, passed via the `Authorization: Bearer <token>` header.
- **FR-007**: System MUST validate calculated scaling targets against defined `min` and `max` guardrails.
- **FR-008**: System MUST debounce scaling events based on a configurable `cooldown` period.
- **FR-009**: System MUST persist the new desired state by updating the Pulumi Stack Configuration programmatically.
- **FR-010**: System MUST execute infrastructure updates using "Targeted Updates" to minimize latency by only refreshing relevant resources.
- **FR-011**: System MUST handle concurrent update errors by implementing a backoff-and-retry mechanism (max 5 retries with exponential backoff).
- **FR-012**: System MUST support a "Dry Run" mode that calculates changes without applying them.

### Key Entities *(include if feature involves data)*

- **ScalingRule**: Defines the resource to scale (URN), the config key to modify, limits (min/max), and strategy. Defined in Stack Output.
- **ScalingIntent**: An internal representation of a scaling request (Target Value or Delta) derived from external webhooks.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: **True IaC Consistency**: 100% of successful scaling events result in the Pulumi Backend state matching the actual infrastructure count.
- **SC-002**: **Recovery**: A standard `pulumi up` execution after a scaler restart restores the *last scaled state*, not the initial code baseline.
- **SC-003**: **Performance**: Targeted scaling operations complete within 60 seconds (excluding cloud provider provisioning time).
- **SC-004**: **Security**: 100% of webhook requests are authenticated; unauthenticated requests are rejected.