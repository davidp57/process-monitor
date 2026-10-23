package session

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/davidp57/process-monitor/internal/config"
	"github.com/davidp57/process-monitor/internal/lock"
	"github.com/davidp57/process-monitor/internal/logx"
)

// The test binary doubles as the supervised program, driven by PM_HELPER:
//
//	append:FILE      append "+game" to FILE, exit 0
//	exit:N           exit with code N
//	sleep:MS         sleep MS milliseconds, exit 0
func TestMain(m *testing.M) {
	mode, arg, _ := strings.Cut(os.Getenv("PM_HELPER"), ":")
	switch mode {
	case "":
		os.Exit(m.Run())
	case "append":
		f, err := os.OpenFile(arg, os.O_APPEND|os.O_WRONLY, 0)
		if err != nil {
			os.Exit(90)
		}
		f.WriteString("+game")
		f.Close()
		os.Exit(0)
	case "exit":
		n, _ := strconv.Atoi(arg)
		os.Exit(n)
	case "sleep":
		ms, _ := strconv.Atoi(arg)
		time.Sleep(time.Duration(ms) * time.Millisecond)
		os.Exit(0)
	}
	os.Exit(98)
}

var seq int

// program copies the test binary under a unique name, so if_running checks
// do not see the test itself or another test's program.
func program(t *testing.T) string {
	t.Helper()
	seq++
	dst := filepath.Join(t.TempDir(), fmt.Sprintf("pm-session-%d-%d.exe", os.Getpid(), seq))
	self, _ := os.Executable()
	src, err := os.Open(self)
	if err != nil {
		t.Fatal(err)
	}
	defer src.Close()
	out, err := os.Create(dst)
	if err != nil {
		t.Fatal(err)
	}
	io.Copy(out, src)
	out.Close()
	return dst
}

func put(t *testing.T, p, s string) {
	t.Helper()
	os.MkdirAll(filepath.Dir(p), 0o755)
	if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
		t.Fatal(err)
	}
}

func read(t *testing.T, p string) string {
	t.Helper()
	b, _ := os.ReadFile(p)
	return string(b)
}

func newConfig(t *testing.T, helper string) *config.Config {
	exe := program(t)
	return &config.Config{
		Name:      filepath.Base(exe),
		Command:   exe,
		Env:       map[string]string{"PM_HELPER": helper},
		IfRunning: config.IfRunningAbort,
		Watch: config.Watch{Restart: config.RestartAlways,
			Delay: config.Duration(50 * time.Millisecond), MaxDelay: config.Duration(200 * time.Millisecond)},
	}
}

func logger(t *testing.T) (*logx.Logger, *[]string) {
	var lines []string
	l, err := logx.New("", "", 0, func(line logx.Line) { lines = append(lines, line.String()) })
	if err != nil {
		t.Fatal(err)
	}
	return l, &lines
}

func runWithin(t *testing.T, s *Session, d time.Duration) error {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- s.Run(context.Background()) }()
	select {
	case err := <-done:
		return err
	case <-time.After(d):
		s.Kill()
		t.Fatalf("session toujours en cours après %s", d)
		return nil
	}
}

func TestRunSyncsProfileInAndBack(t *testing.T) {
	d := t.TempDir()
	prof, live := filepath.Join(d, "prof", "options.lua"), filepath.Join(d, "game", "options.lua")
	put(t, prof, "profile")
	put(t, live, "other profile")
	c := newConfig(t, "append:"+live)
	c.Sync = []config.SyncPair{{Source: prof, Target: live}}
	log, lines := logger(t)

	if err := runWithin(t, New(c, log, config.RestartNever), 15*time.Second); err != nil {
		t.Fatalf("err = %v\n%s", err, strings.Join(*lines, "\n"))
	}
	if got := read(t, prof); got != "profile+game" {
		t.Errorf("profil après la session = %q", got)
	}
}

func TestBeforeFailureLaunchesNothing(t *testing.T) {
	d := t.TempDir()
	marker := filepath.Join(d, "launched")
	put(t, marker, "")
	c := newConfig(t, "append:"+marker)
	c.Sync = []config.SyncPair{{Source: filepath.Join(d, "missing"), Target: filepath.Join(d, "t")}}
	c.Hooks.After = []config.Action{{Delete: marker}}
	log, _ := logger(t)

	err := runWithin(t, New(c, log, config.RestartNever), 15*time.Second)
	if err == nil || !strings.Contains(err.Error(), "pas été lancé") {
		t.Errorf("err = %v", err)
	}
	if read(t, marker) != "" {
		t.Error("le programme a été lancé")
	}
	if _, err := os.Stat(marker); err != nil {
		t.Error("les actions de fin ont tourné alors que rien n'a été lancé")
	}
}

