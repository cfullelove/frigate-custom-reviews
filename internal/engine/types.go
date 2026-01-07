package engine

import (
	"time"

	"frigate-stitcher/internal/models"
)

// TrackedEvent wraps the FrigateEvent with a local timestamp for ghost detection
type TrackedEvent struct {
	Event    *models.FrigateEvent
	LastSeen time.Time
}

// ReviewInstance tracks the runtime state of a stitched review
type ReviewInstance struct {
	ID           string
	Profile      models.Profile
	Events       map[string]*TrackedEvent // Key is Event ID
	LastEventEnd time.Time
	State        string // "active" or "ended"

	// Internal tracking
	LastUpdated    time.Time // Last time we touched this struct (wall clock)
	SentFirstEvent bool      // Whether we've emitted the 'new' message yet
}

type Engine struct {
	profiles       []models.Profile
	activeReviews  map[string]*ReviewInstance // Key is Profile Name
	ingestChan     chan models.FrigateEvent
	mqttClient     MQTTPublisher
	publishTopic   string
	publishUpdates bool
	ghostTimeout   time.Duration
}

// MQTTPublisher interface to decouple engine from specific mqtt implementation
type MQTTPublisher interface {
	Publish(topic string, payload interface{}) error
}
