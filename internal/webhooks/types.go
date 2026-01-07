package webhooks

type IntentAction string

const (
    ActionSet   IntentAction = "set"
    ActionDelta IntentAction = "delta"
)

type ScalingIntent struct {
    // The pool to target (must match a ScalingRule.PoolName)
    TargetPool string

    // What to do
    Action IntentAction

    // The value (e.g., 50 for Set, +1/-1 for Delta)
    Value int
    
    // Metadata for logging
    Source string // "cloudwatch", "prometheus", "manual"
    Reason string // "CPU > 80%", "Alarm Triggered"
    
    DryRun bool
}
