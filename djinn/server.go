package djinn

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/mewa/djinn/djinn/job"
	"github.com/mewa/djinn/schedule"
	"go.opencensus.io/exporter/prometheus"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	"go.uber.org/zap"
	"io"
	"net"
	"net/http"
	"time"
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

type PutOnceJobRequest struct {
	Expression string `json:"time"`
}

type PutJobResponse struct {
	Next int64 `json:"next_execution"`
}

func (d *Djinn) cronHandler(w http.ResponseWriter, r *http.Request) {
	ctx, _ := tag.New(context.Background(), tag.Insert(KeyType, "cron"), tag.Insert(KeyMethod, r.Method))
	start := time.Now()

	vars := mux.Vars(r)
	jobId := vars["job"]

	var buf bytes.Buffer
	io.Copy(&buf, r.Body)

	var s PutCronJobRequest
	json.Unmarshal(buf.Bytes(), &s)

	// validate input
	descr := schedule.JSONSchedule{schedule.TypeSpec, s.Expression}

	if _, err := descr.Schedule(); err != nil {
		ctx, _ = tag.New(ctx, tag.Insert(KeyStatus, "400"))
		stats.Record(ctx, MHttpRequestLatency.M(float64(time.Now().Sub(start)/time.Millisecond)))
		stats.Record(ctx, MHttpRequests.M(1))

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
		ctx, _ = tag.New(ctx, tag.Insert(KeyStatus, "503"))
		stats.Record(ctx, MHttpRequestLatency.M(float64(time.Now().Sub(start)/time.Millisecond)))
		stats.Record(ctx, MHttpRequests.M(1))

		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(err.Error()))
		return
	}

	httpResp, err := json.Marshal(&PutJobResponse{resp.Next})

	if err != nil {
		ctx, _ = tag.New(ctx, tag.Insert(KeyStatus, "500"))
		stats.Record(ctx, MHttpRequestLatency.M(float64(time.Now().Sub(start)/time.Millisecond)))
		stats.Record(ctx, MHttpRequests.M(1))

		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	ctx, _ = tag.New(ctx, tag.Insert(KeyStatus, "200"))
	stats.Record(ctx, MHttpRequestLatency.M(float64(time.Now().Sub(start)/time.Millisecond)))
	stats.Record(ctx, MHttpRequests.M(1))

	w.Write(httpResp)
}

func (d *Djinn) onceHandler(w http.ResponseWriter, r *http.Request) {
	ctx, _ := tag.New(context.Background(), tag.Insert(KeyType, "once"), tag.Insert(KeyMethod, r.Method))
	start := time.Now()

	vars := mux.Vars(r)
	jobId := vars["job"]

	var buf bytes.Buffer
	io.Copy(&buf, r.Body)

	var s PutOnceJobRequest
	json.Unmarshal(buf.Bytes(), &s)

	// validate input
	descr := schedule.JSONSchedule{schedule.TypeOnce, buf.String()}

	if _, err := descr.Schedule(); err != nil {
		ctx, _ = tag.New(ctx, tag.Insert(KeyStatus, "400"))
		stats.Record(ctx, MHttpRequestLatency.M(float64(time.Now().Sub(start)/time.Millisecond)))
		stats.Record(ctx, MHttpRequests.M(1))

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
		ctx, _ = tag.New(ctx, tag.Insert(KeyStatus, "503"))
		stats.Record(ctx, MHttpRequestLatency.M(float64(time.Now().Sub(start)/time.Millisecond)))
		stats.Record(ctx, MHttpRequests.M(1))

		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(err.Error()))
		return
	}

	httpResp, err := json.Marshal(&PutJobResponse{resp.Next})

	if err != nil {
		ctx, _ = tag.New(ctx, tag.Insert(KeyStatus, "500"))
		stats.Record(ctx, MHttpRequestLatency.M(float64(time.Now().Sub(start)/time.Millisecond)))
		stats.Record(ctx, MHttpRequests.M(1))

		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	ctx, _ = tag.New(ctx, tag.Insert(KeyStatus, "200"))
	stats.Record(ctx, MHttpRequestLatency.M(float64(time.Now().Sub(start)/time.Millisecond)))
	stats.Record(ctx, MHttpRequests.M(1))

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

	promExport, err := prometheus.NewExporter(prometheus.Options{
		Namespace: "djinn",
	})
	if err != nil {
		d.log.Error("failed to create Prometheus exporter", zap.Error(err))
		return err
	}
	view.RegisterExporter(promExport)

	r.Handle("/metrics", promExport)
	r.HandleFunc("/status", d.statusHandler)
	r.HandleFunc("/{job}/cron", d.cronHandler).
		Methods("PUT")
	r.HandleFunc("/{job}/once", d.onceHandler).
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
