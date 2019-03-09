package djinn

import (
	"github.com/mewa/djinn/djinn/job"
	"github.com/coreos/etcd/embed"
	"github.com/coreos/etcd/mvcc"
	"github.com/coreos/etcd/mvcc/mvccpb"
	"github.com/mewa/djinn/cron"
	"go.uber.org/zap"
	"net/http"
)

type Djinn struct {
	etcd *embed.Etcd

	cron cron.Cron

	server    *http.Server
	schedules map[string]cron.EntryID

	log *zap.Logger

	Done chan struct{}
}

// TODO: add configuration
// host string, peers []string
func New() *Djinn {
	log, _ := zap.NewDevelopment()

	// TODO: use tmpfs as storage
	conf := embed.NewConfig()
	conf.Dir = "/tmp/djinn/mewa.etcd"

	// disable client access
	conf.LCUrls = nil
	conf.ACUrls = nil

	e, err := embed.StartEtcd(conf)

	if err != nil {
		log.Error("could not start server", zap.Error(err))
		return nil
	}

	return &Djinn{
		etcd: e,

		cron: cron.New(),

		log:  log,
		Done: make(chan struct{}),
	}
}

func (d *Djinn) Start() {
	select {
	case <-d.etcd.Server.ReadyNotify():
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

		for {
			select {
			case r := <-ch:
				for _, event := range r.Events {
					d.applyEvent(event)
				}
			}
		}
	}

	d.log.Info("djinn shutting down")

	d.etcd.Close()
}

func (d *Djinn) applyEvent(event mvccpb.Event) {
	var j job.Job
	err := job.Unmarshal(event.Kv.Value, &j)

	if err != nil {
		d.log.Error("could not unmarshal event", zap.Error(err))
		return
	}

	j.RemoveHandler = d.removeJob

	d.upsertJob(&j)
}

func (d *Djinn) upsertJob(j *job.Job) error {
	d.cron.UpsertEntry(cron.Entry{
		ID: cron.EntryID(j.ID),
		Schedule: j,
		Next: j.NextTime,
		Prev: j.PrevTime,
	})
	return nil
}

func (d *Djinn) removeJob(j *job.Job) {
	d.cron.RemoveEntry(cron.EntryID(j.ID))
}
