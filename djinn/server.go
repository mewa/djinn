package djinn

import (
	"bytes"
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/mewa/djinn/djinn/job"
	"io"
	"net"
	"net/http"
)

type StatusResponse struct {
	Running bool   `json:"running"`
	Name    string `json:"name"`
	Server  string `json:"server"`
	Client  string `json:"client"`
}

type PutCronJobRequest struct {
	Expression string `json:"schedule"`
}

type PutCronJobResponse struct {
	Next int64 `json:"next_execution"`
}

func (d *Djinn) cronHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobId := vars["job"]

	var buf bytes.Buffer
	io.Copy(&buf, r.Body)

	var s PutCronJobRequest
	json.Unmarshal(buf.Bytes(), &s)

	// validate input
	descr := schedule.JSONSchedule{schedule.TypeSpec, s.Expression}

	if _, err := descr.Schedule(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	resp, err := d.Put(&JobPutRequest{
		Job: job.Job{
			ID:         job.ID(jobId),
			Descriptor: descr,
		},
	})

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	httpResp, err := json.Marshal(&PutCronJobResponse{resp.Next})

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	w.Write(httpResp)
}

func (d *Djinn) statusHandler(w http.ResponseWriter, r *http.Request) {
	data, _ := json.Marshal(StatusResponse{
		Running: d.running,
		Name:    d.name,
		Server:  d.serverUrl.String(),
		Client:  d.clientUrl.String(),
	})

	var buf bytes.Buffer
	buf.Write(data)
	io.Copy(w, &buf)
}

func (d *Djinn) Serve() error {
	r := mux.NewRouter()
	r.HandleFunc("/status", d.statusHandler)
	r.HandleFunc("/{job}/cron", d.cronHandler).
		Methods("PUT")

	d.server = &http.Server{
		Handler: r,
	}

	l, err := net.Listen("tcp", d.apiServer)
	if err != nil {
		return err
	}

	go d.server.Serve(l)
	return nil
}
