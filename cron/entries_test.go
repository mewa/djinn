package cron

import (
	"testing"
	"time"
)

func Test_Peek(t *testing.T) {
	e := NewEntries()

	entry := Entry{
		ID:       "e1",
		NextTime: time.Unix(100000, 0),
	}
	e.Add(entry)

	if e.Peek() == nil {
		t.Fatalf("nil peek")
	}
	if e.Peek().ID != entry.ID {
		t.Fatalf("wrong id: expected='%s', actual='%s'", entry, e.Peek())
	}
}

func Test_AddPop(t *testing.T) {
	e := NewEntries()

	step := int64(1000)
	seconds := int64(100000)
	entry1 := Entry{
		ID:       "e1",
		NextTime: time.Unix(seconds, 0),
	}

	e.Add(entry1)
	if e.Peek().ID != entry1.ID {
		t.Fatalf("wrong id: expected='%s', actual='%s'", entry1, e.Peek())
	}

	seconds = seconds - step
	entry2 := Entry{
		ID:       "e2",
		NextTime: time.Unix(seconds, 0),
	}

	e.Add(entry2)
	if e.Peek().ID != entry2.ID {
		t.Fatalf("wrong id: expected='%s', actual='%s'", entry2, e.Peek())
	}

	seconds = seconds - step
	entry3 := Entry{
		ID:       "e3",
		NextTime: time.Unix(seconds, 0),
	}

	e.Add(entry3)
	if e.Peek().ID != entry3.ID {
		t.Fatalf("wrong id: expected='%s', actual='%s'", entry3, e.Peek())
	}

	e.Pop()
	if e.Peek().ID != entry2.ID {
		t.Fatalf("wrong id: expected='%s', actual='%s'", entry2, e.Peek())
	}

	e.Pop()
	if e.Peek().ID != entry1.ID {
		t.Fatalf("wrong id: expected='%s', actual='%s'", entry1, e.Peek())
	}

	e.Pop()
	if e.Peek() != nil {
		t.Fatalf("wrong id: expected=nil, actual='%s'", e.Peek())
	}
}

func Test_Update(t *testing.T) {
	e := NewEntries()

	step := int64(1000)
	seconds := int64(100000)
	entry1 := Entry{
		ID:       "e1",
		NextTime: time.Unix(seconds, 0),
	}

	seconds = seconds - step
	entry2 := Entry{
		ID:       "e2",
		NextTime: time.Unix(seconds, 0),
	}

	e.Add(entry1)
	e.Add(entry2)

	seconds = seconds - step
	entry1.NextTime = time.Unix(seconds, 0)

	if e.Peek().ID != entry2.ID {
		t.Fatalf("wrong id: expected='%s', actual='%s'", entry2, e.Peek())
	}

	e.Update(entry1)
	if e.Peek().ID != entry1.ID {
		t.Fatalf("wrong id: expected='%s', actual='%s'", entry1, e.Peek())
	}
}
