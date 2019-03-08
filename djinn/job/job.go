package job

import (
	"time"
	"encoding/json"
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

	Next time.Time `json:"next"`
	Prev time.Time `json:"prev"`
}

func Marshal(job *Job) ([]byte, error) {
	return json.Marshal(job)
}

func Unmarshal(data []byte, job *Job) error {
	return json.Unmarshal(data, job)
}
