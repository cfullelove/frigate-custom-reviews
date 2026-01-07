package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"frigate-custom-reviews/internal/config"
	"frigate-custom-reviews/internal/engine"
	"frigate-custom-reviews/internal/frigate"
	"frigate-custom-reviews/internal/mqtt"
)

func main() {
	configPath := flag.String("config", "config.yaml", "Path to configuration file")
	flag.Parse()

	// 1. Load Configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}
	log.Printf("Loaded config from %s", *configPath)

	// 2. Initialize Clients
	mqttClient := mqtt.NewClient(cfg.MQTT)
	frigateClient := frigate.NewClient(cfg.Frigate)

	// 3. Initialize Engine
	eng := engine.NewEngine(cfg.Profiles, mqttClient, cfg.MQTT.ReviewsPublishTopic, engine.WithGhostTimeout(cfg.GhostTimeout), engine.WithPublishUpdates(cfg.PublishUpdates))

	// 4. Recover State from Frigate API
	log.Println("Querying Frigate API for active events...")
	activeEvents, err := frigateClient.GetActiveEvents()
	if err != nil {
		log.Printf("Warning: Failed to query Frigate API: %v", err)
	} else {
		log.Printf("Found %d active events from API", len(activeEvents))
		ingest := eng.IngestChannel()
		for _, evt := range activeEvents {
			ingest <- evt
		}
	}

	// 5. Connect to MQTT
	if err := mqttClient.Connect(); err != nil {
		log.Fatalf("Failed to connect to MQTT: %v", err)
	}
	defer mqttClient.Disconnect()

	// 6. Subscribe to Frigate Events
	// We pass the engine's ingest channel directly to the MQTT subscriber
	if err := mqttClient.Subscribe(eng.IngestChannel()); err != nil {
		log.Fatalf("Failed to subscribe to topic: %v", err)
	}

	// 7. Start Engine (Blocking or Non-blocking? Engine.Run is blocking)
	// We run it in a goroutine so we can handle signals
	go eng.Run()

	// 8. Wait for Signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	sig := <-sigChan
	log.Printf("Received signal %v, shutting down...", sig)
}
