package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"frigate-custom-reviews/internal/config"
	"frigate-custom-reviews/internal/engine"
	"frigate-custom-reviews/internal/frigate"
	"frigate-custom-reviews/internal/logger"
	"frigate-custom-reviews/internal/mqtt"
)

func main() {
	configPath := flag.String("config", "config.yaml", "Path to configuration file")
	flag.Parse()

	// 1. Load Configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		logger.Fatalf("Error loading config: %v", err)
	}

	logger.SetLevel(cfg.Logging.Level)
	logger.Infof("Loaded config from %s", *configPath)

	// 2. Initialize Signal Handling early
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// 3. Initialize Clients
	mqttClient := mqtt.NewClient(cfg.MQTT)
	frigateClient := frigate.NewClient(cfg.Frigate)

	// 4. Initialize Engine
	eng := engine.NewEngine(cfg.Profiles, mqttClient, cfg.MQTT.ReviewsPublishTopic, engine.WithGhostTimeout(cfg.GhostTimeout), engine.WithPublishUpdates(cfg.PublishUpdates))

	// 5. Recover State from Frigate API
	logger.Info("Querying Frigate API for active events...")
	activeEvents, err := frigateClient.GetActiveEvents()
	if err != nil {
		logger.Warnf("Failed to query Frigate API: %v", err)
	} else {
		logger.Infof("Found %d active events from API", len(activeEvents))
		ingest := eng.IngestChannel()
		for _, evt := range activeEvents {
			ingest <- evt
		}
	}

	// 6. Connect to MQTT (Asynchronous to allow engine to start and signals to be handled)
	go func() {
		logger.Info("Connecting to MQTT broker...")
		if err := mqttClient.Connect(); err != nil {
			logger.Errorf("MQTT connection failed: %v", err)
		}
	}()
	defer mqttClient.Disconnect()

	// 7. Subscribe to Frigate Events
	// This will automatically subscribe when the connection is established
	if err := mqttClient.Subscribe(eng.IngestChannel()); err != nil {
		logger.Errorf("Failed to setup MQTT subscription: %v", err)
	}

	// 8. Start Engine
	go eng.Run()

	// 9. Wait for Signal
	sig := <-sigChan
	logger.Infof("Received signal %v, shutting down...", sig)
}
