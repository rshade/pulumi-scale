package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/rshade/pulumi-scale/internal/autoscaler"
)

func main() {
	stackName := flag.String("stack", "dev", "The name of the Pulumi stack")
	workDir := flag.String("workdir", ".", "The directory containing the Pulumi program")
	port := flag.Int("port", 8080, "The port to listen on")
	debug := flag.Bool("debug", false, "Enable debug logging")
	flag.Parse()

	// Configure Zerolog
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	if *debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Info().Msg("Received shutdown signal")
		cancel()
	}()

	// Load configuration
	log.Info().Str("stack", *stackName).Str("workdir", *workDir).Msg("Loading scaling rules...")
	loader := autoscaler.NewConfigLoader(*stackName, *workDir)
	rules, err := loader.LoadRules(ctx)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to load rules. (Ensure stack exists and has outputs)")
	} else {
		log.Info().Int("count", len(rules)).Msg("Loaded scaling rules")
		for name, rule := range rules {
			log.Info().
				Str("pool", name).
				Str("targetUrn", rule.TargetURN).
				Int("min", rule.Min).
				Int("max", rule.Max).
				Msg("Rule loaded")
		}
	}

	// Initialize StateManager
	stateManager := autoscaler.NewStateManager(*stackName, *workDir)

	// Initialize Engine
	engine := autoscaler.NewEngine(rules, stateManager)
	go engine.Start(ctx)

	// TODO: Load Auth Token from Pulumi Config (future task)

	server := NewServer(*port)
	// TODO: Apply Auth Middleware to protected routes (future task when wiring routers)

	log.Info().Int("port", *port).Msg("Starting PulumiScale server...")
	if err := server.Start(ctx); err != nil {
		log.Fatal().Err(err).Msg("Server failed")
	}
}
