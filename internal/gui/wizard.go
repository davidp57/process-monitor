package gui

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/davidp57/process-monitor/internal/config"
	"github.com/davidp57/process-monitor/internal/i18n"
	"github.com/davidp57/process-monitor/internal/logx"
	"github.com/davidp57/process-monitor/internal/registry"
	"github.com/davidp57/process-monitor/internal/shortcut"
	"github.com/davidp57/process-monitor/internal/textdiff"
	"github.com/davidp57/process-monitor/internal/wizard"
)

// wizardRun is the wizard the page drives: the engine of pm new, and the
// runner of its actions in this process. One at a time.
//
// mu guards the engine too: every request that reads or changes it holds
// mu throughout, and none changes it while an action runs (busy), whose
// goroutine only reads it until it records its result under mu.
type wizardRun struct {
	w   *wizard.Wizard
	r   *wizard.Runner
	log *logx.Logger

	mu       sync.Mutex
	busy     bool            // an action runs
	err      string          // the last action's failure
	running  *wizard.Running // the game found running, to offer stopping it
	written  []shortcut.Written
	startMnu bool
}

// wizardView is what the page shows of the wizard.
type wizardView struct {
	Active   bool               `json:"active"`
	Step     *wizard.Step       `json:"step,omitempty"`
	Busy     bool               `json:"busy"`
	Error    string             `json:"error,omitempty"`
	Running  *wizard.Running    `json:"running,omitempty"`
	Done     bool               `json:"done"`
	Applied  bool               `json:"applied"`
	Config   string             `json:"config,omitempty"`
	Written  []shortcut.Written `json:"written,omitempty"`
	StartMnu bool               `json:"startMenu,omitempty"`
	Answers  []wizard.Answered  `json:"answers"`
}

func newWizard(path, self string) (*wizardRun, error) {
	log, err := logx.New("", "", 0, nil)
	if err != nil {
		return nil, err
	}
	docs, _ := documents()
	env := wizard.Env{Home: os.Getenv("USERPROFILE"), Documents: docs, Lang: i18n.Lang(), Now: time.Now,
		Exists:   func(p string) bool { _, err := os.Stat(p); return err == nil },
		Profiles: config.Profiles, Icons: wizard.FindIcons}
	w := wizard.New(env, path)
	return &wizardRun{w: w, log: log, r: &wizard.Runner{W: w, Log: log, PM: self}}, nil
}

func (wr *wizardRun) view() wizardView {
	wr.mu.Lock()
	defer wr.mu.Unlock()
	v := wizardView{Active: true, Busy: wr.busy, Error: wr.err, Running: wr.running,
		Applied: wr.w.Applied(), Written: wr.written, StartMnu: wr.startMnu, Answers: wr.w.Answers()}
	step := wr.w.Next()
	if step.Question == nil && step.Action == nil {
		v.Done = true
		if v.Applied {
			v.Config = wr.w.Plan().ConfigPath
		}
		return v
	}
	v.Step = &step
	return v
}

func (s *Server) current() *wizardRun {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.wizard
}

