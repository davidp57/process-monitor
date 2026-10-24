// Package control is a session's control channel: a small HTTP server on
// 127.0.0.1 that every pm run or pm watch opens, so the interface can see
// the sessions running and drive them as the notification icon does
// (stop, pause, resume, restart, detach), whatever started them.
//
// A session writes how to reach it (port and token) next to its lock once
// it holds the lock, and removes it when it ends.
package control

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/davidp57/process-monitor/internal/i18n"
	"github.com/davidp57/process-monitor/internal/lock"
	"github.com/davidp57/process-monitor/internal/logx"
	"github.com/davidp57/process-monitor/internal/session"
)

// Info is what a session writes next to its lock.
type Info struct {
	Name    string `json:"name"`
	Profile string `json:"profile,omitempty"`
	Config  string `json:"config"`
	Mode    string `json:"mode"` // run or watch
	PID     int    `json:"pid"`  // pm's own process
	Port    int    `json:"port"`
	Token   string `json:"token,omitempty"`
	Log     string `json:"log,omitempty"` // the log file
}

// Status is a session's answer to /state.
type Status struct {
	Info  Info          `json:"info"`
	State session.State `json:"state"`
}

// Commands a session accepts on /command.
const (
	Stop    = "stop"
	Pause   = "pause"
	Resume  = "resume"
	Restart = "restart"
	Detach  = "detach"
)

// TokenHeader carries the session's token on every request.
const TokenHeader = "X-PM-Token"

// Server is a session's control channel.
type Server struct {
	info Info
	s    *session.Session
	log  *logx.Logger
	http *http.Server

	mu    sync.Mutex
	path  string // the info file, once written
	data  []byte // what was written in it
	ended bool
}

// Serve opens the control channel of s and arranges for its address to be
// written once s holds its lock. Close it when the session ends.
func Serve(s *session.Session, log *logx.Logger, info Info) (*Server, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	token, err := Token()
	if err != nil {
		ln.Close()
		return nil, err
	}
	info.Port, info.Token, info.PID, info.Log = ln.Addr().(*net.TCPAddr).Port, token, os.Getpid(), log.Path
	srv := &Server{info: info, s: s, log: log}
	srv.http = &http.Server{Handler: Guard(info.Port, token, srv.routes()), ReadHeaderTimeout: 5 * time.Second}
	go srv.http.Serve(ln)
	s.Locked = srv.publish
	return srv, nil
}

// Token is a random secret for a local server.
func Token() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// Guard lets through only requests made to 127.0.0.1 (or localhost) on
// port, carrying token: another program, or a web page that resolves its
// own name to 127.0.0.1, cannot reach what it guards.
func Guard(port int, token string, next http.Handler) http.Handler {
	hosts := map[string]bool{"127.0.0.1:" + strconv.Itoa(port): true, "localhost:" + strconv.Itoa(port): true}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !hosts[r.Host] || subtle.ConstantTimeCompare([]byte(r.Header.Get(TokenHeader)), []byte(token)) != 1 {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// publish writes the info file: the session holds its lock, it is the
// only one with this name.
func (srv *Server) publish() {
	path, err := lock.InfoPath(srv.info.Name)
	if err != nil {
		return
	}
	data, _ := json.Marshal(srv.info)
	srv.mu.Lock()
	defer srv.mu.Unlock()
	if srv.ended {
		return
	}
	if os.WriteFile(path, data, 0o600) == nil {
		srv.path, srv.data = path, data
	}
}

// Close stops the channel and removes the info file, unless another
// session of the same name has written its own there since: this one may
// close long after its lock was released, waiting for a key press.
func (srv *Server) Close() {
	srv.mu.Lock()
	srv.ended = true
	if srv.path != "" {
		if now, err := os.ReadFile(srv.path); err == nil && bytes.Equal(now, srv.data) {
			os.Remove(srv.path)
		}
	}
	srv.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	srv.http.Shutdown(ctx)
}

func (srv *Server) routes() http.Handler {
	mux := http.NewServeMux()
	public := srv.info
	public.Token = ""
	mux.HandleFunc("GET /state", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, Status{Info: public, State: srv.s.State()})
	})
	mux.HandleFunc("GET /log", func(w http.ResponseWriter, r *http.Request) {
		recent, cancel := srv.log.Subscribe(func(logx.Line) {})
		cancel()
		writeJSON(w, recent)
	})
	mux.HandleFunc("GET /events", srv.events)
	mux.HandleFunc("POST /command", func(w http.ResponseWriter, r *http.Request) {
		var c struct{ Command string }
		if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		switch c.Command {
		case Stop:
			srv.s.Kill()
		case Pause:
			srv.s.Pause()
		case Resume:
			srv.s.Resume()
		case Restart:
			srv.s.Restart()
		case Detach:
			srv.s.Detach()
		default:
			http.Error(w, "unknown command", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	return mux
}

// events streams the log lines to come as Server-Sent Events, one JSON
// line per event. A slow reader loses lines rather than slowing the session.
func (srv *Server) events(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	lines := make(chan logx.Line, 256)
	_, cancel := srv.log.Subscribe(func(l logx.Line) {
		select {
		case lines <- l:
		default:
		}
	})
	defer cancel()
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()
	for {
		select {
		case <-r.Context().Done():
			return
		case l := <-lines:
			data, _ := json.Marshal(l)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

// ErrUnreachable is returned by a client whose session does not answer.
var ErrUnreachable error = unreachable{}

// unreachable translates its message when it is read: the language is
// chosen after package initialisation.
type unreachable struct{}

func (unreachable) Error() string { return i18n.T("control.unreachable") }
