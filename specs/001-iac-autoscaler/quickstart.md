# Quickstart: IaC Autoscaler (PulumiScale)

## Prerequisites

1. **Go 1.21+** installed.
2. **Pulumi CLI** installed and authenticated.
3. An existing Pulumi project (or create a new one).

## 1. Setup Pulumi Program

Add the `pulumiscale` contract to your stack outputs.

**TypeScript Example (`index.ts`):**

```typescript
import * as pulumi from "@pulumi/pulumi";

// 1. Get current count from config (default to 1)
const config = new pulumi.Config();
const workerCount = config.requireNumber("workerCount");

// 2. Define your resource (e.g., specific resource URN)
// In a real app, this would be your NodeGroup, ASG, or SpotFleet
const dummyResourceUrn = "urn:pulumi:dev::my-stack::custom:Resource::my-pool";

// 3. Export the contract
export const pulumiscale = {
    "worker-pool": {
        targetUrn: dummyResourceUrn,
        configKey: "workerCount",
        min: 1,
        max: 10,
        cooldown: 60,
        strategy: "incremental",
    }
};
```

**Apply the initial state:**
```bash
pulumi config set workerCount 1
pulumi up
```

## 2. Configure Authentication

Set a secret token for the webhook:
```bash
pulumi config set pulumiscale:webhookToken "super-secret-token" --secret
```

## 3. Run the Autoscaler

The binary must run in the same directory as your Pulumi program.

```bash
# Build (or run directly)
go run cmd/pulumiscale/main.go server --port 8080
```

*Expected Log Output:*
```text
INFO: Found active stack: dev
INFO: Loaded scaling rule for pool: worker-pool (Min: 1, Max: 10)
INFO: Server listening on :8080
```

## 4. Test Scaling

**Increase Count (Delta):**

```bash
curl -X POST http://localhost:8080/webhook/worker-pool/delta \
  -H "Authorization: Bearer super-secret-token" \
  -H "Content-Type: application/json" \
  -d '{"delta": 1}'
```

**Verify:**
1. Check the logs: `INFO: Scaling worker-pool from 1 to 2`
2. Check Pulumi config: `pulumi config get workerCount` -> `2`

**Dry Run:**

```bash
curl -X POST "http://localhost:8080/webhook/worker-pool/count?dryRun=true" \
  -H "Authorization: Bearer super-secret-token" \
  -d '{"value": 5}'
```
