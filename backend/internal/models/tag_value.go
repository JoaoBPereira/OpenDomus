package models

import "time"

// TagValue is implemented by TagValueFloat, TagValueBoolean and TagValueLocation.
// (GetTime keeps its prefix because the structs have a Time field.)
type TagValue interface {
	GetTime() time.Time
}
