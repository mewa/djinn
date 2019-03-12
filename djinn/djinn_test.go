package djinn

import (
	"github.com/mewa/djinn/cron"
	"github.com/mewa/djinn/djinn/job"
	"strings"
	"testing"
	"time"
)

func (d *Djinn) useClusterConfig(peers []string) {
	d.config.InitialCluster = strings.Join(peers, ",")
	d.config.DNSCluster = ""
}

func Test_Djinn(t *testing.T) {
	d := New("add_test01", "http://localhost:4000", "")
	d.useClusterConfig([]string{
		strings.Join([]string{d.name, d.host.Scheme + "://" + d.host.Host}, "="),
	})

	go d.Start()
	defer d.Stop()

	<-d.Started

	j := job.Job{
		ID: "test001",
	}
	_, err := d.Put(&JobPutRequest{
		Job: j,
	})

	if err != nil {
		t.Fatalf("error: %s", err.Error())
	}

	entry := d.cron.Entry(cron.EntryID(j.ID))
	if entry.ID != cron.EntryID(j.ID) {
		t.Fatalf("added entry is missing: expectedId=%s, actualId=%s", j.ID, entry.ID)
	}
}

func Test_Membership_Initial(t *testing.T) {
	d1 := New("membership_test01", "http://localhost:4000", "two.etcd.test.thedjinn.io")
	d2 := New("membership_test02", "http://localhost:4001", "two.etcd.test.thedjinn.io")

	err := d1.Start()
	defer d1.Stop()

	if err != nil {
		t.Fatalf("error starting d1: %s", err)
	}

	err = d2.Start()
	defer d2.Stop()

	if err != nil {
		t.Fatalf("error starting d2: %s", err)
	}

	for i := 0; i < 2; {
		select {
		case <-d1.Started:
			i++
		case <-d2.Started:
			i++
		case <-time.After(2000 * time.Millisecond):
			if i != 2 {
				t.Fatalf("timed out creating cluster")
			}
			return
		}
	}
}

func Test_AddJob(t *testing.T) {
	d := New("add_test01", "http://localhost:4000", "")
	d.useClusterConfig([]string{
		strings.Join([]string{d.name, d.host.Scheme + "://" + d.host.Host}, "="),
	})

	go d.Start()
	defer d.Stop()

	<-d.Started

	j := job.Job{
		ID: "test001",
	}
	_, err := d.Put(&JobPutRequest{
		Job: j,
	})

	if err != nil {
		t.Fatalf("error: %s", err.Error())
	}

	entry := d.cron.Entry(cron.EntryID(j.ID))
	if entry.ID != cron.EntryID(j.ID) {
		t.Fatalf("added entry is missing: expectedId=%s, actualId=%s", j.ID, entry.ID)
	}
}

func Test_AddJob_2(t *testing.T) {
	d1 := New("membership_test01", "http://localhost:4000", "two.etcd.test.thedjinn.io")
	d2 := New("membership_test02", "http://localhost:4001", "two.etcd.test.thedjinn.io")

	err := d1.Start()
	defer d1.Stop()

	if err != nil {
		t.Fatalf("error starting d1: %s", err)
	}

	err = d2.Start()
	defer d2.Stop()

	if err != nil {
		t.Fatalf("error starting d2: %s", err)
	}

	for i := 0; i < 2; i++ {
		select {
		case <-d1.Started:
		case <-d2.Started:
		case <-time.After(2 * time.Second):
			t.Fatalf("timed out")
		}
	}

	j := job.Job{
		ID: "test001",
	}
	req := &JobPutRequest{
		Job: j,
	}
	_, err = d1.Put(req)

	if err != nil {
		t.Fatalf("error: %s", err.Error())
	}

	// raft log is consistent, but we're only watching changes, so
	// they may not be applied to our cron instance
	select {
	case <-d2.wait.Register(req.Id):
	case <-time.After(100 * time.Millisecond):
	}

	entry := d2.cron.Entry(cron.EntryID(j.ID))
	if entry.ID != cron.EntryID(j.ID) {
		t.Fatalf("added entry is missing: expected=%s, actual=%s", j.ID, entry.ID)
	}
}
