package schedule

import (
	"encoding/json"
	"time"
)

const timeFormat = time.RFC3339

type OnceSchedule struct {
	Time    time.Time `json:"time"`
	Ran     bool      `json:"ran"`
	Running bool      `json:"-"`
}

func Once(t time.Time) *OnceSchedule {
	return &OnceSchedule{t, false, false}
}

func (im *OnceSchedule) Next(t time.Time) time.Time {
	if !im.Ran && !im.Running {
		return im.Time
	}
	return Abort
}

func (im *OnceSchedule) BeforeJob() {
	im.Running = true
}

func (im *OnceSchedule) AfterJob() {
	im.Ran = true
}

func (im *OnceSchedule) Serialize() string {
	d, _ := json.Marshal(im)
	return string(d)
}

func (im *OnceSchedule) Deserialize(spec string) error {
	var sched OnceSchedule
	err := json.Unmarshal([]byte(spec), &sched)

	if err != nil {
		return err
	}

	im.Time = sched.Time
	im.Ran = im.Ran || sched.Ran

	return nil
}
