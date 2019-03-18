package storage

import (
	"github.com/mewa/djinn/djinn/job"
)

type Storage interface {
	SaveJobState(id job.ID, state job.State)
}
