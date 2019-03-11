package djinn

import (
	"context"
	"github.com/coreos/etcd/etcdserver/etcdserverpb"
	"github.com/mewa/djinn/djinn/job"
)

type JobPutRequest struct {
	job.Job
}

type JobPutResponse struct {
	job.Job
}

func (d *Djinn) Put(req *JobPutRequest) (*JobPutResponse, error) {
	resp := &JobPutResponse{
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
