# Product Design: PulumiScale (IaC Autoscaler)

## 1. Context & Problem Statement
Current Infrastructure as Code (IaC) architectures are predominantly static. To handle dynamic scaling (e.g., AI inference workloads, rapid traffic spikes), users currently resort to:
1.  **Ignoring State:** Using `ignoreChanges` in Pulumi to cede control to external autoscalers (Karpenter, HPA, ASGs).
2.  **Split Brain:** Maintaining "Macro" infra in IaC and "Micro" scaling in separate, imperative systems.

**Goal:** Build an "IaC Autoscaler" that retains the security, auditability, and predictability of Pulumi while enabling dynamic, metric-driven scaling. It should feel like "Karpenter built with Pulumi."

## 2. Core Requirements
*   **True IaC:** The state file must eventually reflect the scaled state.
*   **Recovery:** If the autoscaler dies, a standard `pulumi up` should resolve the system to a consistent state.
*   **Security:** Leverage existing IaC permissions and policies.
*   **Integration:** Triggerable via standard metrics (CloudWatch, Prometheus) or Webhooks.

## 3. Key Design Questions & Answers

### Q1: Granularity
**Question:** Are we scaling Cloud Infrastructure (AWS ASGs, Managed Node Groups) or Kubernetes Resources (Pods/Deployments)?
**Answer:** Both, but primarily focusing on Cloud Infrastructure where IaC shines (e.g., resizing a NodeGroup, scaling a Spot Fleet) which are typically "heavy" operations compared to Pod scaling.

### Q2: Speed & Latency
**Question:** Can you tolerate a 30-60 second delay for scaling (Waiting for `pulumi up`), or do you need sub-second reaction?
**Answer:** *Pending Confirmation.* The proposed architecture assumes a tolerance for "fast but not instant" scaling (10s - 1m), controlled by "Targeted Updates" (`pulumi up -t urn`). This is suitable for infrastructure scaling but likely too slow for per-request scaling.

### Q3: The "Recovery" Definition
**Question:** If the process dies and we run `pulumi up`, what happens?
*   *Option A:* Sync to baseline (e.g., scale back down to min size).
*   *Option B:* Detect current running count (e.g., 50), validate against metrics, and accept it.
**Answer:** **Option A (Persistent Config)** is the selected path for the "Automation API Operator" model.
*   The autoscaler updates the *Pulumi Config* (e.g., `pulumi config set scale 50`) before running the update.
*   Therefore, the "Desirable State" is persisted in the Pulumi Backend.
*   If the scaler dies, the last known config is preserved. A manual `pulumi up` effectively restores the *last scaled state*, not the arbitrary baseline.

## 4. Proposed Architecture: The "Automation API Operator"

This approach uses a "Sidecar/Operator" pattern leveraging the Pulumi Automation API (`sdk/go/auto`).

### 4.1. The Components
1.  **The "Sidecar" (PulumiScale Service):**
    *   A lightweight Go binary running inside the infrastructure (e.g., K8s Pod, ECS Task).
    *   Wraps the Pulumi Automation API.
    *   Has access to the Stack's source code and credentials.

2.  **The Contract (Stack Output Discovery):**
    *   Instead of maintaining a separate mapping file, the "Contract" is defined within the Pulumi program itself as a **Stack Output**.
    *   This ensures the scaler configuration stays in sync with the codebase (Single Source of Truth).

    **Example (TypeScript):**
    ```typescript
    export const pulumiscale = {
        "worker-pool": {
            "targetUrn": nodeGroup.urn,      // The Resource to watch (for target up)
            "configKey": "workerCount",      // The Config to change
            "strategy": "incremental",       // "incremental" (+/-) or "absolute" (set)
            "min": 1,                        // Safety Guardrail
            "max": 50,                       // Safety Guardrail
            "cooldown": 300                  // Seconds to wait after scaling before allowing another op
        },
        "inference-pods": {
            // ...
        }
    };
    ```

### 4.2. The Workflow (The "Fast Path")
1.  **Startup:** `pulumiscale` boots, runs `pulumi stack output pulumiscale`, and dynamically registers routes:
    *   `POST /webhook/{pool}/cloudwatch`
    *   `POST /webhook/{pool}/prometheus`
    *   `POST /webhook/{pool}/count` (Absolute scaling)
    *   `POST /webhook/{pool}/delta` (Incremental scaling)
2.  **Trigger:** Webhook receives a request on a specific adapter endpoint.
    *   **Adapter Logic:** The specific handler parses the payload (e.g., extracts SNS message for CloudWatch) and normalizes it to an internal `ScalingIntent` (Target Value or Delta).
3.  **Logic & Safety:**
    *   **Debounce/Cooldown:** Check if we scaled recently. If inside cooldown window, ignore.
    *   **State Retrieval:** Use `stack.Outputs()` to instantly retrieve the last known configuration for "Incremental" calculations (`current + delta`). This reads from the backend state and avoids a slow `refresh`.
    *   **Calculate:** Read current config -> apply delta -> clamp to `min`/`max`.
4.  **Persist:** Service calls `stack.SetConfig("workerCount", calculatedValue)`.
    *   *Crucial:* State is now saved in the backend.
5.  **Execute:** Service calls `stack.Up(optup.Target("urn:..."))`.
    *   *Optimization:* Using `Target` skips the refresh/diff for the rest of the stack.
    *   **Concurrency Handling:** The `stack.Up` call must be wrapped in a retry loop. If `IsConcurrentUpdateError` is detected (HTTP 409 from backend), the scaler will back off and retry, as the Automation API does not handle locking waits natively.

## 5. Advanced Capabilities

### 5.1. Payload Adapters
`pulumiscale` avoids complex payload sniffing by exposing dedicated endpoints for supported providers:
1.  **AWS CloudWatch:** `POST /webhook/{pool}/cloudwatch`. Parses SNS JSON structure.
2.  **Prometheus:** `POST /webhook/{pool}/prometheus`. Parses Alertmanager JSON structure.
3.  **Direct:**
    *   `POST /webhook/{pool}/count` -> Expects `{"value": 10}`.
    *   `POST /webhook/{pool}/delta` -> Expects `{"delta": 1}`.

### 5.2. Security
*   **Authentication:** The service will verify a shared secret (Bearer Token) or HMAC signature in webhook headers.
*   **Secret Storage:** This token will be stored in the Pulumi Config (`pulumiscale:webhook-secret`) and read by the binary on startup.

### 5.3. "Dry Run" Endpoint
*   Support `POST /scale/worker-pool?dryRun=true`.
*   Calculates the new config and runs `pulumi preview -t urn:...`.
*   Returns the diff to the caller. Useful for testing alarms and permissions without impact.

## 6. Operational Context
*   **Execution Environment:** The `pulumiscale` binary will run directly within a **Pulumi Program Directory**. It expects to find `Pulumi.yaml` and the language specific project files (e.g., `main.go`, `index.ts`) in its working directory.
*   **Modes of Operation:**
    1.  **Server Mode (`/scale`):** Listens for webhook events to trigger targeted scaling updates.
    2.  **Maintenance Mode (`up`):** The tool should support (or wrap) the standard `pulumi up` behavior to ensure the baseline state is correct before accepting scaling events, or to recover from a crashed state.
*   **Codebase Location:** This will be implemented as a **standalone Go binary** (separate repository or `cmd/pulumiscale`) utilizing the `sdk/go/auto` library. It does not require changes to the core Pulumi CLI.
