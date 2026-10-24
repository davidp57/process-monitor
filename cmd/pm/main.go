// pm runs a program with actions before and after it (pm run), or keeps it
// alive (pm watch). See README.md.
package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"image"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/davidp57/process-monitor/internal/config"
	"github.com/davidp57/process-monitor/internal/control"
	"github.com/davidp57/process-monitor/internal/logx"
	"github.com/davidp57/process-monitor/internal/registry"
	"github.com/davidp57/process-monitor/internal/session"
	"github.com/davidp57/process-monitor/internal/tray"
	"github.com/davidp57/process-monitor/internal/tui"

	"github.com/davidp57/process-monitor/internal/i18n"
)

var version = "dev" // set by the release build: -ldflags "-X main.version=x.y.z"

func main() {
	os.Exit(run(os.Args[1:]))
}

type args struct {
	cmd, path, profile string
	noTUI, tray        bool
	icon, lang         string
}

// langFlag finds --lang before the arguments are parsed.
func langFlag(argv []string) string {
	for i, arg := range argv {
		if arg == "--lang" && i+1 < len(argv) {
			return argv[i+1]
		}
	}
	return ""
}

// parse accepts flags before or after the config path.
func parse(argv []string) (args, error) {
	var a args
	if len(argv) == 0 {
		return a, errors.New(i18n.T("cli.no_command"))
	}
	a.cmd = argv[0]
	rest := argv[1:]
	for i := 0; i < len(rest); i++ {
		switch arg := rest[i]; {
		case arg == "--profile" || arg == "-p":
			if i+1 >= len(rest) {
				return a, i18n.Errorf("cli.profile_value", arg)
			}
			i++
			a.profile = rest[i]
		case strings.HasPrefix(arg, "--profile="):
			a.profile = strings.TrimPrefix(arg, "--profile=")
		case arg == "--no-tui":
			a.noTUI = true
		case arg == "--tray":
			a.tray = true
		case arg == "--lang":
			if i+1 >= len(rest) || !i18n.Supported(rest[i+1]) {
				return a, i18n.Errorf("cli.lang_value", arg)
			}
			i++
			a.lang = rest[i]
		case arg == "--icon":
			if i+1 >= len(rest) {
				return a, i18n.Errorf("cli.icon_value", arg)
			}
			i++
			a.icon = rest[i]
		case strings.HasPrefix(arg, "-"):
			return a, i18n.Errorf("cli.unknown_option", arg)
		case a.path == "":
			a.path = arg
		default:
			return a, i18n.Errorf("cli.extra_argument", arg)
		}
	}
	return a, nil
}

