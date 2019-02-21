package djinn

import (
	"bytes"
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/mewa/djinn/schedule"
	"io"
	"net/http"
	"time"
)

type CronSchedule struct {
	Cron string `json:"cron"`
}

type RunOnceSchedule struct {
	Once int `json:"once"`
}

func (d *Djinn) runOnceHandler(w http.ResponseWriter, r *http.Request) {
	var buf bytes.Buffer
	io.Copy(&buf, r.Body)

	var s RunOnceSchedule
	json.Unmarshal(buf.Bytes(), &s)

	next := time.Duration(s.Once) * time.Second

	sched := schedule.RunOnce(time.Now().Add(next))
	d.Schedule(sched)
}

func (d *Djinn) scheduleHandler(w http.ResponseWriter, r *http.Request) {
	var buf bytes.Buffer
	io.Copy(&buf, r.Body)

	var s CronSchedule
	json.Unmarshal(buf.Bytes(), &s)

	sched, _ := d.cronParser.Parse(s.Cron)
	d.Schedule(sched)
}

func (d *Djinn) Serve() {
	r := mux.NewRouter()
	r.HandleFunc("/schedule", d.scheduleHandler)
	r.HandleFunc("/run", d.runOnceHandler)

	d.server = &http.Server{
		Handler: r,
		Addr:    ":8000",
	}
	go d.server.ListenAndServe()
}
