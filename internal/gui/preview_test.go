package gui

import (
	"context"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/davidp57/process-monitor/internal/config"
	"github.com/davidp57/process-monitor/internal/control"
	"github.com/davidp57/process-monitor/internal/logx"
	"github.com/davidp57/process-monitor/internal/registry"
	"github.com/davidp57/process-monitor/internal/session"
)

// TestServeForPreview serves the interface on PM_GUI_PREVIEW (a port),
// with a registry and configs of its own in a temporary folder, until the
// test is stopped: to look at the page in a browser without starting pm
// gui (which opens the default browser and puts an icon on the desktop).
// Launching runs copies of the test binary that sleep, never a program.
// Skipped unless the variable is set.
func TestServeForPreview(t *testing.T) {
	port, err := strconv.Atoi(os.Getenv("PM_GUI_PREVIEW"))
	if err != nil {
		t.Skip("PM_GUI_PREVIEW not set")
	}
	d := t.TempDir()
	t.Cleanup(registry.UseFile(filepath.Join(d, "configs.json")))
	self, _ := os.Executable()
	// A name of its own: nothing of the user's is ever taken for it.
	game := filepath.Join(d, "pm-preview-game-"+strconv.Itoa(os.Getpid())+".exe")
	data, _ := os.ReadFile(self)
	os.WriteFile(game, data, 0o755)
	cfg := filepath.Join(d, "Profiles", "pm-dcs.yaml")
	put(t, cfg, "# DCS by profile\nname: dcs\ncommand: '"+game+"'\nenv: { PM_HELPER: 'sleep:600000' }\nui: console\nprofiles:\n  DEV: {}\n  VR: {}\n")
	put(t, filepath.Join(d, "Profiles", "Icons", "dcs-dev.ico"), "")
	registry.Remember(cfg)
	ln, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(port))
	if err != nil {
		t.Fatal(err)
	}
	srv := New(Options{Port: port, Token: "preview", Self: self, Defaults: func() []string { return nil }, Launch: func(args []string) error {
		// pm run as the test binary would run the tests again: start the
		// game directly instead, a copy of the test binary that sleeps.
		cmd := exec.Command(game)
		cmd.Env = append(os.Environ(), "PM_HELPER=sleep:60000")
		return cmd.Start()
	}})
	// A session running, to see it listed and drive it: the game sleeps.
	c := &config.Config{Name: "sim-preview", Command: game, Env: map[string]string{"PM_HELPER": "sleep:600000"},
		IfRunning: config.IfRunningAbort, Watch: config.Watch{Restart: config.RestartAlways,
			Delay: config.Duration(time.Second), MaxDelay: config.Duration(time.Minute)}}
	log, _ := logx.New("", "", 0, nil)
	s := session.New(c, log, config.RestartAlways)
	ctl, err := control.Serve(s, log, control.Info{Name: c.Name, Config: cfg, Mode: "watch"})
	if err != nil {
		t.Fatal(err)
	}
	go func() { s.Run(context.Background()); ctl.Close() }()
	t.Cleanup(func() { s.Kill() })
	t.Logf("http://127.0.0.1:%d/?token=preview (configs in %s)", port, d)
	http.Serve(ln, srv.Handler())
}
