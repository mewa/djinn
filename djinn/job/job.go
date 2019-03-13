package job

import (
	"github.com/mewa/djinn/schedule"
	"time"
)

type ID string

type State uint8

const (
	Initial State = iota
	Started
	Done
)

type Job struct {
	ID ID `json:"id"`

	State State `json:"state"`

	Descriptor schedule.JSONSchedule `json:"schedule"`

	NextTime time.Time `json:"next"`
	PrevTime time.Time `json:"prev"`

	schedule      schedule.Schedule `json:"-"`
	RemoveHandler func(job *Job)    `json:"-"`
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
		if job.State == Done {
			go job.RemoveHandler(job)
		}
		return next
	}
	job.PrevTime = job.NextTime
	job.NextTime = next
	return next
}
