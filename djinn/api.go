package djinn

import (
	"context"
	"github.com/coreos/etcd/etcdserver/etcdserverpb"
	"github.com/coreos/etcd/etcdserver/membership"
	"github.com/mewa/djinn/djinn/job"
	"net/url"
	"time"
	"encoding/json"
)

type JobPutRequest struct {
	Id      uint64 `json:"id"`
	job.Job `json:"job"`
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

	if req.Id == 0 {
		req.Id = d.idGen.Next()
	}

	val, err := json.Marshal(&req)
	if err != nil {
		return nil, err
	}

	ctx, _ := context.WithTimeout(context.TODO(), time.Millisecond*time.Duration(3*d.config.ElectionMs))
	ch := d.wait.Register(req.Id)

	_, err = d.etcd.Server.Put(ctx, &etcdserverpb.PutRequest{
		Key:    []byte(req.Job.ID),
		Value:  val,
		PrevKv: true,
	})

	if err != nil {
		d.wait.Trigger(req.Id, nil)
		return nil, err
	}

	select {
	case <-ch:
	case <-ctx.Done():
		d.wait.Trigger(req.Id, nil)
		return nil, ctx.Err()
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
