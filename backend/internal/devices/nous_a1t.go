package devices

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"od/internal/models"
)

// Tasmota "Status 8" answer: {"StatusSNS": {"Time": "...", "ENERGY": {"Power": 12, "Total": 345.6, ...}}}
type tasmotaStatus8 struct {
	StatusSNS struct {
		ENERGY *struct {
			Power float64 `json:"Power"`
			Total float64 `json:"Total"`
		} `json:"ENERGY"`
	} `json:"StatusSNS"`
}

// Tasmota "Power" answer: {"POWER": "ON"} or {"POWER": "OFF"}
type tasmotaPower struct {
	POWER string `json:"POWER"`
}

// NousA1T is a Nous A1T plug running the Tasmota firmware
type NousA1T struct {
	models.PowerDeviceFields
	poller
}

func NewNousA1T(deps Deps, fields models.DeviceFields) (*NousA1T, error) {
	device := &NousA1T{PowerDeviceFields: models.PowerDeviceFields{DeviceFields: fields}}
	device.Type = "NOUS A1T"
	device.Enabled = true
	device.init(deps, fields.ID)
	device.SetPowerTags(device.tags)
	if err := device.start(fields.Periodicity, device.poll); err != nil {
		return nil, err
	}
	return device, nil
}

func (device *NousA1T) Stop() {
	device.stop()
}

func (device *NousA1T) poll(ctx context.Context) {
	// Status 8 gives power, total energy and sensor data
	body, _, err := device.deps.HTTP.Get(ctx, "http://"+device.URL+"/cm?cmnd=Status%208")
	if err != nil {
		slog.Error("nous a1t: status request failed", "device", device.Name, "host", device.URL, "err", err)
		return
	}
	slog.Info("nous a1t: status", "device", device.Name, "host", device.URL, "body", string(body))

	var status tasmotaStatus8
	if err := json.Unmarshal(body, &status); err != nil {
		slog.Error("nous a1t: invalid status answer", "device", device.Name, "host", device.URL, "err", err)
		return
	}
	energy := status.StatusSNS.ENERGY
	if energy == nil {
		slog.Error("nous a1t: no energy data in the Tasmota answer", "device", device.Name, "host", device.URL)
		return
	}

	now := time.Now()
	for _, tag := range device.tags {
		switch tag.Type {
		case "currentPower":
			device.record(ctx, tag, energy.Power, now)
		case "totalPower":
			device.record(ctx, tag, energy.Total, now)
		}
	}
}

func (device *NousA1T) RelayOn(ctx context.Context) (bool, error) {
	body, _, err := device.deps.HTTP.Get(ctx, "http://"+device.URL+"/cm?cmnd=Power")
	if err != nil {
		return false, err
	}
	slog.Info("nous a1t: relay status", "device", device.Name, "host", device.URL, "body", string(body))

	var power tasmotaPower
	if err := json.Unmarshal(body, &power); err != nil || power.POWER == "" {
		return false, errRelayStatusUnknown
	}
	return strings.EqualFold(power.POWER, "ON"), nil
}

func (device *NousA1T) TurnOn(ctx context.Context) {
	device.switchRelay(ctx, "On")
}

func (device *NousA1T) TurnOff(ctx context.Context) {
	device.switchRelay(ctx, "Off")
}

func (device *NousA1T) switchRelay(ctx context.Context, state string) {
	body, _, err := device.deps.HTTP.Get(ctx, "http://"+device.URL+"/cm?cmnd=Power%20"+state)
	if err != nil {
		slog.Error("nous a1t: switching the relay failed", "device", device.Name, "host", device.URL, "state", state, "err", err)
		return
	}
	slog.Info("nous a1t: relay switched", "device", device.Name, "host", device.URL, "state", state, "body", string(body))
}
