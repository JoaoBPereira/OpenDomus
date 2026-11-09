package managers

import (
	"time"

	"github.com/nathan-osman/go-sunrise"
)

// nextSunEvent returns the first sunrise or sunset (plus offsetMinutes) after the given time,
// or a zero time when the sun does not rise or set in the next days
func nextSunEvent(latitude float64, longitude float64, rise bool, offsetMinutes int, after time.Time) time.Time {
	local := after.Local()
	for day := 0; day <= 2; day++ {
		date := local.AddDate(0, 0, day)
		riseUTC, setUTC := sunrise.SunriseSunset(latitude, longitude, date.Year(), date.Month(), date.Day())
		event := setUTC
		if rise {
			event = riseUTC
		}
		if event.IsZero() {
			continue
		}
		event = event.Add(time.Duration(offsetMinutes) * time.Minute).Truncate(time.Minute).Local()
		if event.After(after) {
			return event
		}
	}
	return time.Time{}
}
