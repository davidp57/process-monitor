package wizard

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/davidp57/process-monitor/internal/config"
	"github.com/davidp57/process-monitor/internal/i18n"
	"github.com/davidp57/process-monitor/internal/logx"
	"github.com/davidp57/process-monitor/internal/proc"
	"github.com/davidp57/process-monitor/internal/session"
	"github.com/davidp57/process-monitor/internal/shortcut"
)

// Runner runs the engine's actions on the machine. It is the part of the
// wizard that touches files and starts the game; every front shares it.
type Runner struct {
	W   *Wizard
	Log *logx.Logger // the game session's lines while a profile is learned
	PM  string       // the pm.exe the shortcuts start
}

// Do runs a. Its result goes back to the engine with Wizard.Done; Apply
// returns the shortcuts to write.
func (r *Runner) Do(ctx context.Context, a Action) (any, error) {
	dir := r.W.WizardDir()
	switch a.Kind {
	case ActRecover, ActLearn, ActRestoreOriginal:
		// Never put files back under a running game: it would write its own
		// over them when it closes.
		if err := r.checkGame(); err != nil {
			return nil, err
		}
	}
	switch a.Kind {
	case ActRecover:
		if err := Restore(dir); err != nil {
			return nil, err
		}
		return nil, os.RemoveAll(dir)
	case ActSaveOriginal:
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
		return SaveOriginal(dir, r.W.Roots(), r.W.Ignored())
	case ActLearn:
		return r.learn(ctx, a.Profile)
	case ActRestoreOriginal:
		return nil, Restore(dir)
	case ActApply:
		return r.apply()
	case ActDiscard:
		return nil, os.RemoveAll(dir)
	}
	return nil, nil
}

// Abandon puts the original files back when the wizard stops before the
// end, and removes its copies. If that fails, the copies stay: the next
// pm new on the same config puts them back first.
func (r *Runner) Abandon() error {
	if !r.W.has("config") {
		return nil
	}
	dir := r.W.WizardDir()
	if !HasOriginal(dir) {
		return nil
	}
	if err := r.checkGame(); err != nil {
		return i18n.Errorf("wizard.restore_failed", err, dir)
	}
	if err := Restore(dir); err != nil {
		return i18n.Errorf("wizard.restore_failed", err, dir)
	}
	return os.RemoveAll(dir)
}

// Running is a game process found running when the wizard has to touch
// the game's files.
type Running struct {
	Name string
	PIDs []int
}

func (e *Running) Error() string { return i18n.T("wizard.game_running", e.Name, e.PIDs) }

// gameNames are the processes that mean the game runs: the launcher (busy
// updating, it counts as the game) and the followed process.
func (r *Runner) gameNames() []string {
	cmd, track := r.W.Launch()
	names := []string{filepath.Base(cmd)}
	if filepath.Ext(names[0]) == "" {
		names[0] += ".exe"
	}
	if track != "" && !strings.EqualFold(track, names[0]) {
		names = append(names, track)
	}
	return names
}

// checkGame returns a *Running when the game runs.
func (r *Runner) checkGame() error {
	for _, name := range r.gameNames() {
		pids, err := proc.FindByName(name)
		if err != nil {
			return err
		}
		if len(pids) > 0 {
			return &Running{Name: name, PIDs: pids}
		}
	}
	return nil
}

// StopGame stops the game e found, once the player has asked for it, and
// waits until it is gone. By name, as it is now: the player may answer
// minutes after e was found, and a PID found then may belong to another
// program since; KillByName checks each process's name before stopping it.
func (r *Runner) StopGame(e *Running) error {
	if _, err := proc.KillByName(e.Name, 10*time.Second); err != nil {
		return errors.Join(i18n.Errorf("wizard.stop_failed", e.Name, e.PIDs), err)
	}
	return nil
}

// learn puts the original files back, plays the game for profile and keeps
// what changed: path → hash of the new content, "" for a removed file.
func (r *Runner) learn(ctx context.Context, profile string) (map[string]string, error) {
	dir := r.W.WizardDir()
	if err := Restore(dir); err != nil {
		return nil, err
	}
	if err := r.seed(profile); err != nil {
		return nil, err
	}
	if err := r.play(ctx); err != nil {
		return nil, err
	}
	// Interrupted: the game may still be running, what it left proves nothing.
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m, err := readManifest(dir)
	if err != nil {
		return nil, err
	}
	after, err := Take(m.Roots, m.Ignore)
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	var kept []string
	for p, present := range Changes(m.Original, after) {
		out[p] = ""
		if present {
			out[p] = after[p]
			kept = append(kept, p)
		}
	}
	return out, Store(r.W.ProfileStore(profile), kept)
}

