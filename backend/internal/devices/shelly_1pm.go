package devices

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"od/internal/models"
)

type shelly1PMStatus struct {
	Meters         []shellyMeter `json:"meters"`
	ExtTemperature map[string]struct {
		TC float64 `json:"tC"`
	} `json:"ext_temperature"`
}

type Shelly1PM struct {
	models.DeviceFields
	poller
}

func NewShelly1PM(deps Deps, fields models.DeviceFields) (*Shelly1PM, error) {
	device := &Shelly1PM{DeviceFields: fields}
	device.Type = "Shelly 1 PM"
	device.Enabled = true
	device.init(deps, fields.ID)
	if err := device.start(fields.Periodicity, device.poll); err != nil {
		return nil, err
	}
	return device, nil
}

func (device *Shelly1PM) Stop() {
	device.stop()
}

func (device *Shelly1PM) poll(ctx context.Context) {
	body, _, err := device.deps.HTTP.Get(ctx, "http://"+device.URL+"/status")
	if err != nil {
		slog.Error("shelly 1pm: status request failed", "device", device.Name, "host", device.URL, "err", err)
		return
	}
	slog.Info("shelly 1pm: status", "device", device.Name, "host", device.URL, "body", string(body))

	var status shelly1PMStatus
	if err := json.Unmarshal(body, &status); err != nil {
		slog.Error("shelly 1pm: invalid status answer", "device", device.Name, "host", device.URL, "err", err)
		return
	}

	now := time.Now()
	for _, tag := range device.tags {
		config, err := parseShellyTagConfig(tag)
		if err != nil {
			slog.Error("shelly 1pm: invalid tag config", "tag", tag.ID, "err", err)
			continue
		}
		switch {
		case (tag.Type == "currentPower" || tag.Type == "totalPower") && config.Emeter != nil:
			if value, found := meterValue(status.Meters, config.Emeter); found {
				device.record(ctx, tag, value, now)
			}
		case tag.Type == "temperature" && config.Temperature != nil:
			sensor, found := status.ExtTemperature[config.Temperature.Index.String()]
			if !found {
				slog.Error("shelly 1pm: no external temperature sensor with that index", "host", device.URL, "index", config.Temperature.Index.String())
				continue
			}
			device.record(ctx, tag, sensor.TC, now)
		}
	}
}
