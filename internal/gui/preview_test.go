package gui

import (
	"context"
	"image"
	"image/color"
	"image/png"
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
	"github.com/davidp57/process-monitor/internal/filedialog"
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
	// PM_GUI_DEMO names a folder for the documentation's screenshots: a
	// believable set of configs there, and no path of the user's on screen.
	d, demo := os.Getenv("PM_GUI_DEMO"), true
	if d == "" {
		d, demo = t.TempDir(), false
	} else {
		if _, err := os.Stat(d); err == nil {
			t.Fatalf("%s existe déjà : la démo ne l'écrase pas", d)
		}
		t.Cleanup(func() { os.RemoveAll(d) })
	}
	t.Cleanup(registry.UseFile(filepath.Join(d, "configs.json")))
	self, _ := os.Executable()
	// A name of its own: nothing of the user's is ever taken for it.
	game := filepath.Join(d, "pm-preview-game-"+strconv.Itoa(os.Getpid())+".exe")
	if demo { // its log shows on the screenshots; still not a real program's name
		game = filepath.Join(d, "SimAppPro", "SimAppPro-demo.exe")
	}
	data, _ := os.ReadFile(self)
	put(t, game, "")
	os.WriteFile(game, data, 0o755)
	cfg, name := previewConfigs(t, d, game, demo)
	ln, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(port))
	if err != nil {
		t.Fatal(err)
	}
	// No Windows file window on the desktop: "Browse" answers the game.
	pick := func(filedialog.Options) (string, bool, error) { return game, true, nil }
	srv := New(Options{Port: port, Token: "preview", Self: self, Defaults: func() []string { return nil }, Pick: pick, Launch: func(args []string) error {
		// pm run as the test binary would run the tests again: start the
		// game directly instead, a copy of the test binary that sleeps.
		cmd := exec.Command(game)
		cmd.Env = append(os.Environ(), "PM_HELPER=sleep:60000")
		return cmd.Start()
	}})
	// A session running, to see it listed and drive it: the game sleeps.
	c := &config.Config{Name: name, Command: game, Env: map[string]string{"PM_HELPER": "sleep:600000"},
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

// previewConfigs writes the configs the preview lists, and answers the
// config of the session it runs and that session's name.
func previewConfigs(t *testing.T, d, game string, demo bool) (string, string) {
	if !demo {
		cfg := filepath.Join(d, "Profiles", "pm-dcs.yaml")
		put(t, cfg, "# DCS by profile\nname: dcs\ncommand: '"+game+"'\nenv: { PM_HELPER: 'sleep:600000' }\nui: console\n"+
			"vars: { prof: '${config_dir}\\${profile}', live: '${config_dir}\\Saved Games\\DCS' }\n"+
			"hooks:\n  sync:\n    - ${prof}\\options.lua <-> ${live}\\Config\\options.lua\n    - ${prof}\\Input <-> ${live}\\Config\\Input\n    - ${prof}\\Kneeboard <-> ${live}\\Kneeboard\n"+
			"  before: [ { delete: '${live}\\Temp' } ]\n  after: [ { exec: [ backup.cmd, '${profile}' ] } ]\n"+
			"profiles:\n  DEV: {}\n  VR: {}\n")
		put(t, filepath.Join(d, "Profiles", "DEV", "options.lua"), "")
		put(t, filepath.Join(d, "Profiles", "DEV", "Input", "joystick.lua"), "")
		put(t, filepath.Join(d, "Profiles", "Saved Games", "DCS", "Config", "Input", "joystick.lua"), "")
		put(t, filepath.Join(d, "Profiles", "Icons", "dcs-dev.ico"), "")
		registry.Remember(cfg)
		return cfg, "sim-preview"
	}
	saved := filepath.Join(d, "Saved Games", "DCS")
	profiles := filepath.Join(saved, "Profiles")
	dcs := filepath.Join(profiles, "pm-dcs.yaml")
	put(t, dcs, "# DCS World, one profile per way of flying\n"+
		"name: dcs\n"+
		"command: '"+filepath.Join(d, "DCS World", "bin", "DCS_updater.exe")+"'\n"+
		"track: DCS.exe\n"+
		"kill_on_exit: [ SR-ClientRadio.exe ]\n"+
		"ui: tray\n"+
		"vars:\n"+
		"  prof: ${config_dir}\\${profile}\n"+
		"  live: '"+saved+"'\n"+
		"hooks:\n"+
		"  sync:\n"+
		"    - ${prof}\\options.lua <-> ${live}\\Config\\options.lua\n"+
		"    - ${prof}\\Input <-> ${live}\\Config\\Input\n"+
		"    - ${prof}\\Kneeboard <-> ${live}\\Kneeboard\n"+
		"  before:\n"+
		"    - delete: ${live}\\Temp\n"+
		"shortcuts:\n"+
		"  mode: run\n"+
		"  folder: [ '${desktop}' ]\n"+
		"profiles:\n"+
		"  DEV: { icon: '${config_dir}\\Icons\\dcs-dev.png' }\n"+
		"  VR: { icon: '${config_dir}\\Icons\\dcs-vr.png' }\n"+
		"  STD: {}\n")
	put(t, filepath.Join(d, "DCS World", "bin", "DCS_updater.exe"), "")
	put(t, filepath.Join(saved, "Config", "options.lua"), "")
	put(t, filepath.Join(saved, "Config", "Input", "joystick.lua"), "")
	for _, p := range []string{"DEV", "VR", "STD"} {
		put(t, filepath.Join(profiles, p, "options.lua"), "")
		put(t, filepath.Join(profiles, p, "Input", "joystick.lua"), "")
	}
	put(t, filepath.Join(profiles, "DEV", "Kneeboard", "page1.png"), "")
	demoIcon(t, filepath.Join(profiles, "Icons", "dcs-dev.png"), color.RGBA{0x5f, 0xc0, 0x8a, 0xff})
	demoIcon(t, filepath.Join(profiles, "Icons", "dcs-vr.png"), color.RGBA{0x6a, 0xa7, 0xe5, 0xff})
	logs := filepath.Join(profiles, "logs")
	put(t, filepath.Join(logs, "dcs-DEV-20261023-204512.log"), "20:45:12 INFO  profil : DEV\n20:45:13 INFO  lancement de DCS_updater.exe\n22:31:40 INFO  DCS.exe s'est terminé\n")
	put(t, filepath.Join(logs, "dcs-DEV-20261024-181002.log"), "18:10:02 INFO  profil : DEV\n")

	sim := filepath.Join(d, "SimAppPro", "pm-simapppro.yaml")
	put(t, sim, "# SimAppPro, kept running\n"+
		"name: simapppro\n"+
		"command: '"+game+"'\n"+
		"ui: tray\n"+
		"watch: { restart: always, delay: 5s }\n"+
		"shortcuts: { mode: watch, folder: [ '${start_menu}' ] }\n")
	registry.Remember(dcs)
	registry.Remember(sim)
	return sim, "simapppro"
}

// demoIcon writes a small round icon of colour c.
func demoIcon(t *testing.T, path string, c color.RGBA) {
	img := image.NewRGBA(image.Rect(0, 0, 32, 32))
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			if (x-16)*(x-16)+(y-16)*(y-16) < 15*15 {
				img.Set(x, y, c)
			}
		}
	}
	put(t, path, "")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	png.Encode(f, img)
	f.Close()
}
