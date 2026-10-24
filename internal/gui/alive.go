package gui

import (
	"net/http"
	"time"
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
	pages, last, wr := s.pages, s.lastPage, s.wizard
	s.mu.Unlock()
	if pages > 0 || now.Sub(last) < grace {
		return false
	}
	if wr == nil {
		return true
	}
	wr.mu.Lock()
	defer wr.mu.Unlock()
	return !wr.busy && wr.w.Next().ID == "" // finished: nothing left to put back
}
