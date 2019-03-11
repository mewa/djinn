package djinn

import (
	"context"
	"github.com/coreos/etcd/etcdserver/etcdserverpb"
	"github.com/coreos/etcd/etcdserver/membership"
	"github.com/mewa/djinn/djinn/job"
	"net/url"
)

type JobPutRequest struct {
	job.Job
}

type JobPutResponse struct {
	job.Job
}

type AddMemberRequest struct {
	name string
	host string
}

type AddMemberResponse struct {
	peers []*membership.Member
}

func (d *Djinn) Put(req *JobPutRequest) (*JobPutResponse, error) {
	resp := &JobPutResponse{
		req.Job,
	}

	val, err := job.Marshal(&req.Job)

	if err != nil {
		return nil, err
	}

	_, err = d.etcd.Server.Put(context.TODO(), &etcdserverpb.PutRequest{
		Key:    []byte(req.Job.ID),
		Value:  val,
		PrevKv: true,
	})

	if err != nil {
		return nil, err
	}

	return resp, err
}

func (d *Djinn) AddMember(req *AddMemberRequest) (*AddMemberResponse, error) {
	membUrl, err := url.Parse(req.host)
	member := membership.NewMember(req.name, []url.URL{*membUrl}, d.cluster, nil)
	resp, err := d.etcd.Server.AddMember(context.TODO(), *member)

	if err != nil {
		return nil, err
	}

	return &AddMemberResponse{
		peers: resp,
	}, err
}
