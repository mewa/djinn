package cron

import (
	c "github.com/mewa/cron"
)

type cron struct {
	cron     *c.Cron
	entries  map[EntryID]*Entry
	entryIds map[EntryID]c.EntryID
}

func New() *cron {
	return &cron{
		cron:     c.New(),
		entries:  map[EntryID]*Entry{},
		entryIds: map[EntryID]c.EntryID{},
	}
}

func (c *cron) PutEntry(entry Entry) {
	id := c.cron.Schedule(entry.Schedule, entry.Job)

	c.entries[entry.ID] = &entry
	c.entryIds[entry.ID] = id
}

func (c *cron) DeleteEntry(id EntryID) {
	cid, ok := c.entryIds[id]
	if ok {
		c.cron.Remove(cid)
		delete(c.entryIds, id)
		delete(c.entries, id)
	}
}

func (c *cron) Entry(id EntryID) *Entry {
	if entry := c.entries[id]; entry != nil {
		return entry
	}
	return nil
}

func (c *cron) Start() {
	c.cron.Start()
}

func (c *cron) Stop() {
	c.cron.Stop()
}
