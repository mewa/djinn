package djinn

import (
	"context"
	"github.com/coreos/etcd/etcdserver/etcdserverpb"
	"github.com/mewa/djinn/djinn/job"
)

type JobAddRequest struct {
	job.Job
}

type JobAddResponse struct {
	job.Job
}

func (d *Djinn) Add(req *JobAddRequest) (*JobAddResponse, error) {
	resp := &JobAddResponse{
		req.Job,
	}

	val, err := job.Marshal(&req.Job)

	if err != nil {
		return nil, err
	}

	_, err := d.etcd.Server.Put(context.TODO(), &etcdserverpb.PutRequest{
		Key: []byte(req.Job.ID),
		Value: val,
		PrevKv: true,
	})

	if err != nil {
		return nil, err
	}

	return resp, err
}
