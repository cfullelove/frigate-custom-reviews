package models

// Config defines the user settings
type Config struct {
	MQTT           MQTTConfig    `yaml:"mqtt"`
	Frigate        FrigateConfig `yaml:"frigate"`
	Logging        LoggingConfig `yaml:"logging"`
	Profiles       []Profile     `yaml:"profiles"`
	PublishUpdates bool          `yaml:"publish_updates"`
	GhostTimeout   int           `yaml:"event_timeout"`
}

type LoggingConfig struct {
	Level string `yaml:"level"`
}

type MQTTConfig struct {
	Broker              string `yaml:"broker"`
	ClientID            string `yaml:"client_id"`
	User                string `yaml:"user"`
	Password            string `yaml:"password"`
	FrigateEventsTopic  string `yaml:"frigate_events_topic"`
	ReviewsPublishTopic string `yaml:"reviews_publish_topic"`
}

type FrigateConfig struct {
	URL string `yaml:"url"`
}

type Profile struct {
	Name          string   `yaml:"name"`           // "front_yard"
	Cameras       []string `yaml:"cameras"`        // ["doorbell", "driveway"]
	Labels        []string `yaml:"labels"`         // ["person", "dog"]
	RequiredZones []string `yaml:"required_zones"` // ["driveway", "road"]
	Gap           int      `yaml:"gap"`            // 30
}

type LinkedEventSummary struct {
	ID     string `json:"id"`
	Camera string `json:"camera"`
}

// ReviewState represents the "Data" block in the JSON payload
type ReviewState struct {
	ID           string               `json:"id"`
	ProfileName  string               `json:"profile_name"`
	State        string               `json:"state"` // "active" or "ended"
	StartTime    float64              `json:"start_time"`
	EndTime      *float64             `json:"end_time,omitempty"`
	EventCount   int                  `json:"event_count"`
	ActiveEvents int                  `json:"active_events"`
	LinkedEvents []LinkedEventSummary `json:"linked_events"` // List of Frigate IDs with Camera
	Objects      []string             `json:"objects"`
	Cameras      []string             `json:"cameras"`
	Zones        []string             `json:"zones"`
}

// MessagePayload represents the actual MQTT message
type MessagePayload struct {
	Type   string       `json:"type"` // "new", "update", "end"
	Before *ReviewState `json:"before"`
	After  *ReviewState `json:"after"`
}

// FrigateEvent matches the Frigate JSON payload
type FrigateEvent struct {
	Type   string            `json:"type"`
	Before FrigateEventState `json:"before"`
	After  FrigateEventState `json:"after"`
}

type FrigateEventState struct {
	ID           string   `json:"id"`
	Camera       string   `json:"camera"`
	Label        string   `json:"label"`
	StartTime    float64  `json:"start_time"`
	EndTime      float64  `json:"end_time,omitempty"` // 0 or null if active? usually 0 or missing in Frigate
	CurrentZones []string `json:"current_zones"`
	EnteredZones []string `json:"entered_zones"`
}
