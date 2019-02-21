package schedule

import (
	"time"
)

type Once struct {
	t time.Time
	ran bool
}

func RunOnce(t time.Time) *Once {
	return &Once{t, false}
}

func (im * Once) Next(t time.Time) time.Time {
	if !im.ran {
		im.ran = true
		return im.t
	}
	return time.Time{}
}
