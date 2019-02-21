package djinn

import (
	"github.com/mewa/cron"
	"github.com/mewa/djinn/schedule"
	"go.uber.org/zap"
)

func (d *Djinn) Schedule(sched cron.Schedule) error {
	var schedType string

	switch sched.(type) {
	case *schedule.Once:
		schedType = "once"
	case *cron.SpecSchedule:
		schedType = "cron"
	default:
		return ErrUnknownScheduleType
	}

	d.log.Info("new schedule", zap.String("type", schedType))

	return nil
}
