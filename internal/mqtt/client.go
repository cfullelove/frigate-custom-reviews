package mqtt

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"frigate-custom-reviews/internal/logger"
	"frigate-custom-reviews/internal/models"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type Client struct {
	client     mqtt.Client
	config     models.MQTTConfig
	ingestChan chan<- models.FrigateEvent
	mu         sync.Mutex
}

func NewClient(cfg models.MQTTConfig) *Client {
	c := &Client{
		config: cfg,
	}

	opts := mqtt.NewClientOptions()
	opts.AddBroker(cfg.Broker)
	opts.SetClientID(cfg.ClientID)

	if cfg.User != "" {
		opts.SetUsername(cfg.User)
		opts.SetPassword(cfg.Password)
	}

	opts.SetAutoReconnect(true)
	opts.SetConnectRetry(true)
	opts.SetConnectRetryInterval(5 * time.Second)

	opts.SetOnConnectHandler(func(client mqtt.Client) {
		logger.Infof("Connected to MQTT broker at %s", cfg.Broker)
		c.mu.Lock()
		defer c.mu.Unlock()
		if c.ingestChan != nil {
			c.doSubscribe()
		}
	})
	opts.SetConnectionLostHandler(func(client mqtt.Client, err error) {
		logger.Warnf("Lost connection to MQTT broker: %v", err)
	})

	c.client = mqtt.NewClient(opts)
	return c
}

func (c *Client) Connect() error {
	token := c.client.Connect()
	if token.Wait() && token.Error() != nil {
		return token.Error()
	}
	return nil
}

func (c *Client) Subscribe(ingestChan chan<- models.FrigateEvent) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ingestChan = ingestChan
	if c.client.IsConnected() {
		c.doSubscribe()
	}
	return nil
}

func (c *Client) doSubscribe() {
	token := c.client.Subscribe(c.config.FrigateEventsTopic, 0, func(client mqtt.Client, msg mqtt.Message) {
		var event models.FrigateEvent
		if err := json.Unmarshal(msg.Payload(), &event); err != nil {
			logger.Errorf("Failed to unmarshal Frigate event: %v", err)
			return
		}
		c.ingestChan <- event
	})

	go func() {
		if token.Wait() && token.Error() != nil {
			logger.Errorf("Failed to subscribe to topic %s: %v", c.config.FrigateEventsTopic, token.Error())
		} else {
			logger.Infof("Subscribed to topic: %s", c.config.FrigateEventsTopic)
		}
	}()
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
