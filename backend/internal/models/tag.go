package models

import "sync"

type Tag struct {
	ID               int      `json:"id"`
	Name             string   `json:"name"`
	Type             string   `json:"type"`
	ValueType        string   `json:"valueType"`
	ValueStorage     string   `json:"valueStorage"` // insert, update
	Config           string   `json:"config"`
	DeviceID         int      `json:"deviceId"`
	PeriodicityCount int      `json:"periodicityCount"`
	ExpiresIn        int      `json:"expiresIn"` // days
	Enabled          bool     `json:"enabled"`
	DisplayIn        []string `json:"displayIn"`

	DeviceName        string `json:"deviceName,omitempty"`
	DevicePeriodicity string `json:"devicePeriodicity,omitempty"`

	// LastTagValue is the most recent value seen for this tag. It is exported for the JSON
	// encoder; read it with LastValue() and write it with SetLastValue(), since the devices
	// update it from their own goroutines.
	LastTagValue   TagValue `json:"lastTagValue"`
	lastValueMutex sync.Mutex
}

// ShouldStore says whether the value of the given run is stored: the first run and then every
// PeriodicityCount runs (0 means every run)
func (tag *Tag) ShouldStore(runCount int) bool {
	every := 1
	if tag.PeriodicityCount > 0 {
		every = tag.PeriodicityCount
	}
	return runCount == 1 || runCount%every == 0
}

// SetLastValue keeps the value when it is newer than the current one
func (tag *Tag) SetLastValue(value TagValue) {
	tag.lastValueMutex.Lock()
	defer tag.lastValueMutex.Unlock()

	if tag.LastTagValue == nil || tag.LastTagValue.GetTime().Before(value.GetTime()) {
		tag.LastTagValue = value
	}
}

func (tag *Tag) LastValue() TagValue {
	tag.lastValueMutex.Lock()
	defer tag.lastValueMutex.Unlock()

	return tag.LastTagValue
}
