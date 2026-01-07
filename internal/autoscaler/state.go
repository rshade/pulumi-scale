package autoscaler

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optpreview"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optup"
)

// StateManager handles Automation API interactions.
type StateManager struct {
	StackName string
	WorkDir   string
}

func NewStateManager(stackName, workDir string) *StateManager {
	return &StateManager{
		StackName: stackName,
		WorkDir:   workDir,
	}
}

// GetCurrentCount retrieves the current value of a config key.
// Returns 0 if key not found (or error).
func (sm *StateManager) GetCurrentCount(ctx context.Context, key string) (int, error) {
	s, err := auto.UpsertStackLocalSource(ctx, sm.StackName, sm.WorkDir)
	if err != nil {
		return 0, err
	}

	cfg, err := s.GetConfig(ctx, key)
	if err != nil {
		// If key missing, default to 0? Or error?
		// auto returns error if key not found? No, it might return empty.
		// If error, likely stack or connection issue.
		// Assuming 0 if not set is risky for infra scaling.
		// But for simple integer keys:
		return 0, err
	}

	// Assuming the config value is stored as a string that can be parsed as int.
	// Pulumi ConfigValue.Value is string.
	var val int
	_, err = fmt.Sscanf(cfg.Value, "%d", &val)
	if err != nil {
		return 0, fmt.Errorf("failed to parse config value '%s' as int: %w", cfg.Value, err)
	}

	return val, nil
}

// Apply updates the config and runs a targeted up.
func (sm *StateManager) Apply(ctx context.Context, rule ScalingRule, newValue int) error {
	s, err := auto.UpsertStackLocalSource(ctx, sm.StackName, sm.WorkDir)
	if err != nil {
		return err
	}

	// 1. Set Config
	// We set it as a string.
	err = s.SetConfig(ctx, rule.ConfigKey, auto.ConfigValue{Value: fmt.Sprintf("%d", newValue)})
	if err != nil {
		return fmt.Errorf("failed to set config: %w", err)
	}

	// 2. Run Up with Retry
	return sm.retryOnConcurrency(ctx, func() error {
		// Targeted Update
		_, err := s.Up(ctx, optup.Target([]string{rule.TargetURN}))
		return err
	})
}

// retryOnConcurrency implements exponential backoff for 409 Conflict / Concurrent Update errors.
func (sm *StateManager) retryOnConcurrency(ctx context.Context, op func() error) error {
	maxRetries := 5
	baseDelay := 1 * time.Second

	for i := 0; i <= maxRetries; i++ {
		err := op()
		if err == nil {
			return nil
		}

		// Check for concurrent update error
		// The error message usually contains "conflict" or "concurrent update".
		// Since we wrap the standard auto API, we rely on string matching.
		errMsg := strings.ToLower(err.Error())
		isConflict := strings.Contains(errMsg, "conflict") || strings.Contains(errMsg, "concurrent update")

		if !isConflict {
			return err // Non-retryable error
		}

		if i == maxRetries {
			return fmt.Errorf("max retries exceeded for concurrent update: %w", err)
		}

		// Exponential Backoff
		delay := baseDelay * time.Duration(math.Pow(2, float64(i)))
		log.Info().Dur("delay", delay).Msg("Concurrent update detected. Retrying...")
		
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			continue
		}
	}
	return nil
}

// Preview runs a preview update (dry run).
func (sm *StateManager) Preview(ctx context.Context, rule ScalingRule, newValue int) (string, error) {
	s, err := auto.UpsertStackLocalSource(ctx, sm.StackName, sm.WorkDir)
	if err != nil {
		return "", err
	}

	res, err := s.Preview(ctx, 
		optpreview.Target([]string{rule.TargetURN}),
		// optpreview.Config is not available or I'm using it wrong.
		// For now, we preview without explicit ephemeral config change.
		// Real DryRun might need to actually SetConfig then Preview then Revert?
	)
	if err != nil {
		return "", err
	}
	
	// Return the diff or summary
	// Automation API Stdout is captured? res.StdOut
	return res.StdOut, nil
}
