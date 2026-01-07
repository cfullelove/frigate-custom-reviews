package engine

import (
	"log"
	"slices"
	"time"

	"frigate-stitcher/internal/models"

	"github.com/google/uuid"
)

type EngineOption func(*Engine)

func WithPublishUpdates(publish bool) EngineOption {
	return func(e *Engine) {
		e.publishUpdates = publish
	}
}

func WithGhostTimeout(timeout int) EngineOption {
	return func(e *Engine) {
		e.ghostTimeout = time.Duration(timeout) * time.Second
	}
}

func NewEngine(profiles []models.Profile, mqttClient MQTTPublisher, publishTopic string, opts ...EngineOption) *Engine {
	engine := &Engine{
		profiles:      profiles,
		activeReviews: make(map[string]*ReviewInstance),
		ingestChan:    make(chan models.FrigateEvent, 100),
		mqttClient:    mqttClient,
		publishTopic:  publishTopic,
	}

	for _, opt := range opts {
		opt(engine)
	}

	return engine
}

func (e *Engine) IngestChannel() chan<- models.FrigateEvent {
	return e.ingestChan
}

func (e *Engine) Run() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	log.Println("Engine started")

	for {
		select {
		case evt := <-e.ingestChan:
			e.handleEvent(evt)
		case <-ticker.C:
			e.handleTick()
		}
	}
}

func (e *Engine) handleEvent(evt models.FrigateEvent) {
	state := evt.After

	for _, profile := range e.profiles {
		if !e.matchesProfile(profile, state) {
			continue
		}

		review, exists := e.activeReviews[profile.Name]

		if !exists {
			review = &ReviewInstance{
				ID:           uuid.NewString(),
				Profile:      profile,
				Events:       make(map[string]*TrackedEvent),
				State:        "active",
				LastUpdated:  time.Now(),
				LastEventEnd: time.Time{},
			}
			e.activeReviews[profile.Name] = review
		}

		var beforeState *models.ReviewState
		if review.SentFirstEvent {
			s := e.toReviewState(review)
			beforeState = &s
		}

		// Update Review State
		evtCopy := evt
		review.Events[state.ID] = &TrackedEvent{
			Event:    &evtCopy,
			LastSeen: time.Now(),
		}
		review.LastUpdated = time.Now()

		afterState := e.toReviewState(review)

		payloadType := "new"
		if review.SentFirstEvent {
			payloadType = "update"
		}

		if !e.publishUpdates {
			return
		}

		msg := models.MessagePayload{
			Type:   payloadType,
			Before: beforeState,
			After:  &afterState,
		}

		if err := e.mqttClient.Publish(e.publishTopic, msg); err != nil {
			log.Printf("Error publishing review update: %v", err)
		} else {
			log.Printf("[MQTT] Published '%s' for Review %s (Profile: %s). Events: %d",
				payloadType, review.ID, review.Profile.Name, len(review.Events))
			review.SentFirstEvent = true
		}
	}
}

func (e *Engine) handleTick() {
	for name, review := range e.activeReviews {
		// 1. Check for Ghost Events
		updatedReview := false
		for id, tracked := range review.Events {
			// If event is active (EndTime == 0) and stale
			if tracked.Event.After.EndTime == 0 && time.Since(tracked.LastSeen) > e.ghostTimeout {
				log.Printf("Ghost event detected: %s in review %s. Closing event.", id, review.ID)
				log.Printf("[Debug] %v, %v ", time.Since(tracked.LastSeen), e.ghostTimeout)

				// Force close the event
				// We set EndTime to the timestamp of when it went stale (approx now)
				nowUnix := float64(time.Now().Unix())
				tracked.Event.After.EndTime = nowUnix
				// We don't update LastSeen as we want it to remain 'processed'
				updatedReview = true
			}
		}

		if updatedReview && e.publishUpdates {
			// If we modified events, we should publish an update
			// (Use 'update' message)
			currentState := e.toReviewState(review)
			msg := models.MessagePayload{
				Type:   "update",
				Before: nil, // We could calculate before, but for ghost cleanup current state is vital
				After:  &currentState,
			}
			if err := e.mqttClient.Publish(e.publishTopic, msg); err == nil {
				log.Printf("[MQTT] Published 'update' (ghost cleanup) for Review %s", review.ID)
			}
		}

		// 2. Check if we should close the review
		if e.shouldClose(review) {
			log.Printf("Closing review %s (Profile: %s)", review.ID, name)

			beforeState := e.toReviewState(review)

			review.State = "ended"
			afterState := e.toReviewState(review)

			msg := models.MessagePayload{
				Type:   "end",
				Before: &beforeState,
				After:  &afterState,
			}

			if err := e.mqttClient.Publish(e.publishTopic, msg); err != nil {
				log.Printf("Error publishing review end: %v", err)
			} else {
				log.Printf("[MQTT] Published 'end' for Review %s (Profile: %s)", review.ID, name)
			}

			delete(e.activeReviews, name)
		}
	}
}

func (e *Engine) shouldClose(r *ReviewInstance) bool {
	activeCount := 0
	var maxEndTime float64 = 0

	for _, tracked := range r.Events {
		evt := tracked.Event

		if evt.After.EndTime == 0 {
			activeCount++
		} else {
			if evt.After.EndTime > maxEndTime {
				maxEndTime = evt.After.EndTime
			}
		}
	}

	if activeCount > 0 {
		return false
	}

	log.Printf("[Debug] Entering Gap for review %s", r.ID)

	lastEnd := time.Unix(int64(maxEndTime), 0)
	waited := time.Since(lastEnd)

	return waited.Seconds() > float64(r.Profile.Gap)
}

func zonesOverlap(a, b []string) bool {
	for _, v := range a {
		if slices.Contains(b, v) {
			return true
		}
	}
	return false
}

func (e *Engine) matchesProfile(p models.Profile, state models.FrigateEventState) bool {
	if len(p.Cameras) > 0 && !slices.Contains(p.Cameras, state.Camera) {
		return false
	}

	if len(p.Labels) > 0 && !slices.Contains(p.Labels, state.Label) {
		return false
	}

	if len(p.RequiredZones) > 0 && len(state.EnteredZones) > 0 && !zonesOverlap(p.RequiredZones, state.EnteredZones) {
		return false
	}

	return true
}

func (e *Engine) toReviewState(r *ReviewInstance) models.ReviewState {
	var minStart float64 = 0
	var maxEnd float64 = 0
	activeEvents := 0
	linkedEvents := []models.LinkedEventSummary{}
	objectsSet := make(map[string]bool)

	first := true
	allEnded := true

	for _, tracked := range r.Events {
		evt := tracked.Event
		state := evt.After

		if first || state.StartTime < minStart {
			minStart = state.StartTime
		}

		if state.EndTime == 0 {
			allEnded = false
			activeEvents++
		} else {
			if state.EndTime > maxEnd {
				maxEnd = state.EndTime
			}
		}

		linkedEvents = append(linkedEvents, models.LinkedEventSummary{
			ID:     state.ID,
			Camera: state.Camera,
		})
		objectsSet[state.Label] = true
		first = false
	}

	objects := []string{}
	for k := range objectsSet {
		objects = append(objects, k)
	}

	out := models.ReviewState{
		ID:           r.ID,
		ProfileName:  r.Profile.Name,
		State:        r.State,
		StartTime:    minStart,
		EventCount:   len(r.Events),
		ActiveEvents: activeEvents,
		LinkedEvents: linkedEvents,
		Objects:      objects,
	}

	if allEnded && len(r.Events) > 0 {
		out.EndTime = &maxEnd
	}

	return out
}
