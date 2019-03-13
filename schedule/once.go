package schedule

import (
	"time"
)

const timeFormat = time.RFC3339

type OnceSchedule struct {
	t   time.Time
	ran bool
}

func Once(t time.Time) *OnceSchedule {
	return &OnceSchedule{t, false}
}

func (im *OnceSchedule) Next(t time.Time) time.Time {
	if !im.ran {
		im.ran = true
		return im.t
	}
	return time.Time{}
}

func (im *OnceSchedule) Serialize() string {
	return im.t.Format(timeFormat)
}

func (im *OnceSchedule) Deserialize(spec string) error {
	t, err := time.Parse(timeFormat, spec)

	if err != nil {
		return err
	}

	im.t = t
	return nil
}
