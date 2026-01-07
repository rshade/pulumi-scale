package autoscaler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/auto"
)

// ConfigLoader is responsible for loading scaling rules from the Pulumi stack.
type ConfigLoader struct {
	StackName string
	WorkDir   string
}

// NewConfigLoader creates a new ConfigLoader instance.
func NewConfigLoader(stackName, workDir string) *ConfigLoader {
	return &ConfigLoader{
		StackName: stackName,
		WorkDir:   workDir,
	}
}

// LoadRules retrieves the stack outputs and parses the "pulumiscale" output into a map of ScalingRules.
func (cl *ConfigLoader) LoadRules(ctx context.Context) (map[string]ScalingRule, error) {
	// Initialize the stack (assuming workDir contains a valid Pulumi program)
	// We use UpsertStack to get a handle to the stack, assuming it already exists.
	// If it doesn't, we might need a different approach, but for a sidecar, the stack should exist.
	s, err := auto.UpsertStackLocalSource(ctx, cl.StackName, cl.WorkDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load stack: %w", err)
	}

	// Get stack outputs
	outputs, err := s.Outputs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get stack outputs: %w", err)
	}

	// Look for "pulumiscale" output
	val, ok := outputs["pulumiscale"]
	if !ok {
		return nil, fmt.Errorf("stack output 'pulumiscale' not found")
	}

	// The output value from Automation API is an auto.OutputValue.
	// We need to marshal/unmarshal or type assert to get the structure.
	// auto.OutputValue.Value is interface{}.
	
	// Marshaling the value to JSON and then unmarshaling into our struct is a robust way to handle map[string]interface{}.
	data, err := json.Marshal(val.Value)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal pulumiscale output: %w", err)
	}

	var rules map[string]ScalingRule
	if err := json.Unmarshal(data, &rules); err != nil {
		return nil, fmt.Errorf("failed to unmarshal scaling rules: %w", err)
	}

	// Populate PoolName from the map key since it's ignored in JSON (`json:"-"`)
	for name, rule := range rules {
		rule.PoolName = name
		if err := rule.Validate(); err != nil {
			return nil, fmt.Errorf("invalid rule for pool '%s': %w", name, err)
		}
		rules[name] = rule
	}

	return rules, nil
}

// Validate checks if the ScalingRule is valid.
func (r *ScalingRule) Validate() error {
	if r.TargetURN == "" {
		return fmt.Errorf("targetUrn is required")
	}
	if r.ConfigKey == "" {
		return fmt.Errorf("configKey is required")
	}
	if r.Min < 0 {
		return fmt.Errorf("min must be non-negative")
	}
	if r.Max < r.Min {
		return fmt.Errorf("max must be greater than or equal to min")
	}
	if r.CooldownSeconds < 0 {
		return fmt.Errorf("cooldown must be non-negative")
	}
	return nil
}
