package djinn

import (
	"context"
	"encoding/json"
	"github.com/coreos/etcd/embed"
	"github.com/coreos/etcd/mvcc"
	"github.com/coreos/etcd/mvcc/mvccpb"
	"github.com/coreos/etcd/pkg/idutil"
	"github.com/coreos/etcd/pkg/wait"
	"github.com/mewa/djinn/cron"
	"github.com/mewa/djinn/djinn/job"
	"github.com/mewa/djinn/executor"
	"github.com/mewa/djinn/storage"
	"go.opencensus.io/stats"
	"go.uber.org/zap"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"
)

type Djinn struct {
	etcd   *embed.Etcd
	config *embed.Config

	cluster string
	name    string

	serverUrl *url.URL
	clientUrl *url.URL
	apiServer string
	bindAll   bool

	cron     cron.Cron
	storage  storage.Storage
	executor executor.Executor

	server *http.Server

	jobs     map[job.ID]*job.Job
	progress map[job.ID]bool

	wait  wait.Wait
	idGen *idutil.Generator

	log *zap.Logger

	running bool

	Started chan struct{}
	stop    chan struct{}
	Done    chan struct{}

	mu *sync.Mutex
}

func New(name, server, clientPort, apiServer string, bindAll bool, discovery string, storage storage.Storage, executor executor.Executor) (*Djinn, error) {
	log, _ := zap.NewDevelopment()

	conf := embed.NewConfig()

	conf.Name = name
	conf.Dir = "/tmp/djinn/" + name

	// we don't want to persist data on disk
	err := os.RemoveAll(conf.Dir)

	if err != nil {
		return nil, err
	}

	serverUrl, err := url.Parse(server)
	if err != nil {
		return nil, err
	}

	clientUrl := *serverUrl
	clientUrl.Host = clientUrl.Hostname() + ":" + clientPort

	conf.InitialCluster = ""
	conf.DNSCluster = discovery

	djinn := &Djinn{
		config: conf,

		cluster: "default",
		name:    name,

		serverUrl: serverUrl,
		clientUrl: &clientUrl,
		apiServer: apiServer,
		bindAll:   bindAll,

		storage:  storage,
		executor: executor,

		cron: cron.New(),

		jobs:     map[job.ID]*job.Job{},
		progress: map[job.ID]bool{},

		wait: wait.New(),

		log: log,

		Started: make(chan struct{}, 1),
		stop:    make(chan struct{}),
		Done:    make(chan struct{}),

		mu: new(sync.Mutex),
	}

	return djinn, nil
}

func (d *Djinn) Start() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	err := d.configure()
	if err != nil {
		return err
	}

	e, err := embed.StartEtcd(d.config)

	if err != nil {
		d.log.Error(d.name+": could not start server",
			zap.Error(err),
			zap.String("initial-cluster", d.config.InitialCluster))
		return err
	}

	d.idGen = idutil.NewGenerator(uint16(e.Server.ID()), time.Now())
	d.etcd = e

	err = d.initMetrics()
	if err != nil {
		d.log.Error("could not initialise metrics", zap.String("name", d.config.Name), zap.Error(err))
		return err
	} else {
		d.log.Info("metrics initialised", zap.String("name", d.config.Name))
	}

	err = d.Serve()
	if err != nil {
		d.log.Error("could not start API server", zap.String("name", d.config.Name), zap.String("url", d.apiServer), zap.Error(err))
		return err
	} else {
		d.log.Info("API server up and running", zap.String("name", d.config.Name), zap.String("url", d.apiServer))
	}

	go d.run()
	return nil
}

func (d *Djinn) run() {
	defer d.server.Close()

	select {
	case <-d.etcd.Server.ReadyNotify():
		d.running = true
		d.cron.Start()

		d.log.Info("djinn ready")

		w := d.etcd.Server.Watchable()
		var ws mvcc.WatchStream = w.NewWatchStream()
		ch := ws.Chan()

		// watch all changes starting from revision 1
		id := ws.Watch([]byte{0x0}, []byte{0xff}, 1)
		if id == -1 {
			panic("could not watch changes")
		}

		d.Started <- struct{}{}
	Loop:
		for {
			select {
			case <-d.stop:
				break Loop
			case r := <-ch:
				for _, event := range r.Events {
					d.applyEvent(event)
				}
			}
		}
	}

	d.log.Info("djinn shutting down", zap.String("name", d.config.Name))

	d.etcd.Close()
	d.Done <- struct{}{}
}

func (d *Djinn) Stop() {
	if d.running {
		d.stop <- struct{}{}
		<-d.Done
	}

	d.cron.Stop()
	d.running = false

	d.log.Info("djinn stopped", zap.String("name", d.config.Name))
}

