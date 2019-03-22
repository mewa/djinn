package djinn

import (
	"github.com/mewa/djinn/cron"
	"github.com/mewa/djinn/djinn/job"
	"github.com/mewa/djinn/schedule"
	"math"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

func (d *Djinn) useClusterConfig(peers []string) {
	d.config.InitialCluster = strings.Join(peers, ",")
	d.config.DNSCluster = ""
}

func leader(djinns ...*Djinn) *Djinn {
	for _, djinn := range djinns {
		if djinn.etcd.Server.Leader() == djinn.etcd.Server.ID() {
			return djinn
		}
	}
	return nil
}

func newStorage() *testStorage {
	return &testStorage{map[job.ID][]job.State{}, new(sync.Mutex)}
}

type testStorage struct {
	states map[job.ID][]job.State
	mu     *sync.Mutex
}

func (s *testStorage) SaveJobState(id job.ID, state job.State) {
	s.mu.Lock()
	defer s.mu.Unlock()

	val := s.states[id]
	s.states[id] = append(val, state)
}

func Test_Membership_Initial(t *testing.T) {
	d1 := New("membership_test01", "http://localhost:4000", "localhost:4444", "two.etcd.test.thedjinn.io", newStorage())
	d2 := New("membership_test02", "http://localhost:4001", "localhost:4445", "two.etcd.test.thedjinn.io", newStorage())

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
	d := New("add_test01", "http://localhost:4000", "localhost:4444", "", newStorage())
	d.useClusterConfig([]string{
		strings.Join([]string{d.name, d.host.Scheme + "://" + d.host.Host}, "="),
	})

	go d.Start()
	defer d.Stop()

	<-d.Started

	j := job.Job{
		ID: "test-add-job",
		Descriptor: schedule.JSONSchedule{
			ScheduleType: 1,
			ScheduleData: "* * * * * *",
		},
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
	store := newStorage()

	d1 := New("membership_test01", "http://localhost:4000", "localhost:4444", "two.etcd.test.thedjinn.io", store)
	d2 := New("membership_test02", "http://localhost:4001", "localhost:4445", "two.etcd.test.thedjinn.io", store)

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
		ID: "test-add-job-2",
		Descriptor: schedule.JSONSchedule{
			ScheduleType: 1,
			ScheduleData: "* * * * * *",
		},
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

func Test_ExecuteJob_Once_1(t *testing.T) {
	store := newStorage()
	d := New("execute_once_test", "http://localhost:4000", "localhost:4444", "", store)
	d.useClusterConfig([]string{
		strings.Join([]string{d.name, d.host.Scheme + "://" + d.host.Host}, "="),
	})

	err := d.Start()
	defer d.Stop()

	if err != nil {
		t.Fatalf("error starting djinn: %s", err)
	}

	select {
	case <-d.Started:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out")
	}

	exec := time.Now()
	j := job.Job{
		ID: "test-execute-once-job",
		Descriptor: schedule.JSONSchedule{
			ScheduleType: 0,
			ScheduleData: schedule.Once(exec).Serialize(),
		},
	}
	req := &JobPutRequest{
		Job: j,
	}

	_, err = d.Put(req)

	if err != nil {
		t.Fatal("error", err)
	}

	delay := 2200
	delayC := time.After(time.Duration(delay) * time.Millisecond)

	<-delayC
	actual := store.states[j.ID]
	expected := []job.State{job.State{job.Starting, exec.Unix()}, job.State{job.Started, exec.Unix()}}

	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("invalid job states: expected='%v', actual='%v'", expected, actual)
	}
}

func Test_ExecuteJob_Cron_1(t *testing.T) {
	store := newStorage()
	d := New("execute_test", "http://localhost:4000", "localhost:4444", "", store)
	d.useClusterConfig([]string{
		strings.Join([]string{d.name, d.host.Scheme + "://" + d.host.Host}, "="),
	})

	err := d.Start()
	defer d.Stop()

	if err != nil {
		t.Fatalf("error starting djinn: %s", err)
	}

	select {
	case <-d.Started:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out")
	}

	j := job.Job{
		ID: "test-execute-job",
		Descriptor: schedule.JSONSchedule{
			ScheduleType: 1,
			ScheduleData: "* * * * * *",
		},
	}
	req := &JobPutRequest{
		Job: j,
	}

	resp, err := d.Put(req)

	if err != nil {
		t.Fatal("error", err)
	}

	firstExec := time.Unix(resp.Next, 0)

	delay := 2200
	delayC := time.After(time.Duration(delay) * time.Millisecond)

	<-delayC
	actual := store.states[j.ID]
	expected := []job.State{}

	for i := 1; i <= int(math.Ceil(float64(delay)/float64(1000))); i = i + 1 {
		expected = append(expected, job.State{
			job.Starting,
			firstExec.Add(time.Duration(i) * time.Second).Unix(),
		})
		expected = append(expected, job.State{
			job.Started,
			firstExec.Add(time.Duration(i) * time.Second).Unix(),
		})
	}

	if len(expected) - len(actual) > 2 {
		t.Fatal("too big execution drift")
	}

	if len(expected) - len(actual) >= 2 {
		t.Log("execution drift:", len(expected) - len(actual))
	}

	if !reflect.DeepEqual(expected[:len(actual)], actual) {
		t.Fatalf("invalid job states: expected='%v', actual='%v'", expected, actual)
	}
}
