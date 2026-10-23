package proc

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

// The test binary doubles as the supervised program, driven by PM_HELPER:
//
//	exit:N            print two lines, exit with code N
//	sleep:MS:N        sleep MS milliseconds, exit N
//	launch:EXE:MODE   start EXE with PM_HELPER=MODE and exit at once
//	breakaway:EXE:MODE same, asking to leave the job
func TestMain(m *testing.M) {
	mode := os.Getenv("PM_HELPER")
	if mode == "" {
		os.Exit(m.Run())
	}
	var kind, a, b string
	parts := splitN(mode, 3)
	kind, a = parts[0], parts[1]
	if len(parts) > 2 {
		b = parts[2]
	}
	switch kind {
	case "exit":
		fmt.Println("hello stdout")
		fmt.Fprintln(os.Stderr, "hello stderr")
		n, _ := strconv.Atoi(a)
		os.Exit(n)
	case "sleep":
		ms, _ := strconv.Atoi(a)
		time.Sleep(time.Duration(ms) * time.Millisecond)
		n, _ := strconv.Atoi(b)
		os.Exit(n)
	case "launch", "breakaway", "linger":
		exe, child := a, b
		cmd := exec.Command(exe)
		cmd.Env = append(os.Environ(), "PM_HELPER="+child)
		if kind == "breakaway" {
			cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: windows.CREATE_BREAKAWAY_FROM_JOB}
		}
		if err := cmd.Start(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(99)
		}
		if kind == "linger" {
			time.Sleep(5 * time.Second) // a launcher that outlives the game
		}
		os.Exit(0)
	}
	os.Exit(98)
}

// splitN splits on ':' but keeps a Windows drive colon inside the middle field.
func splitN(s string, n int) []string {
	var out []string
	for len(out) < n-1 {
		i := 0
		for i < len(s) && s[i] != ':' {
			i++
		}
		if i == 1 && len(s) > 2 && s[2] == '\\' { // "C:\..."
			j := 2
			for j < len(s) && s[j] != ':' {
				j++
			}
			i = j
		}
		if i >= len(s) {
			break
		}
		out = append(out, s[:i])
		s = s[i+1:]
	}
	return append(out, s)
}

func self(t *testing.T) string {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	return exe
}

// copySelf gives the test binary another image name, to be tracked by name.
func copySelf(t *testing.T, name string) string {
	dst := filepath.Join(t.TempDir(), name)
	src, err := os.Open(self(t))
	if err != nil {
		t.Fatal(err)
	}
	defer src.Close()
	out, err := os.Create(dst)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(out, src); err != nil {
		t.Fatal(err)
	}
	out.Close()
	return dst
}

func start(t *testing.T, mode string, spec Spec) Process {
	t.Helper()
	if spec.Command == "" {
		spec.Command = self(t)
	}
	spec.Env = append(os.Environ(), "PM_HELPER="+mode)
	p, err := Start(spec)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(p.Close)
	return p
}

func wait(t *testing.T, p Process, d time.Duration) Result {
	t.Helper()
	select {
	case <-p.Done():
		return p.Result()
	case <-time.After(d):
		_ = p.Kill()
		t.Fatalf("session toujours en cours après %s", d)
		return Result{}
	}
}

func TestExitCodeAndCapture(t *testing.T) {
	var mu sync.Mutex
	var lines []string
	p := start(t, "exit:3", Spec{Capture: true, Output: func(l string) {
		mu.Lock()
		lines = append(lines, l)
		mu.Unlock()
	}})
	r := wait(t, p, 10*time.Second)
	if r.ExitCode != 3 || r.Err != nil {
		t.Errorf("result = %+v", r)
	}
	time.Sleep(200 * time.Millisecond) // let the pipe readers drain
	mu.Lock()
	defer mu.Unlock()
	if len(lines) != 2 {
		t.Errorf("lines = %q", lines)
	}
}

func TestTrackFollowsChildAfterLauncherExits(t *testing.T) {
	game := copySelf(t, "pm-test-game.exe")
	var events []string
	p := start(t, "launch:"+game+":sleep:1500:7", Spec{Track: "pm-test-game.exe", TrackGrace: 5 * time.Second,
		Event: func(m string) { events = append(events, m) }})
	begin := time.Now()
	r := wait(t, p, 15*time.Second)
	if r.ExitCode != 7 || r.Err != nil {
		t.Errorf("result = %+v (events %q)", r, events)
	}
	if time.Since(begin) < time.Second {
		t.Errorf("session finie en %s, avant la fin du process suivi", time.Since(begin))
	}
}

