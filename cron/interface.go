package cron

import (
	"github.com/mewa/djinn/schedule"
	"time"
)

type EntryID string

type Entry struct {
	// has to be unique for cron instance
	ID EntryID

	Schedule schedule.Schedule

	Prev time.Time
	Next time.Time
}

type Cron interface {
	PutEntry(entry Entry)
	DeleteEntry(id EntryID)

	Entry(id EntryID) Entry

	Start()
	Stop()
}
