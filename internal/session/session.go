// Package session runs one supervised session: lock, before hooks, launch,
// wait (with restarts for watch), after hooks.
package session

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/davidp57/process-monitor/internal/config"
	"github.com/davidp57/process-monitor/internal/hooks"
	"github.com/davidp57/process-monitor/internal/lock"
	"github.com/davidp57/process-monitor/internal/logx"
	"github.com/davidp57/process-monitor/internal/proc"

	"github.com/davidp57/process-monitor/internal/i18n"
)

// A run shorter than this counts as a crash loop: the restart delay doubles.
const stableRun = 10 * time.Second

// trackGrace is how long to keep looking for the tracked process once the
// launcher and everything it started have exited.
const trackGrace = 10 * time.Second

type Status string

const (
	Starting   Status = "starting"
	Running    Status = "running"
	Paused     Status = "paused"
	Restarting Status = "restarting"
	Stopping   Status = "stopping"
	Finished   Status = "finished"
	Detached   Status = "detached"
)

// State is a snapshot for display.
type State struct {
	Status    Status
	PID       int
	Restarts  int
	StartedAt time.Time
	Ready     string // "", or a catalog key: ready.waiting, ready.done, ready.timeout
	NextStart time.Time
	LastExit  string
	ExitCode  int // of the last run
	// Stopped: the last run ended because the user stopped it. Its exit
	// code is the one Windows gives a terminated process (1), not a failure.
	Stopped bool
}

type command int

const (
	cmdPause command = iota
	cmdResume
	cmdRestart
	cmdKill
	cmdDetach
)

// Session supervises one config. Create it with New, drive it with Run.
type Session struct {
	cfg     *config.Config
	log     *logx.Logger
	restart string // restart policy: never for run, the configured one for watch

	cmds chan command

	readyUsable bool // the stale ready file is gone: its appearance proves something
	hooks       *hooks.Runner

	mu             sync.Mutex
	state          State
	cur            proc.Process
	restartPending bool
}

// New prepares a session. restart is config.RestartNever for pm run.
func New(cfg *config.Config, log *logx.Logger, restart string) *Session {
	return &Session{cfg: cfg, log: log, restart: restart, cmds: make(chan command, 4),
		state: State{Status: Starting}}
}

func (s *Session) Config() *config.Config { return s.cfg }

func (s *Session) State() State {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := s.state
	if s.cur != nil {
		st.PID = s.cur.PID() // follows the switch from launcher to tracked process
	}
	return st
}

func (s *Session) update(fn func(*State)) {
	s.mu.Lock()
	fn(&s.state)
	s.mu.Unlock()
}

func (s *Session) send(c command) {
	if c == cmdRestart {
		// One restart at a time: a second press must not kill the new run.
		s.mu.Lock()
		pending := s.restartPending
		s.restartPending = true
		s.mu.Unlock()
		if pending {
			return
		}
	}
	select {
	case s.cmds <- c:
	default: // a command is already pending; drop repeated key presses
		if c == cmdRestart {
			s.restartHandled() // dropped: a later press must still work
		}
	}
}

// ErrCancelled ends a session the user cancelled before anything was
// launched: not a failure, but not a success either (pm run exits 1).
var ErrCancelled error = cancelled{}

// cancelled translates its message when it is read, not when pm starts:
// the language is chosen after package initialisation.
type cancelled struct{}

func (cancelled) Error() string { return i18n.T("session.cancelled") }

func (s *Session) restartHandled() {
	s.mu.Lock()
	s.restartPending = false
	s.mu.Unlock()
}

func (s *Session) Pause()   { s.send(cmdPause) }
func (s *Session) Resume()  { s.send(cmdResume) }
func (s *Session) Restart() { s.send(cmdRestart) }

// Kill stops the program, runs the after hooks and ends the session.
func (s *Session) Kill() { s.send(cmdKill) }

// Detach stops supervising and leaves the program running; after hooks are skipped.
func (s *Session) Detach() { s.send(cmdDetach) }

