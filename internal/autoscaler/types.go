package autoscaler

type ScalingStrategy string

const (
    StrategyIncremental ScalingStrategy = "incremental" // +/- delta
    StrategyAbsolute    ScalingStrategy = "absolute"    // set to value
)

type ScalingRule struct {
    // The key in the user's stack output map (e.g., "worker-pool")
    PoolName string `json:"-"` 

    // The Pulumi URN of the resource to target with `pulumi up -t`
    // Example: "urn:pulumi:dev::my-stack::aws:autoscaling/group:Group::workers"
    TargetURN string `json:"targetUrn"`

    // The Pulumi Config key to update
    // Example: "workerCount"
    ConfigKey string `json:"configKey"`

    // Scaling limits (Guardrails)
    Min int `json:"min"`
    Max int `json:"max"`

    // Cooldown in seconds before allowing another scale event
    CooldownSeconds int `json:"cooldown"`

    // (Optional) Strategy defaults. Webhooks can override or imply this.
    Strategy ScalingStrategy `json:"strategy"`
}
