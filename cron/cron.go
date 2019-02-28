package cron

import (
	c "github.com/mewa/cron"
	"sync"
	"time"
)

type Schedule interface {
	c.Schedule
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

type cron struct {
	entries *Entries

	running bool

	timer *time.Timer

	addedC   chan Entry
	removedC chan EntryID

	stopC chan struct{}

	mu *sync.Mutex
}

func New() Cron {
	return &cron{
		entries: NewEntries(),

		running: false,

		addedC:   make(chan Entry),
		removedC: make(chan EntryID),
		stopC:    make(chan struct{}),

		mu: new(sync.Mutex),
	}
}

func (c *cron) run() {
	var entry *Entry

	for {
		entry = c.entries.Peek()

		if entry != nil {
			sleep := entry.NextTime.Sub(time.Now())

			if c.timer != nil {
				c.timer.Stop()
			}
			c.timer = time.NewTimer(sleep)

		Loop:
			for {
				select {
				case <-c.stopC:
					return
				case <-c.addedC:
					break Loop
				case <-c.removedC:
					break Loop
				case t := <-c.timer.C:
					entry.PrevTime = t
					entry.NextTime = entry.Schedule.Next(t)

					c.entries.Update(*entry)
					// uncomment when Test_SingleExecution_NextTimeIsZero is implemented
					// if entry.NextTime.IsZero() {
					// 	c.RemoveEntry(entry.ID)
					// }

					break Loop
				}
			}
		} else {
			select {
			case <-c.addedC:
				break
			case <-c.stopC:
				return
			}
		}
	}
}

func (c *cron) AddEntry(entry Entry) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if entry.PrevTime.IsZero() {
		entry.NextTime = entry.Schedule.Next(time.Now())
	} else {
		entry.NextTime = entry.Schedule.Next(entry.PrevTime)
	}

	c.entries.Add(entry)

	if c.running {
		c.addedC <- entry
	}
}

func (c *cron) RemoveEntry(id EntryID) {
	c.entries.Remove(id)
}

func (c *cron) Start() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.running {
		c.running = true
		go c.run()
	}
}

func (c *cron) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.running {
		c.stopC <- struct{}{}
		c.running = false
	}
}
