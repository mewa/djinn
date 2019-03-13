package schedule

import (
	"github.com/mewa/cron"
)

var parser = cron.NewParser(cron.SecondOptional |
	cron.Minute |
	cron.Hour |
	cron.Dom |
	cron.Month |
	cron.Dow |
	cron.Descriptor)

type SpecSchedule struct {
	Spec string `json:"spec"`
	Schedule
}

func (ss *SpecSchedule) Serialize() string {
	return ss.Spec
}

func (ss *SpecSchedule) Deserialize(spec string) error {
	ss.Spec = spec

	sched, err := parser.Parse(spec)

	if err != nil {
		return err
	}

	ss.Schedule = sched
	return nil
}
