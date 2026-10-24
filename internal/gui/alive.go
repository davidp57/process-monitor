package gui

import (
	"net/http"
	"time"

	"github.com/davidp57/process-monitor/internal/i18n"
)

// The interface stops by itself once no page has been open for a while,
// unless a wizard is under way (its game runs in this process, and its
// original files would be left set aside). Each open page holds a
// connection to /api/alive; the count of those is what "open" means.

// alive holds one page's connection open until the page goes away.
func (s *Server) alive(w http.ResponseWriter, r *http.Request) {
	send, ok := sse(w)
	if !ok {
		return
	}
	s.mu.Lock()
	s.pages++
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.pages--
		s.lastPage = time.Now()
		s.mu.Unlock()
	}()
	tick := time.NewTicker(15 * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-tick.C:
			send([]byte(`"ok"`)) // a dead page shows by the write failing
		}
	}
}

// Idle reports whether the interface may stop at now: no page open for
// grace, and no wizard under way.
func (s *Server) Idle(now time.Time, grace time.Duration) bool {
	s.mu.Lock()
	pages, last := s.pages, s.lastPage
	s.mu.Unlock()
	if pages > 0 || now.Sub(last) < grace {
		return false
	}
	return !s.wizardUnderWay()
}

// wizardUnderWay reports a wizard started and not finished: stopping now
// would leave the game's original files set aside.
func (s *Server) wizardUnderWay() bool {
	s.mu.Lock()
	wr := s.wizard
	s.mu.Unlock()
	if wr == nil {
		return false
	}
	wr.mu.Lock()
	defer wr.mu.Unlock()
	return wr.busy || wr.w.Next().ID != "" // finished: nothing left to put back
}

// quit stops this interface for another pm.exe to take its place, unless a
// wizard is under way.
func (s *Server) quit(w http.ResponseWriter, r *http.Request) {
	if s.wizardUnderWay() || s.opt.Quit == nil {
		fail(w, i18n.Errorf("gui.quit_refused"))
		return
	}
	writeJSON(w, "ok")
	go s.opt.Quit()
}
