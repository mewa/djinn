package cron

import (
	"github.com/mewa/djinn/schedule"
	"time"
)

type Job interface {
	Run()
}

type EntryID string

type Entry struct {
	// has to be unique for cron instance
	ID EntryID

	Schedule schedule.Schedule
	Job      Job

	Prev time.Time
	Next time.Time
}

type Cron interface {
	PutEntry(entry Entry)
	DeleteEntry(id EntryID)

	Entry(id EntryID) *Entry

	Start()
	Stop()
}
