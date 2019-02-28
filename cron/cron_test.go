package cron

import (
	"github.com/mewa/djinn/schedule"
	"testing"
	"time"
)

func Test_UpdateDeleted(t *testing.T) {
	cron := New()
	cron.Start()

	cron.AddEntry(Entry{
		ID:       "test01",
		Schedule: schedule.RunOnce(time.Now()),
	})

	cron.RemoveEntry("test01")

	time.Sleep(1 * time.Millisecond)
	cron.Stop()
}

func Test_SingleExecution_NextTimeIsZero(t *testing.T) {
	// create cron
	// add RunOnce entry
	// verify it's only ran a single time
}
