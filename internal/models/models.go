package models

// Config defines the user settings
type Config struct {
	MQTT     MQTTConfig    `yaml:"mqtt"`
	Frigate  FrigateConfig `yaml:"frigate"`
	Profiles []Profile     `yaml:"profiles"`
}

type MQTTConfig struct {
	Broker   string `yaml:"broker"`
	ClientID string `yaml:"client_id"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Topic    string `yaml:"topic"`
}

type FrigateConfig struct {
	URL string `yaml:"url"`
}

type Profile struct {
	Name    string   `yaml:"name"`    // "front_yard"
	Cameras []string `yaml:"cameras"` // ["doorbell", "driveway"]
	Objects []string `yaml:"objects"` // ["person", "dog"]
	Gap     int      `yaml:"gap"`     // 30
}

// ReviewState represents the "Data" block in the JSON payload
type ReviewState struct {
	ID           string   `json:"id"`
	ProfileName  string   `json:"profile_name"`
	State        string   `json:"state"` // "active" or "ended"
	StartTime    float64  `json:"start_time"`
	EndTime      *float64 `json:"end_time,omitempty"`
	EventCount   int      `json:"event_count"`
	ActiveEvents int      `json:"active_events"`
	LinkedEvents []string `json:"linked_events"` // List of Frigate IDs
	Objects      []string `json:"objects"`
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
	ID        string  `json:"id"`
	Camera    string  `json:"camera"`
	Label     string  `json:"label"`
	StartTime float64 `json:"start_time"`
	EndTime   float64 `json:"end_time,omitempty"` // 0 or null if active? usually 0 or missing in Frigate
}
