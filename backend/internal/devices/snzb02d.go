package devices

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"od/internal/models"
)

// SNZB02D is a Zigbee temperature sensor read through Zigbee2MQTT: the device url is the broker
// address (mqtt://host:1883) and the config is the topic to subscribe to
type SNZB02D struct {
	models.DeviceFields
	deps   Deps
	tags   []*models.Tag
	client mqtt.Client
}

func NewSNZB02D(deps Deps, fields models.DeviceFields) *SNZB02D {
	device := &SNZB02D{DeviceFields: fields, deps: deps, tags: deps.Tags.TagsByDeviceID(fields.ID)}
	device.Type = "ZigBee2MQTT"
	device.Enabled = true
	device.connect()
	return device
}

func (device *SNZB02D) connect() {
	options := mqtt.NewClientOptions().AddBroker(device.URL)
	options.SetAutoReconnect(true)
	options.SetConnectRetry(true)

	options.SetConnectionLostHandler(func(client mqtt.Client, err error) {
		slog.Warn("snzb02d: mqtt connection lost", "device", device.Name, "err", err)
	})

	options.SetOnConnectHandler(func(client mqtt.Client) {
		slog.Info("snzb02d: connected to the mqtt broker", "device", device.Name, "broker", device.URL)
		token := client.Subscribe(device.Config, 0, device.onMessage)
		token.Wait()
		if token.Error() != nil {
			slog.Error("snzb02d: subscription failed", "device", device.Name, "topic", device.Config, "err", token.Error())
			return
		}
		slog.Info("snzb02d: subscribed", "device", device.Name, "topic", device.Config)
	})

	device.client = mqtt.NewClient(options)
	token := device.client.Connect()
	go func() {
		token.Wait()
		if token.Error() != nil {
			slog.Error("snzb02d: mqtt connection failed", "device", device.Name, "broker", device.URL, "err", token.Error())
		}
	}()
}

func (device *SNZB02D) onMessage(client mqtt.Client, message mqtt.Message) {
	if message.Topic() != device.Config {
		return
	}
	payload := string(message.Payload())
	slog.Info("snzb02d: status received", "device", device.Name, "topic", device.Config, "payload", payload)

	var data map[string]any
	if err := json.Unmarshal(message.Payload(), &data); err != nil {
		slog.Error("snzb02d: invalid mqtt message", "device", device.Name, "topic", device.Config, "err", err)
		return
	}
	if data["temperature"] == nil {
		slog.Warn("snzb02d: no temperature in the payload", "device", device.Name, "topic", device.Config)
		return
	}
	tag := device.findTag("temperature")
	if tag == nil {
		slog.Warn("snzb02d: no temperature tag for this device", "device", device.Name)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), runTimeout)
	defer cancel()
	if err := device.deps.Tags.SaveTagValue(ctx, tag, data["temperature"], time.Now()); err != nil {
		slog.Error("saving tag value failed", "tag", tag.Name, "err", err)
	}
}

// findTag returns the first tag of the given type, or nil
func (device *SNZB02D) findTag(tagType string) *models.Tag {
	for _, tag := range device.tags {
		if tag.Type == tagType {
			return tag
		}
	}
	return nil
}

func (device *SNZB02D) Stop() {
	if device.client != nil {
		device.client.Disconnect(250)
	}
}
