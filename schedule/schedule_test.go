package schedule

import (
	"testing"
	"time"
)

func Test_Schedule_Serialize_Once(t *testing.T) {
	now := time.Now()
	now = now.Truncate(time.Second)

	sched := JSONSchedule{once, now.Format(timeFormat)}

	s, err := sched.Schedule()
	if err != nil {
		t.Fatal(err)
	}

	once := s.(*OnceSchedule)
	if once.t != now {
		t.Fatalf("failed to deserialize schedule: expected='%s', actual='%s'", now, once.t)
	}
}
