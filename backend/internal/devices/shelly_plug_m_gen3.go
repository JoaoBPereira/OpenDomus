package devices

import "od/internal/models"

// ShellyPlugMGen3 is a ShellyPlusGen3PlugS with another type name
type ShellyPlugMGen3 struct {
	*ShellyPlusGen3PlugS
}

func NewShellyPlugMGen3(deps Deps, fields models.DeviceFields) (*ShellyPlugMGen3, error) {
	inner, err := NewShellyPlusGen3PlugS(deps, fields)
	if err != nil {
		return nil, err
	}
	inner.Type = "Shelly Plug M Gen3"
	return &ShellyPlugMGen3{ShellyPlusGen3PlugS: inner}, nil
}
