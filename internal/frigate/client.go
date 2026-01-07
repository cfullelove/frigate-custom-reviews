package frigate

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"frigate-stitcher/internal/models"
)

type Client struct {
	config models.FrigateConfig
	client *http.Client
}

func NewClient(cfg models.FrigateConfig) *Client {
	return &Client{
		config: cfg,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// FrigateAPIEvent reflects the raw API response which is slightly different from the MQTT "after" payload
// e.g. "top_score" vs "score", but for our purposes (id, camera, label, start/end) matches enough.
type FrigateAPIEvent struct {
	ID        string   `json:"id"`
	Camera    string   `json:"camera"`
	Label     string   `json:"label"`
	StartTime float64  `json:"start_time"`
	EndTime   *float64 `json:"end_time"` // Pointer to handle null
}

func (c *Client) GetActiveEvents() ([]models.FrigateEvent, error) {
	url := fmt.Sprintf("%s/api/events?in_progress=1", c.config.URL)
	
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to query frigate API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("api active check returned status: %d", resp.StatusCode)
	}

	var apiEvents []FrigateAPIEvent
	if err := json.NewDecoder(resp.Body).Decode(&apiEvents); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	var events []models.FrigateEvent
	for _, ae := range apiEvents {
		// Convert API struct to our internal Model
		// The API represents the "current state", so we map it to "After"
		
		endTime := 0.0
		if ae.EndTime != nil {
			endTime = *ae.EndTime
		}

		evt := models.FrigateEvent{
			Type: "update", // Assume update for existing ongoing events
			After: models.FrigateEventState{
				ID:        ae.ID,
				Camera:    ae.Camera,
				Label:     ae.Label,
				StartTime: ae.StartTime,
				EndTime:   endTime,
			},
		}
		events = append(events, evt)
	}

	return events, nil
}
