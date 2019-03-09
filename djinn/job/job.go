package job

import (
	"time"
	"encoding/json"
	"github.com/mewa/djinn/cron"
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

	Schedule cron.Schedule `json:"-"`
	RemoveHandler func(job *Job) `json:"-"`

	NextTime time.Time `json:"next"`
	PrevTime time.Time `json:"prev"`
}

func (job *Job) Next(t time.Time) time.Time {
	next := job.Schedule.Next(t)
	if next.IsZero() && job.State == Done {
		go job.RemoveHandler(job)
	}
	job.PrevTime = job.NextTime
	job.NextTime = next
	return next
}

func Marshal(job *Job) ([]byte, error) {
	return json.Marshal(job)
}

func Unmarshal(data []byte, job *Job) error {
	return json.Unmarshal(data, job)
}
