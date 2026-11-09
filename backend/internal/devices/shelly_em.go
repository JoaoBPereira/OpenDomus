package devices

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"od/internal/models"
)

type shellyEMStatus struct {
	Emeters []shellyMeter `json:"emeters"`
}

type ShellyEM struct {
	models.DeviceFields
	poller
}

func NewShellyEM(deps Deps, fields models.DeviceFields) (*ShellyEM, error) {
	device := &ShellyEM{DeviceFields: fields}
	device.Type = "Shelly EM"
	device.Enabled = true
	device.init(deps, fields.ID)
	if err := device.start(fields.Periodicity, device.poll); err != nil {
		return nil, err
	}
	return device, nil
}

func (device *ShellyEM) Stop() {
	device.stop()
}

func (device *ShellyEM) poll(ctx context.Context) {
	body, _, err := device.deps.HTTP.Get(ctx, "http://"+device.URL+"/status")
	if err != nil {
		slog.Error("shelly em: status request failed", "device", device.Name, "host", device.URL, "err", err)
		return
	}

	var status shellyEMStatus
	if err := json.Unmarshal(body, &status); err != nil {
		slog.Error("shelly em: invalid status answer", "device", device.Name, "host", device.URL, "err", err)
		return
	}

	now := time.Now()
	for _, tag := range device.tags {
		config, err := parseShellyTagConfig(tag)
		if err != nil {
			slog.Error("shelly em: invalid tag config", "tag", tag.ID, "err", err)
			continue
		}
		if config.Emeter == nil {
			continue
		}
		if value, found := meterValue(status.Emeters, config.Emeter); found {
			device.record(ctx, tag, value, now)
		}
	}
}
