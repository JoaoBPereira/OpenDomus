// Package parse converts the loosely typed values the application receives (JSON of unknown shape,
// URL parameters, device answers) into Go values.
package parse

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Float accepts numbers, numeric strings, booleans (1/0) and json.Number
func Float(value any) (float64, error) {
	switch typed := value.(type) {
	case float64:
		return typed, nil
	case float32:
		return float64(typed), nil
	case int:
		return float64(typed), nil
	case int32:
		return float64(typed), nil
	case int64:
		return float64(typed), nil
	case bool:
		if typed {
			return 1, nil
		}
		return 0, nil
	case json.Number:
		return typed.Float64()
	case string:
		return strconv.ParseFloat(strings.TrimSpace(typed), 64)
	default:
		return 0, fmt.Errorf("not a number: %v", value)
	}
}

// Int is Float truncated to an int
func Int(value any) (int, error) {
	number, err := Float(value)
	return int(number), err
}

// String returns the value as text ("" for nil)
func String(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	default:
		return fmt.Sprint(value)
	}
}

// Bool reads the ways a boolean tag value arrives: nil, false, 0, "", "0", "false", "False" and "off" are false
func Bool(value any) bool {
	switch typed := value.(type) {
	case nil:
		return false
	case bool:
		return typed
	case string:
		return typed != "" && typed != "0" && typed != "false" && typed != "False" && typed != "off"
	default:
		number, err := Float(value)
		return err == nil && number != 0
	}
}

// Date parses the timestamp formats the project receives (ISO 8601 and "YYYY-MM-DD HH:MM:SS").
// Dates without a time zone are taken as local time.
func Date(text string) (time.Time, error) {
	layouts := []string{
		time.RFC3339Nano,
		"2006-01-02T15:04:05",
		"2006-01-02T15:04",
		"2006-01-02 15:04:05",
		"2006-01-02",
		time.RFC1123Z,
		time.RFC1123,
	}
	trimmed := strings.TrimSpace(text)
	for _, layout := range layouts {
		if parsed, err := time.ParseInLocation(layout, trimmed, time.Local); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid date: %q", text)
}
