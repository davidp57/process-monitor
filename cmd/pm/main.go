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
	"github.com/davidp57/process-monitor/internal/logx"
	"github.com/davidp57/process-monitor/internal/session"
	"github.com/davidp57/process-monitor/internal/tray"
	"github.com/davidp57/process-monitor/internal/tui"
)

var version = "dev" // set by the release build: -ldflags "-X main.version=x.y.z"

const usage = `pm %s — lance un programme et l'encadre.

  pm run   <config.yaml> [--profile NOM] [--tray]            actions avant → lancement → attente de la fin → actions après
  pm watch <config.yaml> [--profile NOM] [--tray|--no-tui]  garde le programme en vie (relance), actions avant/après la session

  --tray : une icône dans la zone de notification à la place de la fenêtre console (ou ui: tray dans la config)
  --icon FICHIER : l'image de cette icône, .ico (32 bits ou PNG) ou .png ; ou icon: dans la config
  pm check <config.yaml> [--profile NOM]            valide la config et affiche ce qui sera fait
  pm version
`

func main() {
	os.Exit(run(os.Args[1:]))
}

type args struct {
	cmd, path, profile string
	noTUI, tray        bool
	icon               string
}

// parse accepts flags before or after the config path.
func parse(argv []string) (args, error) {
	var a args
	if len(argv) == 0 {
		return a, errors.New("commande manquante")
	}
	a.cmd = argv[0]
	rest := argv[1:]
	for i := 0; i < len(rest); i++ {
		switch arg := rest[i]; {
		case arg == "--profile" || arg == "-p":
			if i+1 >= len(rest) {
				return a, fmt.Errorf("%s attend un nom de profil", arg)
			}
			i++
			a.profile = rest[i]
		case strings.HasPrefix(arg, "--profile="):
			a.profile = strings.TrimPrefix(arg, "--profile=")
		case arg == "--no-tui":
			a.noTUI = true
		case arg == "--tray":
			a.tray = true
		case arg == "--icon":
			if i+1 >= len(rest) {
				return a, fmt.Errorf("%s attend un fichier .ico ou .png", arg)
			}
			i++
			a.icon = rest[i]
		case strings.HasPrefix(arg, "-"):
			return a, fmt.Errorf("option inconnue %s", arg)
		case a.path == "":
			a.path = arg
		default:
			return a, fmt.Errorf("argument en trop : %s", arg)
		}
	}
	return a, nil
}

func run(argv []string) int {
	a, err := parse(argv)
	if err == nil && a.cmd == "version" {
		fmt.Println("pm", version)
		return 0
	}
	if err == nil && a.cmd != "run" && a.cmd != "watch" && a.cmd != "check" {
		err = fmt.Errorf("commande inconnue %q", a.cmd)
	}
	if err == nil && a.path == "" {
		err = errors.New("fichier de config manquant")
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "erreur : %v\n\n"+usage, err, version)
		return 2
	}

	cfg, err := config.Load(a.path, a.profile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "erreur de config :", err)
		if a.cmd != "check" {
			pressKey() // launched from a shortcut, the window would close before it can be read
		}
		return 2
	}
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
			tray.Alert(cfg.Name, fmt.Sprintf("Journal impossible à créer, rien n'a été lancé : %v", err))
			return 1
		}
		fmt.Fprintln(os.Stderr, "journal :", err)
		pressKey()
		return 1
	}
	defer log.Close()
	log.Infof("pm %s — %s (%s)", version, a.cmd, cfg.Path)
	if cfg.Profile != "" {
		log.Infof("profil %s", cfg.Profile)
	}

	restart := config.RestartNever
	if a.cmd == "watch" {
		restart = cfg.Watch.Restart
	}
	s := session.New(cfg, log, restart)

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
		log.Warnf("pm s'arrête, le programme continue sans surveillance")
	} else if code != 0 && a.cmd == "run" {
		log.Warnf("session terminée, le programme a rendu le code %d", code)
	} else {
		log.Infof("session terminée")
	}
	finish(cfg, err != nil || log.Errors() > 0)
	return finalCode(a, s, err, log)
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
		log.Warnf("icône non utilisée : %v", err)
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
			log.Errorf("fenêtre fermée : pm est coupé, les actions de fin ne seront PAS exécutées")
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
			log.Warnf("Ctrl+C encore une fois dans les 5 s pour annuler : rien ne sera lancé, les fichiers live seront remis comme avant")
		case session.Restarting:
			log.Warnf("Ctrl+C encore une fois dans les 5 s pour terminer sans relancer : rien ne tourne, les actions de fin seront exécutées")
		default:
			log.Warnf("Ctrl+C encore une fois dans les 5 s pour détacher : le programme continuera, les actions de fin ne seront PAS exécutées")
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
	fmt.Print("Entrée pour fermer…")
	_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
}

func printPlan(c *config.Config) {
	fmt.Printf("config  : %s\n", c.Path)
	if c.Profile != "" {
		fmt.Printf("profil  : %s\n", c.Profile)
	}
	if len(c.Profiles) > 0 {
		fmt.Printf("session : %s (verrou partagé par tous les profils)\n", c.Name)
	} else {
		fmt.Printf("session : %s\n", c.Name)
	}
	fmt.Printf("lance   : %s %s\n", c.Command, strings.Join(c.Args, " "))
	if c.Workdir != "" {
		fmt.Printf("dossier : %s\n", c.Workdir)
	}
	fmt.Printf("suit    : %s\n", c.ProcessName())
	fmt.Printf("si déjà lancé : %s\n", c.IfRunning)
	if c.Ready != nil {
		fmt.Printf("prêt quand : %s existe (délai %s)\n", c.Ready.File, c.Ready.Timeout.D())
	}
	fmt.Printf("journaux : %s (%d gardés)\n", c.Log.Dir, c.Log.Keep)
	missing := 0
	fmt.Println("\navant :")
	for _, p := range c.Sync {
		mark := ""
		if _, err := os.Stat(p.Source); err != nil {
			mark = "   ← SOURCE ABSENTE"
			missing++
		}
		fmt.Printf("  copie  %s\n      -> %s%s\n", p.Source, p.Target, mark)
	}
	for _, a := range c.Hooks.Before {
		mark := ""
		if a.Copy != nil {
			if _, err := os.Stat(a.Copy.From); err != nil {
				mark = "   ← SOURCE ABSENTE"
				missing++
			}
		}
		fmt.Println("  " + describe(a) + mark)
	}
	fmt.Println("après :")
	for _, p := range c.Sync {
		fmt.Printf("  copie  %s\n      -> %s\n", p.Target, p.Source)
	}
	for _, a := range c.Hooks.After {
		fmt.Println("  " + describe(a))
	}
	if missing > 0 {
		fmt.Printf("\n%d source(s) absente(s) : pm run échouerait avant de lancer quoi que ce soit.\n", missing)
	}
}

func describe(a config.Action) string {
	switch {
	case a.Copy != nil:
		return fmt.Sprintf("copie  %s\n      -> %s", a.Copy.From, a.Copy.To)
	case a.Delete != "":
		return "suppr. " + a.Delete
	default:
		return "exec   " + strings.Join(a.Exec, " ")
	}
}
