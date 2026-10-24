package control

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/davidp57/process-monitor/internal/config"
	"github.com/davidp57/process-monitor/internal/i18n"
	"github.com/davidp57/process-monitor/internal/lock"
	"github.com/davidp57/process-monitor/internal/logx"
	"github.com/davidp57/process-monitor/internal/session"
)

// The test binary doubles as the supervised program, driven by PM_HELPER:
//
//	sleep:MS   sleep MS milliseconds, exit 0
func TestMain(m *testing.M) {
	mode, arg, _ := strings.Cut(os.Getenv("PM_HELPER"), ":")
	switch mode {
	case "":
		i18n.Set("fr")
		os.Exit(m.Run())
	case "sleep":
		ms, _ := strconv.Atoi(arg)
		time.Sleep(time.Duration(ms) * time.Millisecond)
		os.Exit(0)
	}
	os.Exit(98)
}

var seq int

// program copies the test binary under a unique name, so the already
// running check sees neither the test nor another test's program.
func program(t *testing.T) string {
	t.Helper()
	seq++
	dst := filepath.Join(t.TempDir(), fmt.Sprintf("pm-control-%d-%d.exe", os.Getpid(), seq))
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

func find(name string) *Client {
	list, _ := List()
	for _, c := range list {
		if c.Info.Name == name {
			return c
		}
	}
	return nil
}

func TestControlASession(t *testing.T) {
	exe := program(t)
	c := &config.Config{Name: filepath.Base(exe), Command: exe, Env: map[string]string{"PM_HELPER": "sleep:30000"},
		IfRunning: config.IfRunningAbort}
	log, err := logx.New("", "", 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	s := session.New(c, log, config.RestartNever)
	srv, err := Serve(s, log, Info{Name: c.Name, Config: `C:\cfg\game.yaml`, Mode: "run"})
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- s.Run(context.Background()); srv.Close() }()
	t.Cleanup(func() { s.Kill() })

	var cl *Client
	for deadline := time.Now().Add(10 * time.Second); ; time.Sleep(50 * time.Millisecond) {
		if cl = find(c.Name); cl != nil {
			if st, err := cl.State(); err == nil && st.State.Status == session.Running {
				break
			}
		}
		if time.Now().After(deadline) {
			t.Fatal("session jamais listée en cours")
		}
	}
	st, _ := cl.State()
	if st.Info.Config != `C:\cfg\game.yaml` || st.Info.Token != "" || st.Info.PID != os.Getpid() {
		t.Errorf("info : %+v", st.Info)
	}
	lines, err := cl.Log()
	if err != nil || len(lines) == 0 {
		t.Errorf("journal : %v, %v", lines, err)
	}

	// Live lines: the stop below is logged, and reaches the stream.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var mu sync.Mutex
	var live []string
	go cl.Events(ctx, func(l logx.Line) { mu.Lock(); live = append(live, l.Text); mu.Unlock() })
	time.Sleep(300 * time.Millisecond) // let the stream open
	if err := cl.Command(Stop); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(15 * time.Second):
		t.Fatal("session pas arrêtée")
	}
	mu.Lock()
	got := strings.Join(live, "|")
	mu.Unlock()
	if !strings.Contains(got, "arrêt") {
		t.Errorf("lignes en direct : %q", got)
	}
	if !s.State().Stopped || find(c.Name) != nil {
		t.Errorf("après l'arrêt : %+v, encore listée %v", s.State(), find(c.Name) != nil)
	}
	if err := cl.Command("explode"); err == nil {
		t.Error("commande sur une session finie acceptée")
	}
}

// Only a request to 127.0.0.1 with the token gets through: not another
// program without it, not a web page reaching it under another name.
func TestGuard(t *testing.T) {
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	port := ln.Addr().(*net.TCPAddr).Port
	go http.Serve(ln, Guard(port, "secret", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})))
	defer ln.Close()
	url := "http://127.0.0.1:" + strconv.Itoa(port) + "/state"
	for _, tc := range []struct {
		token, host string
		want        int
	}{
		{"secret", "", http.StatusOK},
		{"", "", http.StatusForbidden},
		{"wrong", "", http.StatusForbidden},
		{"secret", "evil.example:" + strconv.Itoa(port), http.StatusForbidden},
	} {
		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Set(TokenHeader, tc.token)
		if tc.host != "" {
			req.Host = tc.host
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != tc.want {
			t.Errorf("%+v : %d", tc, resp.StatusCode)
		}
	}
}

// A file left by a session that died without removing it is cleaned up.
func TestStaleInfoIsRemoved(t *testing.T) {
	gone := exec.Command(program(t))
	gone.Env = append(os.Environ(), "PM_HELPER=sleep:0")
	if err := gone.Run(); err != nil {
		t.Fatal(err)
	}
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close() // nobody answers there
	name := fmt.Sprintf("pm-control-stale-%d", os.Getpid())
	path, _ := lock.InfoPath(name)
	os.MkdirAll(filepath.Dir(path), 0o755)
	data, _ := json.Marshal(Info{Name: name, Port: port, PID: gone.Process.Pid, Token: "x"})
	os.WriteFile(path, data, 0o600)
	if find(name) != nil {
		t.Error("session morte listée")
	}
	if _, err := os.Stat(path); err == nil {
		os.Remove(path)
		t.Error("fichier laissé")
	}
}

// A session that closes late (a key to press after an error) leaves the
// info file of the session that took its name meanwhile.
func TestCloseLeavesTheNextSessionsFile(t *testing.T) {
	name := fmt.Sprintf("pm-control-next-%d", os.Getpid())
	path, _ := lock.InfoPath(name)
	os.MkdirAll(filepath.Dir(path), 0o755)
	t.Cleanup(func() { os.Remove(path) })
	old := &Server{info: Info{Name: name, Port: 1}, http: &http.Server{}}
	old.publish()
	if _, err := os.Stat(path); err != nil {
		t.Fatal("pas publié")
	}
	next, _ := json.Marshal(Info{Name: name, Port: 2})
	os.WriteFile(path, next, 0o600)
	old.Close()
	if now, err := os.ReadFile(path); err != nil || string(now) != string(next) {
		t.Fatalf("le fichier de la session suivante a disparu : %q %v", now, err)
	}
	// Alone, it removes its own.
	alone := &Server{info: Info{Name: name, Port: 3}, http: &http.Server{}}
	alone.publish()
	alone.Close()
	if _, err := os.Stat(path); err == nil {
		t.Error("son propre fichier laissé")
	}
}
