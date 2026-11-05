package devices

import (
	"context"
	"log/slog"
	"time"

	"od/internal/models"
	"od/internal/parse"
)

// RohanAir is a Tuya air quality sensor read through the Tuya local protocol
type RohanAir struct {
	models.DeviceFields
	poller
	tuyaID  string
	tuyaKey string
}

func NewRohanAir(deps Deps, fields models.DeviceFields, tuyaID string, tuyaKey string) (*RohanAir, error) {
	device := &RohanAir{DeviceFields: fields, tuyaID: tuyaID, tuyaKey: tuyaKey}
	device.Type = "Rohan Air"
	device.Enabled = true
	device.init(deps, fields.ID)
	if err := device.start(fields.Periodicity, device.poll); err != nil {
		return nil, err
	}
	return device, nil
}

func (device *RohanAir) Stop() {
	device.stop()
}

func (device *RohanAir) poll(ctx context.Context) {
	tuya := NewTuya(device.tuyaID, device.tuyaKey)
	// When the device url holds the IP, Find() skips the UDP discovery
	tuya.IP = device.URL

	err := tuya.Find()
	if err == nil {
		err = tuya.Connect()
	}
	if err != nil {
		slog.Error("rohan air: connection failed", "device", device.Name, "host", device.URL, "err", err)
		return
	}
	defer tuya.Disconnect()

	for _, tag := range device.tags {
		if tag.Type != "humidity" {
			continue
		}
		raw, err := tuya.Get(3) // data point 3 is the humidity
		if err != nil {
			slog.Error("rohan air: reading the humidity failed", "device", device.Name, "host", device.URL, "err", err)
			return
		}
		value, err := parse.Float(raw)
		if err != nil {
			slog.Error("rohan air: humidity is not a number", "device", device.Name, "value", raw)
			return
		}
		slog.Info("rohan air: humidity", "device", device.Name, "value", value)
		device.record(ctx, tag, value, time.Now())
	}
}