func run(argv []string) int {
	// The language comes first: parsing errors are already messages.
	i18n.Set(i18n.Resolve(langFlag(argv), os.Getenv("PM_LANG"), "", userLanguage))
	// Double-clicked: no arguments, and a console window of its own. From
	// a terminal, no arguments still shows the usage.
	if len(argv) == 0 && interactive() && consoleAlone() {
		return guiMain()
	}
	a, err := parse(argv)
	if err == nil && a.cmd == "version" {
		fmt.Println("pm", version)
		return 0
	}
	switch {
	case err != nil:
	case a.cmd == "new":
		return newConfig(a)
	case a.cmd == "gui":
		return guiMain()
	case a.cmd == "menu":
		return menu()
	case a.cmd != "run" && a.cmd != "watch" && a.cmd != "check" && a.cmd != "shortcut":
		err = i18n.Errorf("cli.unknown_command", a.cmd)
	case a.path == "":
		err = errors.New(i18n.T("cli.no_config"))
	}
	if err != nil {
		fmt.Fprint(os.Stderr, i18n.T("cli.usage_error", err)+"\n\n"+i18n.T("cli.usage", version))
		return 2
	}
	if a.cmd == "shortcut" {
		return shortcuts(a)
	}

	cfg, err := config.Load(a.path, a.profile)
	if err != nil {
		fmt.Fprintln(os.Stderr, i18n.T("cli.config_error", err))
		if a.cmd != "check" {
			pressKey() // launched from a shortcut, the window would close before it can be read
		}
		return 2
	}
	// The config's language counts only when neither --lang nor PM_LANG
	// named one pm speaks: the same order as Resolve. It comes before the
	// setting kept for every config.
	if !i18n.Supported(a.lang) && !i18n.Supported(os.Getenv("PM_LANG")) && cfg.Lang != "" {
		i18n.Set(cfg.Lang)
	}
	registry.Remember(cfg.Path) // for the interface's list of configs
	if a.cmd == "check" {
		printPlan(cfg)
		return 0
	}

	useTray := a.tray || cfg.UI == config.UITray
	useTUI := a.cmd == "watch" && !a.noTUI && !useTray
	// From a shortcut the console is ours alone: give it back, its window
	// closes. From a terminal it is shared: keep printing to it.
	console := !(useTray && releaseConsole())
	prefix := cfg.Name
	if cfg.Profile != "" {
		prefix += "-" + cfg.Profile
	}
	var sink func(logx.Line)
	if !useTUI && console {
		sink = func(l logx.Line) { fmt.Println(l) }
	}
	log, err := logx.New(cfg.Log.Dir, prefix, cfg.Log.Keep, sink)
	if err != nil {
		if !console {
			tray.Alert(cfg.Name, i18n.T("cli.log_failed", err))
			return 1
		}
		fmt.Fprintln(os.Stderr, i18n.T("cli.log_error", err))
		pressKey()
		return 1
	}
	defer log.Close()
	log.Infof("pm %s — %s (%s)", version, a.cmd, cfg.Path)
	if cfg.Profile != "" {
		log.Info(i18n.T("run.profile", cfg.Profile))
	}

	restart := config.RestartNever
	if a.cmd == "watch" {
		restart = cfg.Watch.Restart
	}
	s := session.New(cfg, log, restart)
	// The control channel lets the interface see and drive this session.
	// Without it the session still runs: the icon and the terminal remain.
	if ctl, err := control.Serve(s, log, control.Info{Name: cfg.Name, Profile: cfg.Profile, Config: cfg.Path, Mode: a.cmd}); err != nil {
		log.Warn(i18n.T("run.control_failed", err))
	} else {
		defer ctl.Close()
	}

	if useTUI {
		err = tui.Run(context.Background(), s, log)
		return exitCode(err, log)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if useTray {
		if console {
			// Started from a terminal: Ctrl+C and closing it still need the guard.
			go interrupts(log, cancel, func() session.Status { return s.State().Status })
		}
		err = tray.Run(ctx, s, log, a.cmd == "watch", trayImage(a, cfg, log))
		return finalCode(a, s, err, log)
	}
	go interrupts(log, cancel, func() session.Status { return s.State().Status })
	err = s.Run(ctx)
	code := s.State().ExitCode
	detached := s.State().Status == session.Detached
	if err != nil {
		log.Errorf("%v", err)
	} else if detached {
		log.Warn(i18n.T("run.detached"))
	} else if code != 0 && a.cmd == "run" && !s.State().Stopped {
		log.Warn(i18n.T("run.ended_code", code))
	} else {
		log.Info(i18n.T("run.ended"))
	}
	finish(cfg, err != nil || log.Errors() > 0)
	return finalCode(a, s, err, log)
}

// userLanguage is the language set in the interface (kept by pm for every
// config without lang:), else Windows'.
func userLanguage() string {
	if l := registry.Lang(); l != "" {
		return l
	}
	return i18n.SystemLanguage()
}

// trayImage loads the icon image: --icon, else the config's icon. A file
// that cannot be read is logged and the default disc is used: a missing
// picture must not stop a game from starting.
func trayImage(a args, cfg *config.Config, log *logx.Logger) image.Image {
	path := cfg.Icon
	if a.icon != "" {
		path, _ = filepath.Abs(a.icon)
	}
	if path == "" {
		return nil
	}
	img, err := tray.LoadImage(path)
	if err != nil {
		log.Warn(i18n.T("run.icon_unused", err))
		return nil
	}
	return img
}

// finalCode is pm's exit code: 1 when pm failed, the program's own code for
// pm run, and 1 for a pm run that detached (its copy-back did not happen).
func finalCode(a args, s *session.Session, err error, log *logx.Logger) int {
	if rc := exitCode(err, log); rc != 0 || a.cmd != "run" {
		return rc
	}
	if s.State().Status == session.Detached {
		return 1
	}
	return s.State().ExitCode
}

// interrupts acts on a second Ctrl+C within five seconds: one stray
// keystroke must not cost the copy-back of a profile. What the second one
// does depends on the moment, and the message says which. Closing the
// console window cannot be held off; it is logged so the journal says why
// the after hooks are missing.
func interrupts(log *logx.Logger, stop context.CancelFunc, status func() session.Status) {
	ch := make(chan os.Signal, 2)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	var last time.Time
	for sig := range ch {
		if sig == syscall.SIGTERM {
			log.Error(i18n.T("run.window_closed"))
			stop()
			continue
		}
		if time.Since(last) < 5*time.Second {
			stop()
			continue
		}
		last = time.Now()
		switch status() {
		case session.Starting:
			log.Warn(i18n.T("run.ctrlc_cancel"))
		case session.Restarting:
			log.Warn(i18n.T("run.ctrlc_end"))
		default:
			log.Warn(i18n.T("run.ctrlc_detach"))
		}
	}
}

func exitCode(err error, log *logx.Logger) int {
	if err != nil || log.Errors() > 0 {
		return 1
	}
	return 0
}

func finish(cfg *config.Config, failed bool) {
	if d := cfg.Exit.Delay.D(); d > 0 && !failed {
		time.Sleep(d)
	}
	if cfg.Exit.WaitKey == config.WaitKeyAlways || (cfg.Exit.WaitKey == config.WaitKeyOnError && failed) {
		pressKey()
	}
}

// pressKey waits for Enter, only when a person can press it.
func pressKey() {
	if !interactive() {
		return
	}
	fmt.Print(i18n.T("cli.press_enter"))
	_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
}

func printPlan(c *config.Config) { fmt.Print(c.Plan()) }
