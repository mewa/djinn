package utils

import (
	"math/rand"
	"time"
)

func Backoff(min, timeout time.Duration, fn func () error) {
	var elapsed time.Duration

	wait := min + min * time.Duration(rand.Float32())

retry:
	if elapsed < timeout {
		if err := fn(); err != nil {
			time.Sleep(wait)
			elapsed += wait
			wait *= time.Duration(1.5 + 0.75 * rand.Float32())
			goto retry
		}
	}
}
