package djinn

import (
	"encoding/json"
	"github.com/coreos/etcd/embed"
	"github.com/coreos/etcd/mvcc"
	"github.com/coreos/etcd/mvcc/mvccpb"
	"github.com/coreos/etcd/pkg/idutil"
	"github.com/coreos/etcd/pkg/wait"
	"github.com/mewa/djinn/cron"
	"github.com/mewa/djinn/djinn/job"
	"go.uber.org/zap"
	"net/http"
	"net/url"
	"os"
	"time"
)

type Djinn struct {
	etcd   *embed.Etcd
	config *embed.Config

	cluster string
	name    string
	host    *url.URL

	cron cron.Cron

	server *http.Server

	jobs map[job.ID]*job.Job

	wait  wait.Wait
	idGen *idutil.Generator

	log *zap.Logger

	running bool

	Started chan struct{}
	stop    chan struct{}
	Done    chan struct{}
}

func New(name, host string, discovery string) *Djinn {
	log, _ := zap.NewDevelopment()

	conf := embed.NewConfig()

	conf.Name = name
	conf.Dir = "/tmp/djinn/" + name

	// we don't want to persist data on disk
	os.RemoveAll(conf.Dir)

	// disable client access
	conf.LCUrls = nil
	conf.ACUrls = nil

	hostUrl, _ := url.Parse(host)
	conf.APUrls = []url.URL{*hostUrl}
	conf.LPUrls = []url.URL{*hostUrl}

	conf.InitialCluster = ""
	conf.DNSCluster = discovery

	djinn := &Djinn{
		config: conf,

		cluster: "default",
		name:    name,
		host:    hostUrl,

		cron: cron.New(),

		jobs: map[job.ID]*job.Job{},

		wait: wait.New(),

		log: log,

		Started: make(chan struct{}, 1),
		stop:    make(chan struct{}),
		Done:    make(chan struct{}),
	}

	return djinn
}

func (d *Djinn) Start() error {
	e, err := embed.StartEtcd(d.config)

	if err != nil {
		d.log.Error(d.name+": could not start server",
			zap.Error(err),
			zap.String("initial-cluster", d.config.InitialCluster))
		return err
	}

	d.idGen = idutil.NewGenerator(uint16(e.Server.ID()), time.Now())
	d.etcd = e
	go d.run()
	return nil
}

func (d *Djinn) run() {
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

	if event.Type == mvccpb.PUT {
		var req JobPutRequest
		err := json.Unmarshal(event.Kv.Value, &req)

		if err != nil {
			d.log.Error("could not unmarshal event", zap.Error(err))
			return
		}

		req.Job.Handler = job.Handler{
			Remove: d.deleteJob,
			Run:    d.runJob,
		}

		d.putJob(&req.Job)
		d.wait.Trigger(req.Id, req.Job)
		return
	}
	if event.Type == mvccpb.DELETE {
		panic("not implemented")
		return
	}
}

func (d *Djinn) putJob(j *job.Job) error {
	d.jobs[j.ID] = j
	d.cron.PutEntry(cron.Entry{
		ID:       cron.EntryID(j.ID),
		Schedule: j,
		Job:      j,
		Next:     j.NextTime,
		Prev:     j.PrevTime,
	})
	return nil
}

func (d *Djinn) deleteJob(j *job.Job) {
	delete(d.jobs, j.ID)
	d.cron.DeleteEntry(cron.EntryID(j.ID))
}

func (d *Djinn) runJob(j *job.Job) {}
