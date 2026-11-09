package models

import "time"

// TagValueBoolean is a row of the tagvalueboolean table (see TagValueFloat for the tagid column)
type TagValueBoolean struct {
	TagID int       `json:"tagId" db:"tagid"`
	Time  time.Time `json:"time" db:"time"`
	Value bool      `json:"value" db:"value"`
}

func NewTagValueBoolean(tagID int, at time.Time, value bool) *TagValueBoolean {
	return &TagValueBoolean{TagID: tagID, Time: at, Value: value}
}

func (tagValue *TagValueBoolean) GetTime() time.Time {
	return tagValue.Time
}