func (d *Djinn) applyEvent(event mvccpb.Event) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if event.Type == mvccpb.PUT {
		var req JobPutRequest
		err := json.Unmarshal(event.Kv.Value, &req)

		if err != nil {
			d.log.Error("could not unmarshal event", zap.Error(err))
			return
		}

		req.Job.Handler = job.Handler{
			Run: d.runJob,
		}

		d.putJob(&req.Job)
		d.wait.Trigger(req.Id, req.Job)
		return
	}
	if event.Type == mvccpb.DELETE {
		jid := job.ID(event.Kv.Key)

		saved, exists := d.jobs[jid]
		if exists {
			d.deleteJob(saved)
		}

		hash := uint64(jid.Hash())
		d.wait.Trigger(hash, jid)
		return
	}
}

func (d *Djinn) putJob(j *job.Job) error {
	saved, exists := d.jobs[j.ID]

	if !exists {
		d.jobs[j.ID] = j
		d.cron.PutEntry(cron.Entry{
			ID:       cron.EntryID(j.ID),
			Schedule: j,
			Job:      j,
			Next:     j.NextTime,
			Prev:     j.PrevTime,
		})
	} else {
		saved.Update(j)
	}
	return nil
}

func (d *Djinn) deleteJob(j *job.Job) {
	delete(d.jobs, j.ID)
	d.cron.DeleteEntry(cron.EntryID(j.ID))
}

func (d *Djinn) runJob(j *job.Job) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.isLeader() {
		return
	}

	running := d.progress[j.ID]
	if running {
		d.log.Info("job in progress, skipping", zap.String("name", d.config.Name), zap.Stringer("job", j))
		return
	}
	d.progress[j.ID] = true

	go func(j job.Job) {
		defer func() {
			d.mu.Lock()
			defer d.mu.Unlock()
			d.progress[j.ID] = false
		}()

		d.log.Info("running job", zap.String("name", d.config.Name), zap.Stringer("job", &j))
		// TODO: this doesn't take care of leadership losses while in between states
		if j.State.State == job.Initial || j.State.State == job.Started {
			// by the time we reach this point PrevTime holds current execution's time
			j.State = job.State{job.Starting, j.PrevTime.Unix()}
			req := &JobPutRequest{
				Job: j,
			}

			_, err := d.Put(req)
			if err != nil {
				d.log.Error("error starting job", zap.String("name", d.config.Name), zap.String("job_id", string(j.ID)), zap.Error(err))

				err = d.storage.SaveJobState(j.ID, job.State{job.Error, j.State.Time})
				if err != nil {
					d.log.Error("error saving job state", zap.String("name", d.config.Name), zap.String("job_id", string(j.ID)), zap.Error(err))
				}
				return
			} else {
				err = d.storage.SaveJobState(j.ID, req.Job.State)
				if err != nil {
					d.log.Error("error saving job state", zap.String("name", d.config.Name), zap.String("job_id", string(j.ID)), zap.Error(err))
				}
			}

			// TODO: handle job execution failures
			err = d.executeJob(&j)

			if err != nil {
				j.State.State = job.Error

				err = d.storage.SaveJobState(j.ID, j.State)
				if err != nil {
					d.log.Error("error saving job state", zap.String("name", d.config.Name), zap.String("job_id", string(j.ID)), zap.Error(err))
				}

				return
			} else {
				j.State.State = job.Started
				req = &JobPutRequest{
					Job: j,
				}

				_, err = d.Put(req)
				if err != nil {
					d.log.Error("error updating job state", zap.String("name", d.config.Name), zap.String("job_id", string(j.ID)), zap.Error(err))
				}
			}

			err = d.storage.SaveJobState(j.ID, j.State)
			if err != nil {
				d.log.Error("error saving job state", zap.String("name", d.config.Name), zap.String("job_id", string(j.ID)), zap.Error(err))
			}

			if j.Schedule().Next(time.Now()).IsZero() {
				rmReq := &JobDeleteRequest{
					JobId: j.ID,
				}

				err = d.Delete(rmReq)
				if err != nil {
					d.log.Error("error deleting job", zap.String("name", d.config.Name), zap.String("job_id", string(j.ID)), zap.Error(err))
				}
			}
		}
	}(*j)
}

func (d *Djinn) executeJob(j *job.Job) error {
	stats.Record(context.Background(), MJobExecutions.M(1))

	err := d.executor.Execute(j, job.Remover(d))

	return err
}

func (d *Djinn) isLeader() bool {
	return d.etcd.Server.ID() == d.etcd.Server.Leader()
}

// implements job.Remover
func (d *Djinn) Remove(j *job.Job) error {
	rmReq := &JobDeleteRequest{
		JobId: j.ID,
	}

	err := d.Delete(rmReq)
	if err != nil {
		d.log.Error("error deleting job", zap.String("name", d.config.Name), zap.String("job_id", string(j.ID)), zap.Error(err))
	}
	return err
}
