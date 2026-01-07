package mqtt

import (
	"encoding/json"
	"fmt"
	"log"

	"frigate-custom-reviews/internal/models"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type Client struct {
	client mqtt.Client
	config models.MQTTConfig
}

func NewClient(cfg models.MQTTConfig) *Client {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(cfg.Broker)
	opts.SetClientID(cfg.ClientID)

	if cfg.User != "" {
		opts.SetUsername(cfg.User)
		opts.SetPassword(cfg.Password)
	}

	opts.SetAutoReconnect(true)
	opts.SetOnConnectHandler(func(c mqtt.Client) {
		log.Printf("Connected to MQTT broker at %s", cfg.Broker)
	})
	opts.SetConnectionLostHandler(func(c mqtt.Client, err error) {
		log.Printf("Lost connection to MQTT broker: %v", err)
	})

	client := mqtt.NewClient(opts)
	return &Client{
		client: client,
		config: cfg,
	}
}

func (c *Client) Connect() error {
	token := c.client.Connect()
	if token.Wait() && token.Error() != nil {
		return token.Error()
	}
	return nil
}

func (c *Client) Subscribe(ingestChan chan<- models.FrigateEvent) error {
	token := c.client.Subscribe(c.config.FrigateEventsTopic, 0, func(client mqtt.Client, msg mqtt.Message) {
		var event models.FrigateEvent
		if err := json.Unmarshal(msg.Payload(), &event); err != nil {
			log.Printf("Failed to unmarshal Frigate event: %v", err)
			return
		}
		ingestChan <- event
	})

	if token.Wait() && token.Error() != nil {
		return token.Error()
	}

	log.Printf("Subscribed to topic: %s", c.config.FrigateEventsTopic)
	return nil
}

func (c *Client) Publish(topic string, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	token := c.client.Publish(topic, 0, false, data)
	if token.Wait() && token.Error() != nil {
		return token.Error()
	}
	return nil
}

func (c *Client) Disconnect() {
	c.client.Disconnect(250)
}
