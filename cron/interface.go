package cron

import (
	"time"
)

type Schedule interface {
	Next(time.Time) time.Time
}

type EntryID string

type Entry struct {
	ID EntryID

	Schedule Schedule

	PrevTime time.Time
	NextTime time.Time
}

type Cron interface {
	AddEntry(entry Entry)
	RemoveEntry(id EntryID)

	Start()
	Stop()
}
