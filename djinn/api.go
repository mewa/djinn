package djinn

import (
	"context"
	"encoding/json"
	"github.com/coreos/etcd/etcdserver/etcdserverpb"
	"github.com/coreos/etcd/etcdserver/membership"
	"github.com/mewa/djinn/djinn/job"
	"net/url"
	"time"
)

type JobPutRequest struct {
	Id      uint64 `json:"id"`
	job.Job `json:"job"`
}

type JobPutResponse struct {
	Next int64 `json:"next_execution"`
}

type JobDeleteRequest struct {
	JobId job.ID `json:"job_id"`
}

type AddMemberRequest struct {
	name string
	host string
}

type AddMemberResponse struct {
	peers []*membership.Member
}

func (d *Djinn) Put(req *JobPutRequest) (*JobPutResponse, error) {
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

	var j job.Job
	select {
	case val := <-ch:
		j = val.(job.Job)
	case <-ctx.Done():
		d.wait.Trigger(req.Id, nil)
		return nil, ctx.Err()
	}

	// TODO: replace with actual execution time as calculated by
	// the cron instance
	resp := &JobPutResponse{
		Next: j.Next(time.Now()).Unix(),
	}

	return resp, err
}

func (d *Djinn) Delete(req *JobDeleteRequest) error {
	hash := uint64(req.JobId.Hash())
	ch := d.wait.Register(hash)

	ctx, _ := context.WithTimeout(context.TODO(), time.Millisecond*time.Duration(10*d.config.ElectionMs))
	_, err := d.etcd.Server.DeleteRange(ctx, &etcdserverpb.DeleteRangeRequest{
		Key: []byte(req.JobId),
	})

	if err != nil {
		d.wait.Trigger(hash, nil)
		return err
	}

	select {
	case <-ch:
	case <-ctx.Done():
		d.wait.Trigger(hash, nil)
		return ctx.Err()
	}

	return nil
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
