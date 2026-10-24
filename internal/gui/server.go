// Package gui is pm's interface: a page served on 127.0.0.1 and opened in
// the browser. It lists and drives the sessions running (through their
// control channel), lists the configs (remembered and found in folders),
// launches them, makes their shortcuts, edits them, and hosts the wizard.
package gui

import (
	"crypto/subtle"
	"embed"
	"encoding/json"
	"io/fs"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/davidp57/process-monitor/internal/i18n"
	"github.com/davidp57/process-monitor/internal/registry"
)

//go:embed static
var static embed.FS

// Options are what the server needs from its host.
type Options struct {
	Port  int    // where it listens, for the Host check
	Token string // the secret in the URL first opened
	// Self is the pm.exe that launches configs and that shortcuts start.
	Self string
	// Launch starts a session as its own process; tests replace it.
	Launch func(args []string) error
	// Defaults lists the folders looked into without being given; tests
	// replace it, so they never see the machine's own configs.
	Defaults func() []string
}

// Server serves the interface.
type Server struct {
	opt Options

	mu       sync.Mutex
	wizard   *wizardRun
	pages    int       // pages open (see alive)
	lastPage time.Time // when the last one went away, or the start
}

// New prepares the interface.
func New(opt Options) *Server {
	if opt.Launch == nil {
		opt.Launch = launch
	}
	if opt.Defaults == nil {
		self := opt.Self
		opt.Defaults = func() []string { return defaultFolders(self) }
	}
	// The first page gets longer: the browser may take a while to start.
	return &Server{opt: opt, lastPage: time.Now().Add(30 * time.Second)}
}

// cookie carries the token once the page has been opened with it.
const cookie = "pm_token"

// Handler is the whole interface, guarded: only 127.0.0.1 or localhost on
// the server's port, only with the token (in the first URL, then in a
// cookie no other site gets), and writes only with a header a form posted
// from another site cannot carry.
func (s *Server) Handler() http.Handler {
	hosts := map[string]bool{"127.0.0.1:" + strconv.Itoa(s.opt.Port): true, "localhost:" + strconv.Itoa(s.opt.Port): true}
	mux := s.routes()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !hosts[r.Host] {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if t := r.URL.Query().Get("token"); r.Method == http.MethodGet && r.URL.Path == "/" && t != "" {
			if subtle.ConstantTimeCompare([]byte(t), []byte(s.opt.Token)) != 1 {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			http.SetCookie(w, &http.Cookie{Name: cookie, Value: s.opt.Token, Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode})
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		c, err := r.Cookie(cookie)
		if err != nil || subtle.ConstantTimeCompare([]byte(c.Value), []byte(s.opt.Token)) != 1 {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if r.Method != http.MethodGet && r.Header.Get("X-PM") != "1" {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		mux.ServeHTTP(w, r)
	})
}

func (s *Server) routes() *http.ServeMux {
	mux := http.NewServeMux()
	page, _ := fs.Sub(static, "static")
	mux.Handle("GET /", http.FileServer(http.FS(page)))
	mux.HandleFunc("GET /api/i18n", func(w http.ResponseWriter, r *http.Request) {
		messages := i18n.Prefixed("gui.")
		for _, prefix := range []string{"status.", "ready."} { // a session's state, as the icon says it
			for k, v := range i18n.Prefixed(prefix) {
				messages[k] = v
			}
		}
		writeJSON(w, map[string]any{"lang": i18n.Lang(), "setting": registry.Lang(), "messages": messages})
	})
	mux.HandleFunc("GET /api/ping", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, "pm") })
	mux.HandleFunc("GET /api/alive", s.alive)
	mux.HandleFunc("GET /api/icon", icon)
	mux.HandleFunc("POST /api/lang", func(w http.ResponseWriter, r *http.Request) {
		var req struct{ Lang string }
		if err := read(r, &req); err != nil {
			fail(w, err)
			return
		}
		if req.Lang != "" && !i18n.Supported(req.Lang) {
			fail(w, i18n.Errorf("config.bad_lang", req.Lang))
			return
		}
		if err := registry.Change(func(reg *registry.Registry) { reg.Lang = req.Lang }); err != nil {
			fail(w, err)
			return
		}
		lang := req.Lang
		if lang == "" {
			lang = i18n.Resolve("", "", "", i18n.SystemLanguage)
		}
		i18n.Set(lang)
		writeJSON(w, "ok")
	})
	s.sessionRoutes(mux)
	s.configRoutes(mux)
	s.wizardRoutes(mux)
	return mux
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

// fail answers an error in words for the page.
func fail(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}

// read decodes a JSON body into v.
func read(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}

// sse prepares a Server-Sent Events answer.
func sse(w http.ResponseWriter) (func(data []byte), bool) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, false
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()
	return func(data []byte) {
		w.Write([]byte("data: "))
		w.Write(data)
		w.Write([]byte("\n\n"))
		flusher.Flush()
	}, true
}
