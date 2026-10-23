// Package proc starts a program and waits for the end of what it started.
//
// On Windows the program is placed in a Job Object, so every process it
// spawns (a launcher handing over to the real game, for instance) stays in
// view. When Track is set, the session follows that process by handle; it
// ends when it exits, wherever it ended up.
package proc

import (
	"bufio"
	"io"
	"strings"
	"time"
)

// Spec describes what to start.
type Spec struct {
	Command string
	Args    []string
	Dir     string
	Env     []string // full environment; nil inherits
	Capture bool     // capture stdout/stderr into Output
	Track   string   // process name to follow instead of the launched one
	// TrackGrace is how long to keep looking for Track once everything
	// else has exited.
	TrackGrace time.Duration
	Output     func(line string)
	// Event reports notable steps (tracked process found, fallbacks).
	Event func(msg string)
}

// Result is how a session ended.
type Result struct {
	PID      int
	ExitCode int
	Err      error // the session could not be followed to its end
}

// Process is a running session.
type Process interface {
	// PID of the followed process: the tracked one once found, else the launched one.
	PID() int
	// Done is closed when the session has ended; Result is then valid.
	Done() <-chan struct{}
	Result() Result
	Kill() error
	Suspend() error
	Resume() error
	// Close releases handles without touching the processes.
	Close()
}

func pump(r io.ReadCloser, out func(string)) {
	defer r.Close()
	s := bufio.NewScanner(r)
	s.Buffer(make([]byte, 64*1024), 1024*1024)
	for s.Scan() {
		out(strings.TrimRight(s.Text(), "\r"))
	}
}

func (s Spec) event(format string) {
	if s.Event != nil {
		s.Event(format)
	}
}
