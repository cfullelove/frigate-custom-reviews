package engine

import (
	"fmt"
	"testing"
	"time"

	"frigate-stitcher/internal/models"
)

// MockMQTTPublisher captures messages for verification
type MockMQTTPublisher struct {
	PublishedMessages []struct {
		Topic   string
		Payload models.MessagePayload
	}
}

func (m *MockMQTTPublisher) Publish(topic string, payload interface{}) error {
	msg, ok := payload.(models.MessagePayload)
	if !ok {
		return fmt.Errorf("invalid payload type")
	}
	m.PublishedMessages = append(m.PublishedMessages, struct {
		Topic   string
		Payload models.MessagePayload
	}{Topic: topic, Payload: msg})
	return nil
}

func (m *MockMQTTPublisher) LastMessage() *models.MessagePayload {
	if len(m.PublishedMessages) == 0 {
		return nil
	}
	return &m.PublishedMessages[len(m.PublishedMessages)-1].Payload
}

func (m *MockMQTTPublisher) Clear() {
	m.PublishedMessages = []struct {
		Topic   string
		Payload models.MessagePayload
	}{}
}

func TestMatchesProfile(t *testing.T) {
	e := &Engine{}

	tests := []struct {
		name    string
		profile models.Profile
		state   models.FrigateEventState
		want    bool
	}{
		{
			name: "Exact Match",
			profile: models.Profile{
				Cameras: []string{"cam1"},
				Labels:  []string{"person"},
			},
			state: models.FrigateEventState{
				Camera: "cam1",
				Label:  "person",
			},
			want: true,
		},
		{
			name: "Camera Mismatch",
			profile: models.Profile{
				Cameras: []string{"cam1"},
				Labels:  []string{"person"},
			},
			state: models.FrigateEventState{
				Camera: "cam2",
				Label:  "person",
			},
			want: false,
		},
		{
			name: "Label Mismatch",
			profile: models.Profile{
				Cameras: []string{"cam1"},
				Labels:  []string{"person"},
			},
			state: models.FrigateEventState{
				Camera: "cam1",
				Label:  "dog",
			},
			want: false,
		},
		{
			name: "Zone Match",
			profile: models.Profile{
				Cameras:       []string{"cam1"},
				RequiredZones: []string{"zoneA"},
			},
			state: models.FrigateEventState{
				Camera:       "cam1",
				EnteredZones: []string{"zoneB", "zoneA"},
			},
			want: true,
		},
		{
			name: "Zone Match Only",
			profile: models.Profile{
				RequiredZones: []string{"zoneA"},
			},
			state: models.FrigateEventState{
				EnteredZones: []string{"zoneB", "zoneA"},
			},
			want: true,
		},
		{
			name: "Zone Mismatch",
			profile: models.Profile{
				Cameras:       []string{"cam1"},
				RequiredZones: []string{"zoneA"},
			},
			state: models.FrigateEventState{
				Camera:       "cam1",
				EnteredZones: []string{"zoneB", "zoneC"},
			},
			want: false,
		},
		{
			name: "Empty Profile (Wildcard)",
			profile: models.Profile{
				Cameras: []string{}, // Any
				Labels:  []string{}, // Any
			},
			state: models.FrigateEventState{
				Camera: "cam99",
				Label:  "ufo",
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := e.matchesProfile(tt.profile, tt.state); got != tt.want {
				t.Errorf("matchesProfile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEngine_Lifecycle(t *testing.T) {
	mockMQTT := &MockMQTTPublisher{}
	profile := models.Profile{
		Name:    "test_profile",
		Cameras: []string{"cam1"},
		Labels:  []string{"person"},
		Gap:     1,
	}

	engine := NewEngine([]models.Profile{profile}, mockMQTT, "test/review", WithPublishUpdates(true))

	// Phase 1: Start Event A
	evtA := models.FrigateEvent{
		Type: "new",
		After: models.FrigateEventState{
			ID:        "eventA",
			Camera:    "cam1",
			Label:     "person",
			StartTime: 1000,
		},
	}

	engine.handleEvent(evtA)

	if len(mockMQTT.PublishedMessages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(mockMQTT.PublishedMessages))
	}
	if mockMQTT.LastMessage().Type != "new" {
		t.Errorf("Expected 'new', got %s", mockMQTT.LastMessage().Type)
	}

	// Phase 2: Start Event B (Update)
	evtB := models.FrigateEvent{
		Type: "new",
		After: models.FrigateEventState{
			ID:        "eventB",
			Camera:    "cam1",
			Label:     "person",
			StartTime: 1005,
		},
	}
	engine.handleEvent(evtB)

	if len(mockMQTT.PublishedMessages) != 2 {
		t.Fatalf("Expected 2 messages, got %d", len(mockMQTT.PublishedMessages))
	}
	if mockMQTT.LastMessage().Type != "update" {
		t.Errorf("Expected 'update', got %s", mockMQTT.LastMessage().Type)
	}

	// Phase 3: End Events
	evtA.After.EndTime = 1010
	evtB.After.EndTime = 1015

	engine.handleEvent(evtA) // Updates state
	engine.handleEvent(evtB)

	// Check ReviewState
	// We need to verify internal state or message payload
	// The messages were updates.

	// Phase 4: Gap Expiry
	// Gap is 1s. Last end time is 1015 (timestamp).
	// Wait, the engine logic uses time.Now() vs LastEnd.
	// LastEnd in `toReviewState` comes from `evt.After.EndTime`.
	// `shouldClose` compares `time.Since(time.Unix(maxEndTime))` > Gap.
	// Since maxEndTime (1015) is way in the past relative to time.Now() (2025/2026), it should close immediately on tick.
	// To test properly with timestamps, we should use current timestamps.

	// Clean slate for strict timing test
}

func TestEngine_GapLogic(t *testing.T) {
	mockMQTT := &MockMQTTPublisher{}
	profile := models.Profile{
		Name:    "test_gap",
		Cameras: []string{"cam1"},
		Labels:  []string{"person"},
		Gap:     1, // 1 second gap
	}
	engine := NewEngine([]models.Profile{profile}, mockMQTT, "test/review")

	now := float64(time.Now().Unix())

	// Start event
	evt := models.FrigateEvent{
		After: models.FrigateEventState{
			ID:        "evt1",
			Camera:    "cam1",
			Label:     "person",
			StartTime: now,
		},
	}
	engine.handleEvent(evt)
	mockMQTT.Clear()

	// End event
	evt.After.EndTime = now // Ends "now"
	engine.handleEvent(evt)
	mockMQTT.Clear() // Clear "update" from end event

	// Tick immediately - Should NOT close because time.Since(now) < 1s
	engine.handleTick()
	if len(mockMQTT.PublishedMessages) > 0 {
		t.Errorf("Review closed too early! Messages: %v", mockMQTT.PublishedMessages)
	}

	// Wait 1.1s
	time.Sleep(1100 * time.Millisecond)

	// Tick - Should close
	engine.handleTick()

	if len(mockMQTT.PublishedMessages) == 0 {
		t.Fatal("Review failed to close after gap")
	}
	if mockMQTT.LastMessage().Type != "end" {
		t.Errorf("Expected 'end', got %s", mockMQTT.LastMessage().Type)
	}
}

func TestGhostEvents(t *testing.T) {
	mockMQTT := &MockMQTTPublisher{}
	profile := models.Profile{
		Name:    "test_ghost",
		Cameras: []string{"cam1"},
		Labels:  []string{"person"},
		Gap:     1,
	}

	engine := NewEngine([]models.Profile{profile}, mockMQTT, "test/review", WithPublishUpdates(true))
	engine.ghostTimeout = 10 * time.Millisecond

	// Start event
	evt := models.FrigateEvent{
		After: models.FrigateEventState{
			ID:        "ghost_evt",
			Camera:    "cam1",
			Label:     "person",
			StartTime: float64(time.Now().Unix()),
		},
	}
	engine.handleEvent(evt)
	mockMQTT.Clear()

	// Verify internal state has event
	review := engine.activeReviews["test_ghost"]
	tracked := review.Events["ghost_evt"]
	if tracked.Event.After.EndTime != 0 {
		t.Fatal("Event should be active")
	}

	// Wait for ghost timeout
	time.Sleep(20 * time.Millisecond)

	// Tick - Should detect ghost and close event
	engine.handleTick()

	// Expect an update message (ghost cleanup)
	if len(mockMQTT.PublishedMessages) == 0 {
		t.Fatal("Expected ghost update message")
	}
	lastMsg := mockMQTT.LastMessage()
	if lastMsg.Type != "update" {
		t.Errorf("Expected 'update', got %s", lastMsg.Type)
	}

	// Verify event logic closed
	if tracked.Event.After.EndTime == 0 {
		t.Error("Ghost event was not closed (EndTime is still 0)")
	}
}