// seed copies the files the player chose into the game, before it starts
// for profile. A file outside the watched folders, or missing, is left out
// and the log says so: the player sees it before the game starts.
func (r *Runner) seed(profile string) error {
	s := r.W.Seed(profile)
	var copies []FileCopy
	outside := 0
	switch {
	case s.Dir != "":
		err := filepath.WalkDir(s.Dir, func(p string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			rel, _ := filepath.Rel(s.Dir, p)
			first, rest, _ := strings.Cut(rel, string(filepath.Separator))
			for _, root := range r.W.Roots() {
				if rest != "" && strings.EqualFold(filepath.Base(root), first) {
					copies = append(copies, FileCopy{From: p, To: filepath.Join(root, rest)})
					return nil
				}
			}
			outside++
			return nil
		})
		if err != nil {
			return err
		}
	case s.Config != "":
		c, err := config.Load(s.Config, s.Profile)
		if err != nil {
			return err
		}
		for _, pair := range c.Sync {
			copies = append(copies, FileCopy{From: underConfig(pair.Source, filepath.Dir(c.Path)), To: pair.Target})
		}
	default:
		copies = s.Copies
	}
	for _, c := range copies {
		if !r.W.inRoots(c.To) {
			outside++
			continue
		}
		if _, err := os.Stat(c.From); err != nil {
			r.Log.Warn(i18n.T("wizard.seed_missing", c.From))
			continue
		}
		if err := copyFile(c.From, c.To); err != nil {
			return err
		}
		r.Log.Info(i18n.T("wizard.seeded", c.From, c.To))
	}
	if outside > 0 {
		r.Log.Warn(i18n.T("wizard.seed_outside", outside))
	}
	return nil
}

// underConfig finds a profile file an older config names, next to that
// config first: the config imported is usually a backup moved elsewhere,
// whose paths still point where it used to live. The longest end of the
// path found under dir wins (...\Profiles\STD\options.lua is dir\STD\options.lua);
// with none, the path as written.
func underConfig(path, dir string) string {
	parts := strings.Split(filepath.Clean(path), string(filepath.Separator))
	for i := 1; i < len(parts); i++ {
		candidate := filepath.Join(dir, filepath.Join(parts[i:]...))
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return path
}

// play starts the game the way pm run does, and returns once it is over.
func (r *Runner) play(ctx context.Context) error {
	cmd, track := r.W.Launch()
	y := "name: " + quote(r.W.one("name")) + "\ncommand: " + quote(cmd) + "\n"
	if track != "" {
		y += "track: " + quote(track) + "\n"
	}
	if r.W.one("srs") == "yes" {
		y += "kill_on_exit: [SR-ClientRadio.exe]\n"
	}
	f := filepath.Join(r.W.WizardDir(), "play.yaml")
	if err := os.WriteFile(f, []byte(y), 0o644); err != nil {
		return err
	}
	c, err := config.Load(f, "")
	if err != nil {
		return err
	}
	return session.New(c, r.Log, config.RestartNever).Run(ctx)
}

// apply writes the config and the profile files, then lists the shortcuts.
// The config is checked by pm itself before it takes its place: a wizard
// that wrote a config pm refuses would be a bug.
func (r *Runner) apply() ([]shortcut.Link, error) {
	p := r.W.Plan()
	if _, err := os.Stat(p.ConfigPath); err == nil {
		return nil, i18n.Errorf("wizard.config_exists", p.ConfigPath)
	}
	if err := os.MkdirAll(filepath.Dir(p.ConfigPath), 0o755); err != nil {
		return nil, err
	}
	tmp := filepath.Join(filepath.Dir(p.ConfigPath), "."+filepath.Base(p.ConfigPath)+".new")
	if err := os.WriteFile(tmp, []byte(p.YAML), 0o644); err != nil {
		return nil, err
	}
	profiles := p.Profiles
	if len(profiles) == 0 {
		profiles = []string{""}
	}
	for _, prof := range profiles {
		if _, err := config.Load(tmp, prof); err != nil {
			os.Remove(tmp)
			return nil, err
		}
	}
	// The profile folders are new (checked when the profiles were named,
	// again here): on failure, removing them takes nothing that was there.
	for _, prof := range p.Profiles {
		if _, err := os.Stat(r.W.profileDir(prof)); err == nil {
			os.Remove(tmp)
			return nil, i18n.Errorf("wizard.profile_exists", r.W.profileDir(prof))
		}
	}
	undo := func() {
		os.Remove(tmp)
		for _, prof := range p.Profiles {
			os.RemoveAll(r.W.profileDir(prof))
		}
	}
	var errs []error
	for _, f := range p.Files {
		errs = append(errs, copyFile(f.From, f.To))
	}
	if err := errors.Join(errs...); err != nil {
		undo()
		return nil, err
	}
	if err := os.Rename(tmp, p.ConfigPath); err != nil {
		undo()
		return nil, err
	}
	// The config is in place: what follows can no longer fail the wizard.
	// A .pm-new left behind holds nothing the next pm new would act on,
	// since the config it belongs to now exists.
	if p.WizardDir != "" {
		os.RemoveAll(p.WizardDir)
	}
	return shortcut.ForConfig(p.ConfigPath, "", r.PM)
}
