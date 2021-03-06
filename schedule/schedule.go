package schedule

import (
	"errors"
	"time"
)

type Schedule interface {
	Next(time.Time) time.Time
}

var (
	Abort time.Time
)

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
	TypeOnce SchedType = iota
	TypeSpec
)

type JSONSchedule struct {
	ScheduleType SchedType `json:"type"`
	ScheduleData string    `json:"schedule"`
}

func (js JSONSchedule) Schedule() (SerializableSchedule, error) {
	switch js.ScheduleType {
	case TypeOnce:
		sched := new(OnceSchedule)
		return sched, sched.Deserialize(js.ScheduleData)
	case TypeSpec:
		sched := new(SpecSchedule)
		return sched, sched.Deserialize(js.ScheduleData)
	}
	return nil, ErrUnknownScheduleType
}

var (
	ErrUnknownScheduleType = errors.New("unknown schedule type")
)
