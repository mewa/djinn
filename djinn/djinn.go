package djinn

import (
	"github.com/coreos/etcd/embed"
	"github.com/coreos/etcd/mvcc"
	"github.com/coreos/etcd/mvcc/mvccpb"
	"github.com/mewa/cron"
	"go.uber.org/zap"
	"net/http"
)

type Djinn struct {
	etcd *embed.Etcd

	cronParser cron.Parser

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

	cronParser := cron.NewParser(
		cron.SecondOptional |
			cron.Minute |
			cron.Hour |
			cron.Dom |
			cron.Month |
			cron.Dow |
			cron.Descriptor,
	)

	return &Djinn{
		etcd: e,

		cronParser: cronParser,

		log:  log,
		Done: make(chan struct{}),
	}
}

// TODO: add implementation
func (d *Djinn) applyEvent(event mvccpb.Event) {
}

func (d *Djinn) Start() {
	select {
	case <-d.etcd.Server.ReadyNotify():

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
