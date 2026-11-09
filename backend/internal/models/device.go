package models

import "context"

// Device is implemented by every device type. The common fields live in DeviceFields, which
// every device embeds; Stop ends its polling.
type Device interface {
	Fields() *DeviceFields
	Stop()
}

// DeviceFields are the columns of the device table
type DeviceFields struct {
	ID          int    `json:"id"`
	Type        string `json:"type"`
	URL         string `json:"url"`
	Name        string `json:"name"`
	Periodicity string `json:"periodicity"`
	Enabled     bool   `json:"enabled"`
	Config      string `json:"config"`
}

func (fields *DeviceFields) Fields() *DeviceFields {
	return fields
}

// PowerDevice is a device with a relay and power meters
type PowerDevice interface {
	Device
	CurrentPowerTag() *Tag
	CurrentPower() TagValue
	TotalPower() TagValue
	// RelayOn reports whether relay 0 is on; the error says when the status is unknown
	RelayOn(ctx context.Context) (bool, error)
	TurnOn(ctx context.Context)
	TurnOff(ctx context.Context)
}

// PowerDeviceFields is embedded by the power devices: the tags of the current and total power meters
type PowerDeviceFields struct {
	DeviceFields
	CurrentTag *Tag `json:"-"`
	TotalTag   *Tag `json:"-"`
}

// SetPowerTags picks the currentPower and totalPower tags out of the tags of the device
func (fields *PowerDeviceFields) SetPowerTags(tags []*Tag) {
	for _, tag := range tags {
		switch tag.Type {
		case "currentPower":
			fields.CurrentTag = tag
		case "totalPower":
			fields.TotalTag = tag
		}
	}
}

func (fields *PowerDeviceFields) CurrentPowerTag() *Tag {
	return fields.CurrentTag
}

func (fields *PowerDeviceFields) CurrentPower() TagValue {
	if fields.CurrentTag == nil {
		return nil
	}
	return fields.CurrentTag.LastValue()
}

func (fields *PowerDeviceFields) TotalPower() TagValue {
	if fields.TotalTag == nil {
		return nil
	}
	return fields.TotalTag.LastValue()
}
