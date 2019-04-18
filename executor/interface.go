package executor

import (
	"github.com/mewa/djinn/djinn/job"
)

type Executor interface {
	Execute(job *job.Job, rm job.Remover) error
}
