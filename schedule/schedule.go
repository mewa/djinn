package schedule

import (
	"errors"
	"time"
)

type Schedule interface {
	Next(time.Time) time.Time
}

type SchedType uint16

type Serializable interface {
	Serialize() string
	Deserialize(string) error
}

type SerializableSchedule interface {
	Schedule
	Serializable
}

const (
	once SchedType = iota
	spec
)

type JSONSchedule struct {
	ScheduleType SchedType `json:"type"`
	ScheduleData string    `json:"schedule"`
}

func (js JSONSchedule) Schedule() (SerializableSchedule, error) {
	switch js.ScheduleType {
	case once:
		sched := new(OnceSchedule)
		return sched, sched.Deserialize(js.ScheduleData)
	case spec:
		sched := new(SpecSchedule)
		return sched, sched.Deserialize(js.ScheduleData)
	}
	return nil, ErrUnknownScheduleType
}

var (
	ErrUnknownScheduleType = errors.New("unknown schedule type")
)