func TestTrackFollowsChildThatLeftTheJob(t *testing.T) {
	game := copySelf(t, "pm-test-escapee.exe")
	var events []string
	p := start(t, "breakaway:"+game+":sleep:1500:5", Spec{Track: "pm-test-escapee.exe", TrackGrace: 5 * time.Second,
		Event: func(m string) { events = append(events, m) }})
	r := wait(t, p, 15*time.Second)
	if r.ExitCode != 5 || r.Err != nil {
		t.Errorf("result = %+v (events %q)", r, events)
	}
	if len(events) == 0 || !strings.Contains(events[len(events)-1], "quitté le Job Object") {
		t.Errorf("le process n'a pas été retrouvé par le chemin de secours : %q", events)
	}
}

// Windows 11's System32\notepad.exe hands over to a packaged Notepad.exe:
// launcher and target share a name. Following the launcher alone would end
// the session (and copy a profile back) while the target still runs.
func TestTrackFollowsHandoverToSameName(t *testing.T) {
	launcher := copySelf(t, "pm-test-same.exe")
	target := copySelf(t, "pm-test-same.exe") // other directory, same name
	var events []string
	p := start(t, "launch:"+target+":sleep:4000:7", Spec{Command: launcher, Track: "pm-test-same.exe",
		TrackGrace: 5 * time.Second, Event: func(m string) { events = append(events, m) }})
	begin := time.Now()
	r := wait(t, p, 15*time.Second)
	if r.ExitCode != 7 || r.Err != nil || time.Since(begin) < 3*time.Second {
		t.Errorf("result = %+v après %s (events %q)", r, time.Since(begin), events)
	}
}

// The game exits but its launcher still runs (it may start the game again):
// the session must not end, or the profile would be copied back too early.
func TestTrackWaitsForLingeringLauncher(t *testing.T) {
	game := copySelf(t, "pm-test-linger.exe")
	var events []string
	p := start(t, "linger:"+game+":sleep:500:7", Spec{Track: "pm-test-linger.exe", TrackGrace: 5 * time.Second,
		Event: func(m string) { events = append(events, m) }})
	begin := time.Now()
	r := wait(t, p, 20*time.Second)
	if r.ExitCode != 7 || r.Err != nil || time.Since(begin) < 4*time.Second {
		t.Errorf("result = %+v après %s (events %q)", r, time.Since(begin), events)
	}
}

func TestTrackNeverAppears(t *testing.T) {
	p := start(t, "exit:0", Spec{Track: "pm-test-nothing.exe", TrackGrace: 1500 * time.Millisecond})
	r := wait(t, p, 10*time.Second)
	if r.Err == nil {
		t.Errorf("result = %+v", r)
	}
}

func TestWithoutTrackWaitsForWholeJob(t *testing.T) {
	child := copySelf(t, "pm-test-child.exe")
	p := start(t, "launch:"+child+":sleep:1500:0", Spec{})
	begin := time.Now()
	wait(t, p, 15*time.Second)
	if time.Since(begin) < time.Second {
		t.Errorf("session finie en %s alors qu'un enfant tournait encore", time.Since(begin))
	}
}

func TestKillAndSuspend(t *testing.T) {
	p := start(t, "sleep:60000:0", Spec{})
	time.Sleep(300 * time.Millisecond)
	if err := p.Suspend(); err != nil {
		t.Errorf("suspend : %v", err)
	}
	if err := p.Resume(); err != nil {
		t.Errorf("resume : %v", err)
	}
	if err := p.Kill(); err != nil {
		t.Errorf("kill : %v", err)
	}
	r := wait(t, p, 10*time.Second)
	if r.ExitCode != 1 {
		t.Errorf("result = %+v", r)
	}
}

func TestFindAndKillByName(t *testing.T) {
	victim := copySelf(t, "pm-test-victim.exe")
	cmd := exec.Command(victim)
	cmd.Env = append(os.Environ(), "PM_HELPER=sleep:60000:0")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	pids, err := FindByName("PM-TEST-VICTIM.EXE")
	if err != nil || len(pids) != 1 || pids[0] != cmd.Process.Pid {
		t.Fatalf("pids = %v, err = %v", pids, err)
	}
	n, err := KillByName("pm-test-victim.exe", 5*time.Second)
	if n != 1 || err != nil {
		t.Errorf("killed = %d, err = %v", n, err)
	}
	_ = cmd.Wait()
}