// Run blocks until the session ends.
func (s *Session) Run(ctx context.Context) error {
	c := s.cfg
	owner := c.Name
	if c.Profile != "" {
		owner += " / profil " + c.Profile
	}
	// One lock for the session name, one per live file it writes: two
	// configs that share a live file can never run together either.
	names := []string{c.Name}
	for _, p := range c.Sync {
		names = append(names, "live-"+liveKey(p.Target))
	}
	for _, n := range names {
		lk, err := lock.Acquire(n, owner)
		if err != nil {
			var busy *lock.Busy
			if errors.As(err, &busy) {
				return i18n.Errorf("session.busy", busy)
			}
			return i18n.Errorf("session.lock_failed", err)
		}
		defer lk.Release()
	}

	if err := s.handleAlreadyRunning(); err != nil {
		return err
	}

	h := &hooks.Runner{Log: s.log, Backup: c.BackupEnabled(), Timeout: c.Hooks.Timeout.D(), Dir: filepath.Dir(c.Path)}
	s.hooks = h
	undo, err := h.Before(ctx, c)
	if err != nil {
		s.update(func(st *State) { st.Status = Finished })
		if ctx.Err() != nil {
			return ErrCancelled
		}
		return i18n.Errorf("session.before_failed", c.Command, err)
	}
	// Stopped or detached while the files were being copied: nothing runs
	// yet, so there is nothing to detach from. Cancel instead of launching
	// a program only to leave it without its copy-back.
	if s.cancelledBeforeLaunch(ctx) {
		undo()
		s.update(func(st *State) { st.Status = Finished })
		return ErrCancelled
	}

	runErr, detached := s.loop(ctx)

	if detached {
		s.update(func(st *State) { st.Status = Detached })
		if len(c.Sync) > 0 || len(c.Hooks.After) > 0 {
			s.log.Warn(i18n.T("session.detached", c.ProcessName()))
		}
		return runErr
	}
	s.update(func(st *State) { st.Status = Stopping })
	// Last safety net: never copy back while the program runs unfollowed,
	// nor when that cannot be checked.
	if c.Track != "" {
		pids, err := proc.FindByName(c.ProcessName())
		if err != nil || len(pids) > 0 {
			s.update(func(st *State) { st.Status = Finished })
			why := i18n.T("session.unfollowed", pids)
			if err != nil {
				why = i18n.T("session.unsearchable", err)
			}
			return errors.Join(runErr, i18n.Errorf("session.safety_net", c.ProcessName(), why))
		}
	}
	afterErr := h.After(context.WithoutCancel(ctx), c)
	s.update(func(st *State) { st.Status = Finished })
	if afterErr != nil {
		afterErr = i18n.Errorf("session.after_failed", afterErr)
	}
	return errors.Join(runErr, afterErr)
}

// handleAlreadyRunning checks the tracked process and the launcher: a
// launcher busy updating the game counts as a running game.
func (s *Session) handleAlreadyRunning() error {
	// Launcher first: killed after its target, it could start it again.
	names := []string{s.cfg.LauncherName()}
	if target := s.cfg.ProcessName(); !strings.EqualFold(target, names[0]) {
		names = append(names, target)
	}
	if err := s.checkRunning(names); err != nil || s.cfg.IfRunning == config.IfRunningAbort {
		return err
	}
	// Killed: make sure nothing came back in between.
	for _, name := range names {
		if pids, err := proc.FindByName(name); err != nil || len(pids) > 0 {
			return i18n.Errorf("session.survived_kill", name, pids, err)
		}
	}
	return nil
}

func (s *Session) checkRunning(names []string) error {
	for _, name := range names {
		pids, err := proc.FindByName(name)
		if err != nil {
			return i18n.Errorf("session.search_failed", name, err)
		}
		if len(pids) == 0 {
			continue
		}
		if s.cfg.IfRunning == config.IfRunningAbort {
			return i18n.Errorf("session.already_running", name, pids)
		}
		n, err := proc.KillByName(name, 10*time.Second)
		s.log.Info(i18n.T("session.killed_existing", n, name))
		if err != nil {
			return i18n.Errorf("session.kill_existing_failed", name, err)
		}
	}
	return nil
}

// cancelledBeforeLaunch reports whether the user asked to stop or detach
// before the first launch. Other pending commands (pause, restart) mean
// nothing yet and are dropped.
func (s *Session) cancelledBeforeLaunch(ctx context.Context) bool {
	if ctx.Err() != nil {
		return true
	}
	for {
		select {
		case cmd := <-s.cmds:
			if cmd == cmdKill || cmd == cmdDetach {
				return true
			}
		default:
			return false
		}
	}
}

