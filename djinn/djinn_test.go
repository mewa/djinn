package djinn

import (
	"github.com/mewa/djinn/cron"
	"github.com/mewa/djinn/djinn/job"
	"testing"
	"time"
)

func Test_Djinn(t *testing.T) {
	d := New("add_test01", "http://localhost:4001", nil)

	go d.Start()
	defer d.Stop()

	<-d.Started

	j := job.Job{
		ID: "test001",
	}
	_, err := d.Add(&JobAddRequest{
		j,
	})

	if err != nil {
		t.Fatalf("error: %s", err.Error())
	}

	select {
	case <-d.Done:
	case <-time.After(500 * time.Millisecond):
	}

	entry := d.cron.Entry(cron.EntryID(j.ID))
	if entry.ID != cron.EntryID(j.ID) {
		t.Fatalf("added entry is missing: expectedId=%s, actualId=%s", j.ID, entry.ID)
	}
}

func Test_Membership_Initial(t *testing.T) {
	d1 := New("membership_test01", "http://localhost:4000", []string{
		"membership_test01=http://localhost:4000",
		"membership_test02=http://localhost:4001",
	})
	d2 := New("membership_test02", "http://localhost:4001", []string{
		"membership_test01=http://localhost:4000",
		"membership_test02=http://localhost:4001",
	})

	go d1.Start()
	defer d1.Stop()
	go d2.Start()
	defer d2.Stop()

	for i := 0; i < 2; {
		select {
		case <-d1.Started:
			i++
		case <-d2.Started:
			i++
		case <-time.After(5000 * time.Millisecond):
			if i != 2 {
				t.Fatalf("timed out creating cluster")
			}
			return
		}
	}
}

func Test_Membership_WithJoin(t *testing.T) {
	// initial cluster, we don't know about future members yet
	d1 := New("membership_test01", "http://localhost:4000", nil)
	// add new member to an existing cluster
	d2 := New("membership_test02", "http://localhost:4001", []string{
		"membership_test01=http://localhost:4000",
		"membership_test02=http://localhost:4001",
	})

	err := d1.Start()
	defer d1.Stop()

	if err != nil {
		t.Fatalf("error starting d1: %s", err)
	}

	select {
	case <-d1.Started:
		_, err := d1.AddMember(&AddMemberRequest{
			name: d2.name,
			host: d2.host,
		})

		if err != nil {
			t.Fatalf("could not add member %s", err)
		}

		// d1 is already properly configured
		err = d2.Start()
		defer d2.Stop()

		if err != nil {
			t.Fatalf("error starting d2: %s", err)
		}

		select {
		case <-d2.Started:
			return
		case <-time.After(2000 * time.Millisecond):
		}
	case <-time.After(2000 * time.Millisecond):
	}

	t.Fatal("timed out creating cluster")
}
