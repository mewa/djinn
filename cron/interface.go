package cron

import (
	"time"
)

type Schedule interface {
	Next(time.Time) time.Time
}

type EntryID string

type Entry struct {
	// has to be unique for cron instance
	ID EntryID

	Schedule Schedule

	Prev time.Time
	Next time.Time
}

type Cron interface {
	AddEntry(entry Entry)
	UpsertEntry(entry Entry)

	UpdateEntry(entry Entry)
	Entry(id EntryID) Entry
	RemoveEntry(id EntryID)

	Start()
	Stop()
}