func TestAlreadyRunningAbortsBeforeTouchingAnything(t *testing.T) {
	d := t.TempDir()
	prof, live := filepath.Join(d, "prof.lua"), filepath.Join(d, "live.lua")
	put(t, prof, "VR")
	put(t, live, "DEV in use")
	c := newConfig(t, "sleep:10")
	c.Sync = []config.SyncPair{{Source: prof, Target: live}}

	other := exec.Command(c.Command)
	other.Env = append(os.Environ(), "PM_HELPER=sleep:20000")
	if err := other.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { other.Process.Kill(); other.Wait() }()
	time.Sleep(200 * time.Millisecond)

	log, _ := logger(t)
	err := runWithin(t, New(c, log, config.RestartNever), 15*time.Second)
	if err == nil || !strings.Contains(err.Error(), "tourne déjà") {
		t.Errorf("err = %v", err)
	}
	if read(t, live) != "DEV in use" || read(t, prof) != "VR" {
		t.Errorf("fichiers touchés : live=%q prof=%q", read(t, live), read(t, prof))
	}
}

func TestIfRunningKill(t *testing.T) {
	c := newConfig(t, "exit:0")
	c.IfRunning = config.IfRunningKill
	other := exec.Command(c.Command)
	other.Env = append(os.Environ(), "PM_HELPER=sleep:20000")
	if err := other.Start(); err != nil {
		t.Fatal(err)
	}
	defer other.Wait()
	time.Sleep(200 * time.Millisecond)
	log, _ := logger(t)
	if err := runWithin(t, New(c, log, config.RestartNever), 15*time.Second); err != nil {
		t.Fatal(err)
	}
}

func TestSecondSessionIsRefused(t *testing.T) {
	d := t.TempDir()
	prof, live := filepath.Join(d, "prof.lua"), filepath.Join(d, "live.lua")
	put(t, prof, "VR")
	put(t, live, "DEV in use")
	c := newConfig(t, "exit:0")
	c.Sync = []config.SyncPair{{Source: prof, Target: live}}
	held, err := lock.Acquire(c.Name, "profil DEV")
	if err != nil {
		t.Fatal(err)
	}
	defer held.Release()

	log, _ := logger(t)
	err = runWithin(t, New(c, log, config.RestartNever), 15*time.Second)
	if err == nil || !strings.Contains(err.Error(), "profil DEV") {
		t.Errorf("err = %v", err)
	}
	if read(t, live) != "DEV in use" {
		t.Error("le fichier live a été écrasé")
	}
}

func TestWatchRestartsUntilMax(t *testing.T) {
	d := t.TempDir()
	after := filepath.Join(d, "after")
	put(t, after, "")
	c := newConfig(t, "exit:3")
	c.Watch.MaxRestarts = 2
	c.Hooks.After = []config.Action{{Exec: []string{"cmd", "/c", "echo done>>" + after}}}
	log, _ := logger(t)
	s := New(c, log, config.RestartAlways)

	err := runWithin(t, s, 30*time.Second)
	if err == nil || !strings.Contains(err.Error(), "2 relances") {
		t.Errorf("err = %v", err)
	}
	if s.State().Restarts != 2 {
		t.Errorf("relances = %d", s.State().Restarts)
	}
	if n := strings.Count(read(t, after), "done"); n != 1 {
		t.Errorf("actions de fin exécutées %d fois", n)
	}
}

func TestWatchOnFailureStopsOnCleanExit(t *testing.T) {
	c := newConfig(t, "exit:0")
	c.Watch.MaxRestarts = 5
	log, _ := logger(t)
	s := New(c, log, config.RestartOnFailure)
	if err := runWithin(t, s, 15*time.Second); err != nil || s.State().Restarts != 0 {
		t.Errorf("err = %v, relances = %d", err, s.State().Restarts)
	}
}

// Between two restarts nothing runs: detaching then must still run the after hooks.
func TestDetachDuringRestartDelayRunsAfterHooks(t *testing.T) {
	d := t.TempDir()
	marker := filepath.Join(d, "marker")
	put(t, marker, "x")
	c := newConfig(t, "exit:1")
	c.Watch.Delay = config.Duration(5 * time.Second)
	c.Hooks.After = []config.Action{{Delete: marker}}
	log, _ := logger(t)
	s := New(c, log, config.RestartAlways)
	go func() {
		for s.State().Status != Restarting {
			time.Sleep(50 * time.Millisecond)
		}
		s.Detach()
	}()
	if err := runWithin(t, s, 15*time.Second); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(marker); err == nil {
		t.Error("les actions de fin n'ont pas tourné")
	}
}

func TestKillRunsAfterHooksDetachSkipsThem(t *testing.T) {
	for _, detach := range []bool{false, true} {
		d := t.TempDir()
		marker := filepath.Join(d, "marker")
		put(t, marker, "x")
		c := newConfig(t, "sleep:30000")
		c.Hooks.After = []config.Action{{Delete: marker}}
		log, _ := logger(t)
		s := New(c, log, config.RestartAlways)
		go func() {
			for s.State().PID == 0 {
				time.Sleep(50 * time.Millisecond)
			}
			if detach {
				s.Detach()
			} else {
				s.Kill()
			}
		}()
		if err := runWithin(t, s, 15*time.Second); err != nil {
			t.Fatal(err)
		}
		_, statErr := os.Stat(marker)
		if detach && statErr != nil {
			t.Error("détaché : les actions de fin ont tourné")
		}
		if !detach && statErr == nil {
			t.Error("arrêté : les actions de fin n'ont pas tourné")
		}
		if detach {
			exec.Command("taskkill", "/F", "/IM", filepath.Base(c.Command)).Run()
		}
	}
}
