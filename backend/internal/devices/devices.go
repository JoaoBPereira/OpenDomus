// Package devices holds one type per kind of device of the house. A device polls its hardware on
// the periodicity of its row in the device table (a cron expression) and hands the readings to
// the TagStore.
package devices

import (
	"context"
	"log/slog"
	"strings"
	"sync/atomic"
	"time"

	"od/internal/httpclient"
	"od/internal/models"
	"od/internal/schedule"
)

// TagStore is what the devices need from the TagsManager
type TagStore interface {
	TagsByDeviceID(deviceID int) []*models.Tag
	SaveTagValue(ctx context.Context, tag *models.Tag, value any, at time.Time) error
}

// Deps is what every device is built with
type Deps struct {
	Tags TagStore
	HTTP *httpclient.Client
}

// runTimeout bounds one polling run: the requests to the device and the database writes
const runTimeout = 30 * time.Second

// poller is embedded by the devices that poll their hardware. It runs poll on the cron expression
// of the device, once right away, and counts the runs for Tag.ShouldStore.
type poller struct {
	deps     Deps
	tags     []*models.Tag
	job      *schedule.Job
	runCount atomic.Int64
}

func (p *poller) init(deps Deps, deviceID int) {
	p.deps = deps
	p.tags = deps.Tags.TagsByDeviceID(deviceID)
}

// start polls on the periodicity cron expression, and once right away; an empty periodicity
// means the device is not polled
func (p *poller) start(periodicity string, poll func(ctx context.Context)) error {
	if strings.TrimSpace(periodicity) == "" {
		return nil
	}
	run := func() {
		p.runCount.Add(1)
		ctx, cancel := context.WithTimeout(context.Background(), runTimeout)
		defer cancel()
		poll(ctx)
	}
	job, err := schedule.New(periodicity, run)
	if err != nil {
		return err
	}
	p.job = job
	go run()
	return nil
}

func (p *poller) stop() {
	p.job.Cancel()
	p.job = nil
}

// record keeps value as the last value of the tag and stores it when this run is one that stores
func (p *poller) record(ctx context.Context, tag *models.Tag, value float64, at time.Time) {
	tag.SetLastValue(models.NewTagValueFloat(tag.ID, at, value))
	p.store(ctx, tag, value, at)
}

// store writes the value when this run is one that stores (see Tag.ShouldStore)
func (p *poller) store(ctx context.Context, tag *models.Tag, value any, at time.Time) {
	if !tag.ShouldStore(int(p.runCount.Load())) {
		return
	}
	if err := p.deps.Tags.SaveTagValue(ctx, tag, value, at); err != nil {
		slog.Error("saving tag value failed", "tag", tag.Name, "err", err)
	}
}
