package models

import "time"

// PowerConsumptionNearTimestamp is one result of TagsManager.PowerConsumptionNearTimestamp
type PowerConsumptionNearTimestamp struct {
	Tag   *Tag      `json:"tag"`
	Value float64   `json:"value"`
	Time  time.Time `json:"time"`
}

// PaginatedResult is the { rows, meta } object returned by the paginated endpoints (/tagValueFloat and /logs).
// Rows holds a slice of row structs ([]TagValueFloat, []logRow).
type PaginatedResult struct {
	Rows any            `json:"rows"`
	Meta PaginationMeta `json:"meta"`
}

type PaginationMeta struct {
	Total  int `json:"total"`
	Offset int `json:"offset"`
	Count  int `json:"count"`
	Pages  int `json:"pages"`
}