// killOnExit stops the processes listed in kill_on_exit, wherever they run:
// what the game started for itself (a radio client) has no use once it is gone.
func (s *Session) killOnExit() {
	for _, name := range s.cfg.KillOnExit {
		n, err := proc.KillByName(name, 10*time.Second)
		if err != nil {
			s.log.Error(i18n.T("session.kill_on_exit_failed", name, err))
		} else if n > 0 {
			s.log.Info(i18n.T("session.kill_on_exit", n, name))
		}
	}
}

// liveKey names the lock of a live file, the same for any spelling of its path.
func liveKey(path string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(filepath.Clean(path))))
	return hex.EncodeToString(sum[:8])
}

func (s *Session) env() []string {
	if len(s.cfg.Env) == 0 {
		return nil
	}
	env := os.Environ()
	for k, v := range s.cfg.Env {
		env = append(env, k+"="+v)
	}
	return env
}

func (s *Session) start() (proc.Process, error) {
	c := s.cfg
	s.readyUsable = c.Ready != nil
	if c.Ready != nil {
		// A ready file left by a crash would make the program look ready at once.
		if err := os.Remove(c.Ready.File); err == nil {
			s.log.Info(i18n.T("session.stale_ready_removed", c.Ready.File))
		} else if !errors.Is(err, os.ErrNotExist) {
			s.readyUsable = false
			s.log.Warn(i18n.T("session.stale_ready_kept", err))
		}
	}
	s.log.Info(i18n.T("session.launching", c.Command, strings.Join(c.Args, " ")))
	return proc.Start(proc.Spec{
		Command:    c.Command,
		Args:       c.Args,
		Dir:        c.Workdir,
		Env:        s.env(),
		Capture:    c.Capture,
		Track:      c.Track,
		TrackGrace: trackGrace,
		Output:     s.log.Output,
		Event:      func(m string) { s.log.Infof("%s", m) },
	})
}

// watchReady reports when the ready file appears, or that it never did.
func (s *Session) watchReady(done <-chan struct{}) {
	r := s.cfg.Ready
	if r == nil || !s.readyUsable {
		return
	}
	s.update(func(st *State) { st.Ready = "ready.waiting" })
	deadline := time.Now().Add(r.Timeout.D())
	t := time.NewTicker(250 * time.Millisecond)
	defer t.Stop()
	for {
		select {
		case <-done:
			return
		case <-t.C:
		}
		if _, err := os.Stat(r.File); err == nil {
			s.update(func(st *State) { st.Ready = "ready.done" })
			s.log.Info(i18n.T("session.ready", r.File))
			return
		}
		if time.Now().After(deadline) {
			s.update(func(st *State) { st.Ready = "ready.timeout" })
			s.log.Warn(i18n.T("session.ready_timeout", r.File, r.Timeout.D()))
			return
		}
	}
}

