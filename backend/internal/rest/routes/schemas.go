package routes

import (
	"fmt"
	"slices"
	"strings"

	"od/internal/models"
)

// Request body validation. Each Validate() returns the first problem found as the message for
// the 400 response, or "" when the body is valid. Optional fields are pointers, so a missing
// field can be told apart from an empty one.

type LoginSchema struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (schema *LoginSchema) Validate() string {
	if message := checkString(&schema.Username, "username", true, 1, 64); message != "" {
		return message
	}
	return checkString(&schema.Password, "password", true, 1, 128)
}

type DeviceSchema struct {
	Type        *string `json:"type"`
	URL         *string `json:"url"`
	Name        *string `json:"name"`
	Periodicity *string `json:"periodicity"`
	Enabled     *bool   `json:"enabled"`
	Config      *string `json:"config"`
}

type DeviceCreateSchema struct {
	Device *DeviceSchema `json:"device"`
}

func (schema *DeviceCreateSchema) Validate() string {
	if schema.Device == nil {
		return "device: Invalid input: expected object, received undefined"
	}
	return schema.Device.validate(true)
}

type DeviceUpdateSchema struct {
	Device *DeviceSchema `json:"device"`
}

func (schema *DeviceUpdateSchema) Validate() string {
	if schema.Device == nil {
		return "device: Invalid input: expected object, received undefined"
	}
	return schema.Device.validate(false)
}

// validate checks the fields; type and name are required when creating a device
func (device *DeviceSchema) validate(isCreate bool) string {
	if message := checkString(device.Type, "type", isCreate, 1, 50); message != "" {
		return message
	}
	if message := checkString(device.URL, "url", false, 0, 255); message != "" {
		return message
	}
	if message := checkString(device.Name, "name", isCreate, 1, 100); message != "" {
		return message
	}
	if message := checkString(device.Periodicity, "periodicity", false, 0, 50); message != "" {
		return message
	}
	return checkString(device.Config, "config", false, 0, 500)
}

// ToDevice converts the validated body into the device columns. Fields that were not sent are empty.
func (device *DeviceSchema) ToDevice() *models.DeviceFields {
	return &models.DeviceFields{
		Type:        deref(device.Type),
		URL:         deref(device.URL),
		Name:        deref(device.Name),
		Periodicity: deref(device.Periodicity),
		Enabled:     deref(device.Enabled),
		Config:      deref(device.Config),
	}
}

type TagSchema struct {
	Name             *string  `json:"name"`
	Type             *string  `json:"type"`
	ValueType        *string  `json:"valueType"`
	ValueStorage     *string  `json:"valueStorage"`
	Config           *string  `json:"config"`
	DeviceID         *int     `json:"deviceId"`
	PeriodicityCount *int     `json:"periodicityCount"`
	ExpiresIn        *int     `json:"expiresIn"`
	Enabled          *bool    `json:"enabled"`
	DisplayIn        []string `json:"displayIn"`
}

type TagCreateSchema struct {
	Tag *TagSchema `json:"tag"`
}

func (schema *TagCreateSchema) Validate() string {
	if schema.Tag == nil {
		return "tag: Invalid input: expected object, received undefined"
	}
	return schema.Tag.validate(true)
}

type TagUpdateSchema struct {
	Tag *TagSchema `json:"tag"`
}

func (schema *TagUpdateSchema) Validate() string {
	if schema.Tag == nil {
		return "tag: Invalid input: expected object, received undefined"
	}
	return schema.Tag.validate(false)
}

// validate checks the fields; name, type, valueType, valueStorage and deviceId are required when creating a tag
func (tag *TagSchema) validate(isCreate bool) string {
	if message := checkString(tag.Name, "name", isCreate, 1, 100); message != "" {
		return message
	}
	if message := checkString(tag.Type, "type", isCreate, 1, 50); message != "" {
		return message
	}
	if message := checkString(tag.ValueType, "valueType", isCreate, 1, 50); message != "" {
		return message
	}
	if message := checkEnum(tag.ValueStorage, "valueStorage", isCreate, []string{"insert", "update"}); message != "" {
		return message
	}
	if message := checkString(tag.Config, "config", false, 0, 500); message != "" {
		return message
	}
	if message := checkInt(tag.DeviceID, "deviceId", isCreate, 1); message != "" {
		return message
	}
	if message := checkInt(tag.PeriodicityCount, "periodicityCount", false, 0); message != "" {
		return message
	}
	return checkInt(tag.ExpiresIn, "expiresIn", false, 0)
}

// ToTag converts the validated body into a Tag. Fields that were not sent are empty / 0.
func (tag *TagSchema) ToTag() *models.Tag {
	displayIn := tag.DisplayIn
	if displayIn == nil {
		displayIn = []string{}
	}
	return &models.Tag{
		Name:             deref(tag.Name),
		Type:             deref(tag.Type),
		ValueType:        deref(tag.ValueType),
		ValueStorage:     deref(tag.ValueStorage),
		Config:           deref(tag.Config),
		DeviceID:         deref(tag.DeviceID),
		PeriodicityCount: deref(tag.PeriodicityCount),
		ExpiresIn:        deref(tag.ExpiresIn),
		Enabled:          deref(tag.Enabled),
		DisplayIn:        displayIn,
	}
}

// deref returns *pointer, or the zero value when the pointer is nil
func deref[T any](pointer *T) T {
	if pointer == nil {
		var zero T
		return zero
	}
	return *pointer
}

// ---- field checks, with messages in the style of zod ----

func checkString(value *string, name string, required bool, minLength int, maxLength int) string {
	if value == nil {
		if required {
			return name + ": Invalid input: expected string, received undefined"
		}
		return ""
	}
	if len(*value) < minLength {
		return fmt.Sprintf("%s: Too small: expected string to have >=%d characters", name, minLength)
	}
	if len(*value) > maxLength {
		return fmt.Sprintf("%s: Too big: expected string to have <=%d characters", name, maxLength)
	}
	return ""
}

func checkInt(value *int, name string, required bool, minimum int) string {
	if value == nil {
		if required {
			return name + ": Invalid input: expected number, received undefined"
		}
		return ""
	}
	if *value < minimum {
		return fmt.Sprintf("%s: Too small: expected number to be >=%d", name, minimum)
	}
	return ""
}

func checkEnum(value *string, name string, required bool, allowed []string) string {
	expected := `"` + strings.Join(allowed, `"|"`) + `"`
	if value == nil {
		if required {
			return fmt.Sprintf("%s: Invalid option: expected one of %s", name, expected)
		}
		return ""
	}
	if !slices.Contains(allowed, *value) {
		return fmt.Sprintf("%s: Invalid option: expected one of %s", name, expected)
	}
	return ""
}
