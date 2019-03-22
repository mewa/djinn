package djinn

import (
	"bytes"
	"encoding/json"
	"github.com/gorilla/mux"
	"io"
	"net"
	"net/http"
)

func (d *Djinn) statusHandler(w http.ResponseWriter, r *http.Request) {
	data, _ := json.Marshal(StatusResponse{
		Running: d.running,
		Name:    d.name,
		Host:    d.host.String(),
	})

	var buf bytes.Buffer
	buf.Write(data)
	io.Copy(w, &buf)
}

func (d *Djinn) Serve() error {
	r := mux.NewRouter()
	r.HandleFunc("/status", d.statusHandler)

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
