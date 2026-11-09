package devices

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"od/internal/models"
)

// Shelly.GetStatus answers one object per component: {"switch:0": {"apower": 12.3, "aenergy": {"total": 456.7}}, "sys": {...}, ...}
type shellyGen3Status map[string]json.RawMessage

type shellyGen3Switch struct {
	APower  float64 `json:"apower"`
	AEnergy struct {
		Total float64 `json:"total"`
	} `json:"aenergy"`
}

type ShellyPlusGen3PlugS struct {
	models.PowerDeviceFields
	poller
}

func NewShellyPlusGen3PlugS(deps Deps, fields models.DeviceFields) (*ShellyPlusGen3PlugS, error) {
	device := &ShellyPlusGen3PlugS{PowerDeviceFields: models.PowerDeviceFields{DeviceFields: fields}}
	device.Type = "Shelly Gen3 Plug S" // also sold as Shelly Plus Plug S
	device.Enabled = fields.Enabled 
	device.init(deps, fields.ID)
	device.SetPowerTags(device.tags)
	if err := device.start(fields.Periodicity, device.poll); err != nil {
		return nil, err
	}
	return device, nil
}

func (device *ShellyPlusGen3PlugS) Stop() {
	device.stop()
}

func (device *ShellyPlusGen3PlugS) poll(ctx context.Context) {
	body, _, err := device.deps.HTTP.Get(ctx, "http://"+device.URL+"/rpc/Shelly.GetStatus")
	if err != nil {
		slog.Error("shelly gen3 plug: status request failed", "device", device.Name, "host", device.URL, "err", err)
		return
	}
	slog.Info("shelly gen3 plug: status", "device", device.Name, "host", device.URL, "body", string(body))

	var status shellyGen3Status
	if err := json.Unmarshal(body, &status); err != nil {
		slog.Error("shelly gen3 plug: invalid status answer", "device", device.Name, "host", device.URL, "err", err)
		return
	}

	now := time.Now()
	for _, tag := range device.tags {
		if tag.Type != "currentPower" && tag.Type != "totalPower" {
			continue
		}
		config, err := parseShellyTagConfig(tag)
		if err != nil {
			slog.Error("shelly gen3 plug: invalid tag config", "tag", tag.ID, "err", err)
			continue
		}
		if config.Emeter == nil {
			continue
		}
		value, found := switchValue(status, config.Emeter)
		if !found || (value == 0 && config.Emeter.Type == "total") {
			continue // a total of 0 is a Shelly glitch, not a reset
		}
		device.record(ctx, tag, value, now)
	}
}

// switchValue reads the power or total of the "switch:<index>" component the config points at
func switchValue(status shellyGen3Status, config *shellyEmeterConfig) (float64, bool) {
	raw, exists := status["switch:"+config.Index.String()]
	if !exists {
		return 0, false
	}
	var switchStatus shellyGen3Switch
	if err := json.Unmarshal(raw, &switchStatus); err != nil {
		return 0, false
	}
	switch config.Type {
	case "power":
		return switchStatus.APower, true
	case "total":
		return switchStatus.AEnergy.Total, true
	}
	return 0, false
}

func (device *ShellyPlusGen3PlugS) RelayOn(ctx context.Context) (bool, error) {
	body, _, err := device.deps.HTTP.Get(ctx, "http://"+device.URL+"/relay/0")
	if err != nil {
		return false, err
	}
	slog.Info("shelly gen3 plug: relay status", "device", device.Name, "host", device.URL, "body", string(body))

	var relay relayAnswer
	if err := json.Unmarshal(body, &relay); err != nil || relay.IsOn == nil {
		return false, errRelayStatusUnknown
	}
	return *relay.IsOn, nil
}

func (device *ShellyPlusGen3PlugS) TurnOn(ctx context.Context) {
	device.switchRelay(ctx, "on")
}

func (device *ShellyPlusGen3PlugS) TurnOff(ctx context.Context) {
	device.switchRelay(ctx, "off")
}

func (device *ShellyPlusGen3PlugS) switchRelay(ctx context.Context, state string) {
	body, _, err := device.deps.HTTP.Get(ctx, "http://"+device.URL+"/relay/0?turn="+state)
	if err != nil {
		slog.Error("shelly gen3 plug: switching the relay failed", "device", device.Name, "host", device.URL, "state", state, "err", err)
		return
	}
	slog.Info("shelly gen3 plug: relay switched", "device", device.Name, "host", device.URL, "state", state, "body", string(body))
}
