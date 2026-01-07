# PulumiScale (IaC Autoscaler)

PulumiScale is a sidecar service that enables dynamic, metric-driven scaling for Pulumi infrastructure while maintaining state consistency ("True IaC").

## Features

- **True IaC**: Updates Pulumi Config and Backend state so infrastructure code reflects reality.
- **Dynamic Scaling**: Triggerable via Webhooks (CloudWatch, Prometheus) or direct API.
- **Safety**: Guardrails (Min/Max), Cooldowns, and Atomic Updates.
- **Recovery**: Restores last scaled state on restart.

## Quickstart

### Prerequisites
- Go 1.21+
- Pulumi CLI
- Active Pulumi Stack

### Installation
```bash
go install github.com/rshade/pulumi-scale/cmd/pulumiscale@latest
```

### Usage
Run the sidecar in your Pulumi program directory:
```bash
pulumiscale --stack dev --port 8080
```

### Configuration
Define scaling rules in your Pulumi Stack Outputs:
```typescript
export const pulumiscale = {
    "worker-pool": {
        targetUrn: "urn:pulumi:...",
        configKey: "workerCount",
        min: 1,
        max: 10,
        cooldown: 60
    }
};
```

## API

- `POST /webhook/{pool}/cloudwatch` - AWS SNS
- `POST /webhook/{pool}/prometheus` - Alertmanager
- `POST /webhook/{pool}/delta` - Incremental (`{"delta": 1}`)
- `POST /webhook/{pool}/count` - Absolute (`{"value": 5}`)
