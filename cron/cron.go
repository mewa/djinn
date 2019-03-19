package cron

import (
	c "github.com/mewa/cron"
	"sync"
)

type cron struct {
	cron     *c.Cron
	entries  map[EntryID]*Entry
	entryIds map[EntryID]c.EntryID

	mu *sync.RWMutex
}

func New() *cron {
	return &cron{
		cron:     c.New(),
		entries:  map[EntryID]*Entry{},
		entryIds: map[EntryID]c.EntryID{},
		mu:       new(sync.RWMutex),
	}
}

func (c *cron) PutEntry(entry Entry) {
	c.mu.Lock()
	defer c.mu.Unlock()

	id := c.cron.Schedule(entry.Schedule, entry.Job)

	c.entries[entry.ID] = &entry
	c.entryIds[entry.ID] = id
}

func (c *cron) DeleteEntry(id EntryID) {
	c.mu.Lock()
	defer c.mu.Unlock()

	cid, ok := c.entryIds[id]
	if ok {
		c.cron.Remove(cid)
		delete(c.entryIds, id)
		delete(c.entries, id)
	}
}

func (c *cron) Entry(id EntryID) *Entry {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if entry := c.entries[id]; entry != nil {
		return entry
	}
	return nil
}

func (c *cron) Start() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cron.Start()
}

func (c *cron) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cron.Stop()
}
