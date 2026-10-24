package gui

import (
	"encoding/json"
	"net/http"

	"github.com/davidp57/process-monitor/internal/control"
	"github.com/davidp57/process-monitor/internal/i18n"
	"github.com/davidp57/process-monitor/internal/logx"
)

// sessionView is a running session as the page shows it.
type sessionView struct {
	control.Status
	Name string `json:"name"`
}

func (s *Server) sessionRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/sessions", func(w http.ResponseWriter, r *http.Request) {
		list, err := control.List()
		if err != nil {
			fail(w, err)
			return
		}
		out := []sessionView{}
		for _, c := range list {
			if st, err := c.State(); err == nil {
				out = append(out, sessionView{Status: st, Name: c.Info.Name})
			}
		}
		writeJSON(w, out)
	})
	mux.HandleFunc("POST /api/sessions/command", func(w http.ResponseWriter, r *http.Request) {
		var req struct{ Name, Command string }
		if err := read(r, &req); err != nil {
			fail(w, err)
			return
		}
		c, err := find(req.Name)
		if err == nil {
			err = c.Command(req.Command)
		}
		if err != nil {
			fail(w, err)
			return
		}
		writeJSON(w, "ok")
	})
	mux.HandleFunc("GET /api/sessions/log", func(w http.ResponseWriter, r *http.Request) {
		c, err := find(r.URL.Query().Get("name"))
		if err != nil {
			fail(w, err)
			return
		}
		lines, err := c.Log()
		if err != nil {
			fail(w, err)
			return
		}
		writeJSON(w, lines)
	})
	mux.HandleFunc("GET /api/sessions/events", func(w http.ResponseWriter, r *http.Request) {
		c, err := find(r.URL.Query().Get("name"))
		if err != nil {
			fail(w, err)
			return
		}
		send, ok := sse(w)
		if !ok {
			return
		}
		c.Events(r.Context(), func(l logx.Line) {
			data, _ := json.Marshal(l)
			send(data)
		})
	})
}

// find is the running session called name.
func find(name string) (*control.Client, error) {
	list, err := control.List()
	if err != nil {
		return nil, err
	}
	for _, c := range list {
		if c.Info.Name == name {
			return c, nil
		}
	}
	return nil, i18n.Errorf("gui.session_gone", name)
}
