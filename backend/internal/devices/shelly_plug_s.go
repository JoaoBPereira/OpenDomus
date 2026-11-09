package devices

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"od/internal/models"
)

type shellyPlugStatus struct {
	Meters []shellyMeter `json:"meters"`
}

type ShellyPlugS struct {
	models.PowerDeviceFields
	poller
}

func NewShellyPlugS(deps Deps, fields models.DeviceFields) (*ShellyPlugS, error) {
	device := &ShellyPlugS{PowerDeviceFields: models.PowerDeviceFields{DeviceFields: fields}}
	device.Type = "Shelly Plug S"
	device.Enabled = true
	device.init(deps, fields.ID)
	device.SetPowerTags(device.tags)
	if err := device.start(fields.Periodicity, device.poll); err != nil {
		return nil, err
	}
	return device, nil
}

func (device *ShellyPlugS) Stop() {
	device.stop()
}

func (device *ShellyPlugS) poll(ctx context.Context) {
	body, _, err := device.deps.HTTP.Get(ctx, "http://"+device.URL+"/status")
	if err != nil {
		slog.Error("shelly plug s: status request failed", "device", device.Name, "host", device.URL, "err", err)
		return
	}
	slog.Info("shelly plug s: status", "device", device.Name, "host", device.URL, "body", string(body))

	var status shellyPlugStatus
	if err := json.Unmarshal(body, &status); err != nil {
		slog.Error("shelly plug s: invalid status answer", "device", device.Name, "host", device.URL, "err", err)
		return
	}

	now := time.Now()
	for _, tag := range device.tags {
		if tag.Type != "currentPower" && tag.Type != "totalPower" {
			continue
		}
		config, err := parseShellyTagConfig(tag)
		if err != nil {
			slog.Error("shelly plug s: invalid tag config", "tag", tag.ID, "err", err)
			continue
		}
		if config.Emeter == nil {
			continue
		}
		value, found := meterValue(status.Meters, config.Emeter)
		if !found || (value == 0 && config.Emeter.Type == "total") {
			continue // a total of 0 is a Shelly glitch, not a reset
		}
		device.record(ctx, tag, value, now)
	}
}

var errRelayStatusUnknown = errors.New("relay status unknown")

func (device *ShellyPlugS) RelayOn(ctx context.Context) (bool, error) {
	body, _, err := device.deps.HTTP.Get(ctx, "http://"+device.URL+"/relay/0")
	if err != nil {
		return false, err
	}
	slog.Info("shelly plug s: relay status", "device", device.Name, "host", device.URL, "body", string(body))

	var relay relayAnswer
	if err := json.Unmarshal(body, &relay); err != nil || relay.IsOn == nil {
		return false, errRelayStatusUnknown
	}
	// Kept from the TypeScript version (`res.data?.ison ? res.data.ison : null`): a relay that is
	// off is reported as unknown, so TasksManager never turns this plug back on.
	if !*relay.IsOn {
		return false, errRelayStatusUnknown
	}
	return true, nil
}

func (device *ShellyPlugS) TurnOn(ctx context.Context) {
	device.switchRelay(ctx, "on")
}

func (device *ShellyPlugS) TurnOff(ctx context.Context) {
	device.switchRelay(ctx, "off")
}

func (device *ShellyPlugS) switchRelay(ctx context.Context, state string) {
	body, _, err := device.deps.HTTP.Get(ctx, "http://"+device.URL+"/relay/0?turn="+state)
	if err != nil {
		slog.Error("shelly plug s: switching the relay failed", "device", device.Name, "host", device.URL, "state", state, "err", err)
		return
	}
	slog.Info("shelly plug s: relay switched", "device", device.Name, "host", device.URL, "state", state, "body", string(body))
}