func (s *Server) wizardRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/wizard", func(w http.ResponseWriter, r *http.Request) {
		if wr := s.current(); wr != nil {
			writeJSON(w, wr.view())
			return
		}
		writeJSON(w, wizardView{})
	})
	mux.HandleFunc("POST /api/wizard/start", func(w http.ResponseWriter, r *http.Request) {
		var req struct{ Path string }
		read(r, &req)
		s.mu.Lock()
		if s.wizard != nil {
			s.mu.Unlock()
			fail(w, errors.New(i18n.T("gui.wizard_running")))
			return
		}
		wr, err := newWizard(req.Path, s.opt.Self)
		if err != nil {
			s.mu.Unlock()
			fail(w, err)
			return
		}
		s.wizard = wr
		s.mu.Unlock()
		writeJSON(w, wr.view())
	})
	// with runs fn on the wizard, unless an action is running.
	with := func(fn func(wr *wizardRun, r *http.Request) error) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			wr := s.current()
			if wr == nil {
				fail(w, errors.New(i18n.T("gui.wizard_none")))
				return
			}
			wr.mu.Lock()
			err := errors.New(i18n.T("gui.wizard_busy"))
			if !wr.busy {
				err = fn(wr, r)
			}
			wr.mu.Unlock()
			if err != nil {
				fail(w, err)
				return
			}
			writeJSON(w, wr.view())
		}
	}
	mux.HandleFunc("POST /api/wizard/answer", with(func(wr *wizardRun, r *http.Request) error {
		var req struct{ Values []string }
		if err := read(r, &req); err != nil {
			return err
		}
		if wr.w.Next().Question == nil {
			return errors.New(i18n.T("gui.wizard_busy"))
		}
		return wr.w.Answer(req.Values)
	}))
	mux.HandleFunc("POST /api/wizard/back", with(func(wr *wizardRun, r *http.Request) error {
		if !wr.w.Back() && wr.w.ActionsRan() {
			return errors.New(i18n.T("wizard.no_back"))
		}
		return nil
	}))
	mux.HandleFunc("POST /api/wizard/run", with(func(wr *wizardRun, r *http.Request) error {
		step := wr.w.Next()
		if step.Action == nil {
			return errors.New(i18n.T("gui.wizard_busy"))
		}
		wr.busy, wr.err, wr.running = true, "", nil
		go s.runAction(wr, *step.Action)
		return nil
	}))
	mux.HandleFunc("POST /api/wizard/stopgame", with(func(wr *wizardRun, r *http.Request) error {
		if wr.running == nil {
			return nil
		}
		if err := wr.r.StopGame(wr.running); err != nil {
			return err
		}
		wr.running, wr.err = nil, ""
		return nil
	}))
	mux.HandleFunc("POST /api/wizard/close", func(w http.ResponseWriter, r *http.Request) {
		wr := s.current()
		if wr == nil {
			writeJSON(w, wizardView{})
			return
		}
		wr.mu.Lock()
		err := errors.New(i18n.T("gui.wizard_busy"))
		if !wr.busy {
			err = nil
			// Unfinished: the original files go back, as Ctrl+C in pm new.
			if !wr.w.Applied() && wr.w.Next().ID != "" {
				err = wr.r.Abandon()
			}
		}
		wr.mu.Unlock()
		if err != nil {
			fail(w, err)
			return
		}
		s.mu.Lock()
		s.wizard = nil
		s.mu.Unlock()
		writeJSON(w, wizardView{})
	})
	mux.HandleFunc("GET /api/wizard/events", func(w http.ResponseWriter, r *http.Request) {
		wr := s.current()
		if wr == nil {
			fail(w, errors.New(i18n.T("gui.wizard_none")))
			return
		}
		send, ok := sse(w)
		if !ok {
			return
		}
		lines := make(chan logx.Line, 256)
		_, cancel := wr.log.Subscribe(func(l logx.Line) {
			select {
			case lines <- l:
			default:
			}
		})
		defer cancel()
		for {
			select {
			case <-r.Context().Done():
				return
			case l := <-lines:
				data, _ := json.Marshal(l)
				send(data)
			}
		}
	})
	mux.HandleFunc("GET /api/wizard/diff", func(w http.ResponseWriter, r *http.Request) {
		wr := s.current()
		if wr == nil {
			fail(w, errors.New(i18n.T("gui.wizard_none")))
			return
		}
		wr.mu.Lock()
		step := wr.w.Next()
		profile, ok := strings.CutPrefix(step.ID, "review:")
		var d fileDiff
		if ok {
			d = diffOf(wr.w, profile, r.URL.Query().Get("file"))
		}
		wr.mu.Unlock()
		if !ok {
			fail(w, errors.New(i18n.T("gui.wizard_busy")))
			return
		}
		writeJSON(w, d)
	})
}

// runAction runs the wizard's current action in the background; the page
// follows it through /api/wizard and the lines through /api/wizard/events.
func (s *Server) runAction(wr *wizardRun, a wizard.Action) {
	res, err := wr.r.Do(context.Background(), a)
	wr.mu.Lock()
	defer wr.mu.Unlock()
	wr.busy = false
	if err != nil {
		wr.err = err.Error()
		var running *wizard.Running
		if errors.As(err, &running) {
			wr.running = running
		}
		return
	}
	wr.w.Done(res)
	if links, ok := res.([]shortcut.Link); ok {
		wr.written, wr.startMnu = shortcut.WriteAll(links)
		registry.Remember(wr.w.Plan().ConfigPath)
	}
}

// fileDiff is what the report shows of a file a profile changed.
type fileDiff struct {
	Binary bool          `json:"binary"`
	Before int64         `json:"before"` // sizes; -1 when the file did not exist
	After  int64         `json:"after"`
	Ops    []textdiff.Op `json:"ops,omitempty"`
}

// textLimit is the size above which a file is shown by its size only.
const textLimit = 1 << 20

func diffOf(w *wizard.Wizard, profile, f string) fileDiff {
	beforePath, afterPath := w.Versions(profile, f)
	load := func(p string) ([]byte, int64) {
		if p == "" {
			return nil, -1
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return nil, -1
		}
		return data, int64(len(data))
	}
	a, sa := load(beforePath)
	b, sb := load(afterPath)
	d := fileDiff{Before: sa, After: sb}
	if textdiff.Binary(a, textLimit) || textdiff.Binary(b, textLimit) {
		d.Binary = true
		return d
	}
	d.Ops = textdiff.Hunks(textdiff.Lines(string(a), string(b)), 3)
	return d
}