// loop runs the program until the policy says stop. It returns detached=true
// when the user left the program running.
func (s *Session) loop(ctx context.Context) (err error, detached bool) {
	c := s.cfg
	delay := c.Watch.Delay.D()
	quick := 0 // consecutive quick crashes, for the backoff
	for crashes, first := 0, true; ; first = false {
		if !first && len(c.Hooks.Before) > 0 {
			s.log.Info(i18n.T("session.replay_before"))
			if err := s.hooks.BeforeRestart(ctx, c); err != nil {
				s.log.Warn(i18n.T("session.replay_failed", err))
			}
		}
		p, startErr := s.start()
		begin := time.Now()
		s.restartHandled() // a new run exists: a new press may restart it
		var res proc.Result
		killed, restartAsked := false, false

		if startErr != nil {
			s.log.Error(i18n.T("session.launch_failed", startErr))
			res = proc.Result{ExitCode: -1, Err: startErr}
		} else {
			s.update(func(st *State) {
				st.Status, st.StartedAt, st.Ready, st.NextStart = Running, begin, "", time.Time{}
			})
			s.mu.Lock()
			s.cur = p
			s.mu.Unlock()
			stopReady := make(chan struct{})
			go s.watchReady(stopReady)
			res, killed, restartAsked, detached = s.supervise(ctx, p)
			close(stopReady)
			s.mu.Lock()
			s.cur = nil
			s.mu.Unlock()
			if detached {
				p.Close()
				return nil, true
			}
			if s.restart != config.RestartNever {
				_ = p.Kill() // leftovers of this run must not pile up with the next one
			}
			p.Close()
			s.killOnExit()
		}

		ran := time.Since(begin).Round(time.Second)
		summary := i18n.T("session.exit_summary", res.ExitCode, ran)
		if res.Err != nil {
			summary = res.Err.Error()
		}
		s.update(func(st *State) { st.LastExit, st.ExitCode, st.Stopped = summary, res.ExitCode, killed })
		switch {
		case res.Err != nil:
			s.log.Error(i18n.T("session.ended_error", summary))
		case killed:
			s.log.Info(i18n.T("session.stopped_on_request", c.ProcessName()))
		default:
			s.log.Info(i18n.T("session.program_exited", c.ProcessName(), summary))
		}

		switch {
		case killed:
			return nil, false
		case restartAsked: // asked by the user: no delay, not counted against max_restarts
			s.update(func(st *State) { st.Restarts++ })
			delay, quick = c.Watch.Delay.D(), 0
			continue
		case s.restart == config.RestartNever:
			return res.Err, false
		case s.restart == config.RestartOnFailure && res.ExitCode == 0 && res.Err == nil:
			return nil, false
		case c.Watch.MaxRestarts > 0 && crashes >= c.Watch.MaxRestarts:
			return i18n.Errorf("session.gave_up", c.Watch.MaxRestarts), false
		}
		crashes++

		// The first quick stop waits the base delay, each following one twice as long.
		if time.Since(begin) < stableRun {
			if quick > 0 {
				delay = min(delay*2, c.Watch.MaxDelay.D())
			}
			quick++
		} else {
			delay, quick = c.Watch.Delay.D(), 0
		}
		next := time.Now().Add(delay)
		s.update(func(st *State) { st.Status, st.NextStart = Restarting, next })
		s.log.Info(i18n.T("session.restart_in", delay))
		// Nothing runs during the delay: detaching then is just ending, after hooks included.
		if cmd := s.pause(ctx, delay); cmd == cmdKill || cmd == cmdDetach {
			s.log.Info(i18n.T("session.restart_cancelled"))
			return nil, false
		}
		s.update(func(st *State) { st.Restarts++ })
	}
}

// pause waits between restarts; it returns the command that cut it short.
func (s *Session) pause(ctx context.Context, d time.Duration) command {
	t := time.NewTimer(d)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			return cmdRestart
		case <-ctx.Done():
			return cmdDetach
		case cmd := <-s.cmds:
			if cmd == cmdKill || cmd == cmdDetach || cmd == cmdRestart {
				return cmd
			}
		}
	}
}

func (s *Session) supervise(ctx context.Context, p proc.Process) (res proc.Result, killed, restart, detached bool) {
	for {
		select {
		case <-p.Done():
			return p.Result(), killed, restart, false
		case <-ctx.Done():
			if ended(p) {
				continue // it just ended: report that, not a detach
			}
			return proc.Result{}, false, false, true
		case cmd := <-s.cmds:
			switch cmd {
			case cmdPause:
				if err := p.Suspend(); err != nil {
					s.log.Error(i18n.T("session.pause_failed", err))
				} else {
					s.update(func(st *State) { st.Status = Paused })
					s.log.Info(i18n.T("session.paused", s.cfg.ProcessName()))
				}
			case cmdResume:
				if err := p.Resume(); err != nil {
					s.log.Error(i18n.T("session.resume_failed", err))
				} else {
					s.update(func(st *State) { st.Status = Running })
					s.log.Info(i18n.T("session.resumed", s.cfg.ProcessName()))
				}
			case cmdRestart, cmdKill:
				if s.State().Status == Paused {
					_ = p.Resume() // a suspended process may not die cleanly
				}
				s.log.Info(i18n.T("session.stop_asked", s.cfg.ProcessName()))
				s.update(func(st *State) { st.Status = Stopping })
				if err := p.Kill(); err != nil {
					s.log.Error(i18n.T("session.stop_failed", err))
				}
				killed, restart = cmd == cmdKill, cmd == cmdRestart
			case cmdDetach:
				if ended(p) {
					continue
				}
				return proc.Result{}, false, false, true
			}
		}
	}
}

// PID of the followed process, 0 when none.
func (s *Session) PID() int { return s.State().PID }

func ended(p proc.Process) bool {
	select {
	case <-p.Done():
		return true
	default:
		return false
	}
}
