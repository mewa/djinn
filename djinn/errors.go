package djinn

import (
	"errors"
)

var (
	ErrUnknownScheduleType = errors.New("unknown schedule type")
	ErrCannotResolveService = errors.New("could not resolve service")
)
