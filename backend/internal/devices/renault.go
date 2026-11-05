package devices

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"time"

	"od/internal/models"
	"od/internal/parse"
	"od/internal/renault"
)

// Tag config: {"endpoint": "battery-status", "field": "batteryLevel"}
// Location tags (valueType tagvaluelocation): {"endpoint": "location"}
type renaultTagConfig struct {
	Endpoint string `json:"endpoint"`
	Field    string `json:"field"`
}

type renaultTag struct {
	tag    *models.Tag
	config renaultTagConfig
}

// Renault is a car read through the MyRenault API. The device url is the VIN (empty: the first
// car of the account). Keep the periodicity at 10-15 minutes or more, Renault rate limits the API.
type Renault struct {
	models.DeviceFields
	poller
	client *renault.Client
	vin    string
}

func NewRenault(deps Deps, fields models.DeviceFields, client *renault.Client) (*Renault, error) {
	device := &Renault{DeviceFields: fields, client: client, vin: strings.TrimSpace(fields.URL)}
	device.Type = "Renault"
	device.Enabled = true
	device.init(deps, fields.ID)
	if err := device.start(fields.Periodicity, device.poll); err != nil {
		return nil, err
	}
	return device, nil
}

func (device *Renault) Stop() {
	device.stop()
}

func (device *Renault) getVIN(ctx context.Context) (string, error) {
	if device.vin != "" {
		return device.vin, nil
	}
	vehicles, err := device.client.Vehicles(ctx)
	if err != nil {
		return "", err
	}
	if len(vehicles) == 0 {
		return "", errors.New("no vehicles found in the Renault account")
	}
	slog.Info("renault: vehicles in the account", "device", device.Name, "vehicles", vehicles)
	device.vin = vehicles[0].VIN
	return device.vin, nil
}

func (device *Renault) poll(ctx context.Context) {
	// Group the tags by endpoint, so each endpoint is called once (in first-seen order)
	endpoints := []string{}
	tagsByEndpoint := map[string][]renaultTag{}
	for _, tag := range device.tags {
		var config renaultTagConfig
		if err := json.Unmarshal([]byte(tag.Config), &config); err != nil {
			slog.Error("renault: invalid tag config", "tag", tag.ID, "err", err)
			continue
		}
		if _, known := renault.EndpointVersions[config.Endpoint]; !known {
			slog.Error("renault: unknown endpoint in tag config", "tag", tag.ID, "endpoint", config.Endpoint)
			continue
		}
		if _, exists := tagsByEndpoint[config.Endpoint]; !exists {
			endpoints = append(endpoints, config.Endpoint)
		}
		tagsByEndpoint[config.Endpoint] = append(tagsByEndpoint[config.Endpoint], renaultTag{tag: tag, config: config})
	}
	if len(endpoints) == 0 {
		return
	}

	vin, err := device.getVIN(ctx)
	if err != nil {
		slog.Error("renault: no VIN", "device", device.Name, "err", err)
		return
	}

	// Sequential on purpose, to go easy on the Renault API
	for _, endpoint := range endpoints {
		data, err := device.client.CarData(ctx, vin, endpoint)
		if err != nil {
			slog.Error("renault: request failed", "device", device.Name, "endpoint", endpoint, "err", err)
			continue
		}
		slog.Info("renault: answer", "device", device.Name, "endpoint", endpoint, "data", data)
		for _, item := range tagsByEndpoint[endpoint] {
			device.saveTag(ctx, item, data)
		}
	}
}

func (device *Renault) saveTag(ctx context.Context, item renaultTag, data map[string]any) {
	var value any
	at := time.Now()

	if item.config.Endpoint == "location" {
		latitude, latitudeErr := parse.Float(data["gpsLatitude"])
		longitude, longitudeErr := parse.Float(data["gpsLongitude"])
		if latitudeErr != nil || longitudeErr != nil {
			return
		}
		if parsed, err := parse.Date(parse.String(data["lastUpdateTime"])); err == nil {
			at = parsed
		}
		value = models.NewTagValueLocation(item.tag.ID, latitude, longitude, 0, 0, 0, at)
	} else {
		value = data[item.config.Field]
		if value == nil {
			slog.Warn("renault: field not in the answer", "tag", item.tag.ID, "endpoint", item.config.Endpoint, "field", item.config.Field)
			return
		}
		if parsed, err := parse.Date(parse.String(data["timestamp"])); err == nil {
			at = parsed
		}
	}
	device.store(ctx, item.tag, value, at)
}
