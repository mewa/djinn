package schedule

import (
	"testing"
	"time"
)

func Test_Schedule_Serialize_Once(t *testing.T) {
	now := time.Now()
	now = now.Truncate(time.Second)

	sched := JSONSchedule{TypeOnce, Once(now).Serialize()}

	s, err := sched.Schedule()
	if err != nil {
		t.Fatal(err)
	}

	once := s.(*OnceSchedule)
	if once.Time != now {
		t.Fatalf("failed to deserialize schedule: expected='%s', actual='%s'", now, once.Time)
	}
}
