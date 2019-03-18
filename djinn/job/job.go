package job

import (
	"fmt"
	"github.com/mewa/djinn/schedule"
	"time"
)

type ID string

type state uint8

const (
	Initial state = iota
	Starting
	Started
	Error
)

type State struct {
	State state
	Time  int64
}

type Handler struct {
	Remove func(job *Job)
	Run    func(job *Job)
}

type Job struct {
	ID ID `json:"id"`

	State State `json:"state"`

	Descriptor schedule.JSONSchedule `json:"schedule"`

	NextTime time.Time `json:"next"`
	PrevTime time.Time `json:"prev"`

	schedule schedule.Schedule `json:"-"`

	Handler Handler `json:"-"`
}

func (job *Job) Schedule() schedule.Schedule {
	if job.schedule == nil {
		job.schedule, _ = job.Descriptor.Schedule()
	}
	return job.schedule
}

func (job *Job) Next(t time.Time) time.Time {
	next := job.Schedule().Next(t)

	if next.IsZero() {
		if job.State.State == Started {
			go job.Handler.Remove(job)
		}
		return next
	}
	job.PrevTime = job.NextTime
	job.NextTime = next
	return next
}

func (job *Job) Run() {
	job.Handler.Run(job)
}

func (s State) String() string {
	return fmt.Sprintf("{%s@%s}", time.Unix(s.Time, 0), s.State)
}

func (s state) String() string {
	switch s {
	case Initial:
		return "initial"
	case Starting:
		return "starting"
	case Started:
		return "started"
	}
	return "unknown"
}

func (job *Job) String() string {
	return fmt.Sprintf("{id=%s, state=%s, descriptor=%v, next=%s, prev=%s}", job.ID, job.State, job.Descriptor, job.NextTime, job.PrevTime)
}
