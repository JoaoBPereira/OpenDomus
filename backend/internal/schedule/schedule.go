// Package schedule runs functions on cron expressions. Expressions have 5 fields
// (minute hour day month weekday) or 6 when the first one is seconds; times are local.
package schedule

import (
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
)

type Job struct {
	cron *cron.Cron
}

var parser = cron.NewParser(cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)

// New starts running the function on the given cron expression
func New(expression string, run func()) (*Job, error) {
	jobCron := cron.New(cron.WithParser(parser))
	if _, err := jobCron.AddFunc(expression, run); err != nil {
		return nil, fmt.Errorf("invalid cron expression %q: %w", expression, err)
	}
	jobCron.Start()
	return &Job{cron: jobCron}, nil
}

// Cancel stops the job. Safe to call on a nil job.
func (job *Job) Cancel() {
	if job == nil {
		return
	}
	job.cron.Stop()
}

// Validate checks a cron expression without scheduling it
func Validate(expression string) error {
	if _, err := parser.Parse(expression); err != nil {
		return fmt.Errorf("invalid cron expression %q: %w", expression, err)
	}
	return nil
}

// nextFunc is a cron schedule computed by a function; a zero time means no next run
type nextFunc func(time.Time) time.Time

func (next nextFunc) Next(after time.Time) time.Time {
	return next(after)
}

// NewFunc starts running the function at the times returned by next (for times that change
// every day, like sunrise)
func NewFunc(next func(after time.Time) time.Time, run func()) *Job {
	jobCron := cron.New()
	jobCron.Schedule(nextFunc(next), cron.FuncJob(run))
	jobCron.Start()
	return &Job{cron: jobCron}
}
