package devices

import (
	"encoding/json"
	"strconv"

	"od/internal/models"
)

// Tag config shared by the Shelly devices: {"Emeter": {"index": 0, "type": "power"}} for the power
// tags and {"temperature": {"index": 0}} for the external temperature sensors of the Shelly 1 PM.
// The index is stored as a number or as a string, which json.Number accepts both.
type shellyTagConfig struct {
	Emeter      *shellyEmeterConfig `json:"Emeter"`
	Temperature *struct {
		Index json.Number `json:"index"`
	} `json:"temperature"`
}

type shellyEmeterConfig struct {
	Index json.Number `json:"index"`
	Type  string      `json:"type"` // power | total
}

// shellyMeter is one entry of the meters / emeters arrays of the Gen1 status answer
type shellyMeter struct {
	Power float64 `json:"power"`
	Total float64 `json:"total"`
}

func parseShellyTagConfig(tag *models.Tag) (shellyTagConfig, error) {
	var config shellyTagConfig
	err := json.Unmarshal([]byte(tag.Config), &config)
	return config, err
}

// meterValue reads the power or total of the meter the config points at
func meterValue(meters []shellyMeter, config *shellyEmeterConfig) (float64, bool) {
	index, err := strconv.Atoi(config.Index.String())
	if err != nil || index < 0 || index >= len(meters) {
		return 0, false
	}
	switch config.Type {
	case "power":
		return meters[index].Power, true
	case "total":
		return meters[index].Total, true
	}
	return 0, false
}

// relayAnswer is the answer of the Gen1 /relay/0 endpoint
type relayAnswer struct {
	IsOn *bool `json:"ison"`
}
