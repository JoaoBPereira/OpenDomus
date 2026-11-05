package models

import "time"

// TagValueLocation is a row of the tagvaluelocation table (see TagValueFloat for the tagid column).
// Accuracy, altitude and speed are nullable columns.
type TagValueLocation struct {
	ID        int       `json:"id" db:"id"`
	TagID     int       `json:"tagId" db:"tagid"`
	Latitude  float64   `json:"latitude" db:"latitude"`
	Longitude float64   `json:"longitude" db:"longitude"`
	Accuracy  *float64  `json:"accuracy" db:"accuracy"`
	Altitude  *float64  `json:"altitude" db:"altitude"`
	Speed     *float64  `json:"speed" db:"speed"`
	Time      time.Time `json:"time" db:"time"`
}

func NewTagValueLocation(tagID int, latitude float64, longitude float64, accuracy float64, altitude float64, speed float64, at time.Time) *TagValueLocation {
	if at.IsZero() {
		at = time.Now()
	}
	return &TagValueLocation{
		TagID:     tagID,
		Latitude:  latitude,
		Longitude: longitude,
		Accuracy:  &accuracy,
		Altitude:  &altitude,
		Speed:     &speed,
		Time:      at,
	}
}

func (tagValue *TagValueLocation) GetTime() time.Time {
	return tagValue.Time
}
