package devices

import (
	"context"
	"encoding/json"
	"log/slog"
	"slices"
	"strings"
	"time"

	"od/internal/models"
	"od/internal/parse"
)

// Tag config: {"operationId": 1002, "tagName": "temperatura_consigna"}
type ecoForestTagConfig struct {
	OperationID json.Number `json:"operationId"` // number or string in the database
	TagName     string      `json:"tagName"`
}

type ecoForestTag struct {
	tag    *models.Tag
	config ecoForestTagConfig
}

// EcoForest is a pellet stove read through its local web interface
type EcoForest struct {
	models.DeviceFields
	poller
	auth string // value of the Authorization: Basic header
}

func NewEcoForest(deps Deps, fields models.DeviceFields, auth string) (*EcoForest, error) {
	device := &EcoForest{DeviceFields: fields, auth: auth}
	device.Type = "EcoForest"
	device.Enabled = true
	device.init(deps, fields.ID)
	if err := device.start(fields.Periodicity, device.poll); err != nil {
		return nil, err
	}
	return device, nil
}

func (device *EcoForest) Stop() {
	device.stop()
}

func (device *EcoForest) poll(ctx context.Context) {
	tags := []ecoForestTag{}
	for _, tag := range device.tags {
		var config ecoForestTagConfig
		if err := json.Unmarshal([]byte(tag.Config), &config); err != nil {
			slog.Error("ecoforest: invalid tag config", "tag", tag.ID, "err", err)
			continue
		}
		tags = append(tags, ecoForestTag{tag: tag, config: config})
	}

	// Each operation id is requested once, even when several tags share it
	operations := []string{}
	for _, item := range tags {
		if operation := item.config.OperationID.String(); !slices.Contains(operations, operation) {
			operations = append(operations, operation)
		}
	}

	headers := map[string]string{
		"Accept":        "*/*",
		"content-type":  "application/x-www-form-urlencoded; charset=UTF-8",
		"Authorization": "Basic " + device.auth,
	}
	for _, operation := range operations {
		body, _, err := device.deps.HTTP.Post(ctx, "http://"+device.URL+"/recepcion_datos_4.cgi", "idOperacion="+operation, headers)
		if err != nil {
			slog.Error("ecoforest: request failed", "device", device.Name, "host", device.URL, "operation", operation, "err", err)
			continue
		}
		data := string(body)
		slog.Info("ecoforest: answer", "device", device.Name, "host", device.URL, "operation", operation, "body", data)
		for _, item := range tags {
			if item.config.OperationID.String() == operation {
				device.readTag(ctx, data, item)
			}
		}
	}
}

// readTag finds the "name=value" line of the tag in the answer of its operation
func (device *EcoForest) readTag(ctx context.Context, data string, item ecoForestTag) {
	for _, line := range strings.Split(data, "\n") {
		if !strings.Contains(line, item.config.TagName) {
			continue
		}
		_, text, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		value, err := parse.Float(text)
		if err != nil {
			slog.Error("ecoforest: value is not a number", "tag", item.tag.Name, "line", line)
			continue
		}
		device.record(ctx, item.tag, value, time.Now())
	}
}
