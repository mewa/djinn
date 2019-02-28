package cron

import (
	"container/heap"
	"sync"
)

type idxEntry struct {
	entry *Entry
	index int
}

type entries struct {
	entries []*idxEntry
	ids     map[EntryID]*idxEntry
}

type Entries struct {
	queue *entries
	mu    *sync.RWMutex
}

func NewEntries() *Entries {
	return &Entries{
		mu: new(sync.RWMutex),
		queue: &entries{
			entries: []*idxEntry{},
			ids:     map[EntryID]*idxEntry{},
		},
	}
}

func (e *Entries) Add(entry Entry) {
	e.mu.Lock()
	defer e.mu.Unlock()

	_, ok := e.queue.ids[entry.ID]
	if ok {
		e.update(&entry)
		return
	}
	heap.Push(e.queue, &entry)
}

func (e *Entries) Peek() *Entry {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if len(e.queue.entries) > 0 {
		return e.queue.entries[0].entry
	}
	return nil
}

func (e *Entries) Pop() *Entry {
	e.mu.Lock()
	defer e.mu.Unlock()

	return heap.Pop(e.queue).(*Entry)
}

func (e *Entries) Update(entry Entry) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.update(&entry)
}

func (e *Entries) update(entry *Entry) {
	ent, ok := e.queue.ids[entry.ID]
	if ok {
		ent.entry = entry
		heap.Fix(e.queue, ent.index)
	}
}

func (e *Entries) Remove(id EntryID) bool {
	item, ok := e.queue.ids[id]
	if ok {
		heap.Remove(e.queue, item.index)
	}
	return ok
}

/* priority queue implementation */

// sort.Interface
func (e *entries) Len() int {
	return len(e.entries)
}

// sort.Interface
func (e *entries) Less(i, j int) bool {
	return e.entries[i].entry.NextTime.Before(e.entries[j].entry.NextTime)
}

// sort.Interface
func (e *entries) Swap(i, j int) {
	e.entries[i], e.entries[j] = e.entries[j], e.entries[i]
	e.entries[i].index = i
	e.entries[j].index = j
}

// heap.Interface
func (e *entries) Push(val interface{}) {
	entry := val.(*Entry)
	idx := &idxEntry{entry, len(e.entries)}

	e.entries = append(e.entries, idx)
	e.ids[entry.ID] = idx
}

// heap.Interface
func (e *entries) Pop() interface{} {
	len := len(e.entries)
	val, entries := e.entries[len-1], e.entries[:len-1]
	e.entries = entries

	delete(e.ids, val.entry.ID)
	return val.entry
}
