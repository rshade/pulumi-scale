package autoscaler

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/rshade/pulumi-scale/internal/webhooks"
)

// Engine is responsible for processing ScalingIntents and triggering state updates.
type Engine struct {
	Rules       map[string]ScalingRule
	State       *StateManager // To be implemented in US3
	LastScaled  map[string]time.Time
	mu          sync.Mutex
	IntentChan  chan webhooks.ScalingIntent
}

func NewEngine(rules map[string]ScalingRule, state *StateManager) *Engine {
	return &Engine{
		Rules:      rules,
		State:      state,
		LastScaled: make(map[string]time.Time),
		IntentChan: make(chan webhooks.ScalingIntent, 100),
	}
}

func (e *Engine) Start(ctx context.Context) {
	log.Info().Msg("Engine started, waiting for intents...")
	for {
		select {
		case <-ctx.Done():
			return
		case intent := <-e.IntentChan:
			e.ProcessIntent(ctx, intent)
		}
	}
}

func (e *Engine) ProcessIntent(ctx context.Context, intent webhooks.ScalingIntent) {
	e.mu.Lock()
	defer e.mu.Unlock()

	log.Info().
		Str("pool", intent.TargetPool).
		Str("action", string(intent.Action)).
		Int("value", intent.Value).
		Str("reason", intent.Reason).
		Msg("Processing intent")

	rule, ok := e.Rules[intent.TargetPool]
	if !ok {
		log.Error().Str("pool", intent.TargetPool).Msg("No rule found for pool")
		return
	}

	// Cooldown Check (T018)
	if !e.checkCooldown(rule) {
		log.Info().Str("pool", intent.TargetPool).Msg("Skipping intent: Cooldown active")
		return
	}

	// Retrieve Current Count
	current, err := e.State.GetCurrentCount(ctx, rule.ConfigKey)
	if err != nil {
		// Log warning, but maybe proceed if ActionSet?
		// If ActionDelta, we MUST have current.
		if intent.Action == webhooks.ActionDelta {
			log.Error().Err(err).Msg("Error getting current count for delta scaling")
			return
		}
		// If ActionSet, we might not strictly need current, but good for logging.
		log.Warn().Err(err).Msg("Could not get current count. Assuming unknown.")
	}

	var target int
	if intent.Action == webhooks.ActionSet {
		target = intent.Value
	} else {
		target = current + intent.Value
	}

	// Guardrails
	if target < rule.Min {
		target = rule.Min
	}
	if target > rule.Max {
		target = rule.Max
	}

	log.Info().
		Str("pool", intent.TargetPool).
		Int("target", target).
		Int("current", current).
		Msg("Calculated target")

	if target == current {
		log.Info().Msg("Target equals current. No change needed.")
		return
	}

	// Apply State
	if intent.DryRun {
		log.Info().Int("target", target).Msg("DryRun detected. Previewing scale...")
		diff, err := e.State.Preview(ctx, rule, target)
		if err != nil {
			log.Error().Err(err).Msg("Error previewing scaling")
			return
		}
		log.Info().Msgf("DryRun Result:\n%s", diff)
		// Do not update LastScaled or persist
		return
	}

	startTime := time.Now()
	if err := e.State.Apply(ctx, rule, target); err != nil {
		log.Error().Err(err).Msg("Error applying scaling")
		return
	}
	duration := time.Since(startTime)
	log.Info().
		Str("pool", intent.TargetPool).
		Int("target", target).
		Dur("duration", duration).
		Msg("Successfully scaled")

	e.LastScaled[rule.PoolName] = time.Now()
}

func (e *Engine) checkCooldown(rule ScalingRule) bool {
	last, ok := e.LastScaled[rule.PoolName]
	if !ok {
		return true // Never scaled
	}
	if time.Since(last) < time.Duration(rule.CooldownSeconds)*time.Second {
		return false
	}
	return true
}
