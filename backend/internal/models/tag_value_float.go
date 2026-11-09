package models

import "time"

// TagValueFloat is a row of the tagvaluefloat table. The queries select "tag_id AS tagId",
// which PostgreSQL returns as the column "tagid".
type TagValueFloat struct {
	TagID int       `json:"tagId" db:"tagid"`
	Time  time.Time `json:"time" db:"time"`
	Value float64   `json:"value" db:"value"`
}

func NewTagValueFloat(tagID int, at time.Time, value float64) *TagValueFloat {
	return &TagValueFloat{TagID: tagID, Time: at, Value: value}
}

func (tagValue *TagValueFloat) GetTime() time.Time {
	return tagValue.Time
}

// TagValueFloatTimeSeries is one bucket of TagsManager.TagValueFloatTimeSeries
type TagValueFloatTimeSeries struct {
	Bucket   time.Time `json:"bucket" db:"bucket"`
	AvgValue *float64  `json:"avg_value" db:"avg_value"`
	MaxValue *float64  `json:"max_value" db:"max_value"`
	MinValue *float64  `json:"min_value" db:"min_value"`
}
