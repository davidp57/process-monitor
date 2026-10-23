// Package config loads a pm YAML file, applies the selected profile and
// expands ${...} variables. Loading is strict: unknown keys are errors, so a
// typo never silently disables an action.
package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"
)

// Config is a fully loaded, profile-applied and expanded configuration.
type Config struct {
	Name      string             `yaml:"name"`
	Vars      map[string]string  `yaml:"vars"`
	Command   string             `yaml:"command"`
	Args      []string           `yaml:"args"`
	Workdir   string             `yaml:"workdir"`
	Env       map[string]string  `yaml:"env"`
	Track     string             `yaml:"track"`
	IfRunning string             `yaml:"if_running"`
	Capture   bool               `yaml:"capture"`
	Ready     *Ready             `yaml:"ready"`
	Hooks     Hooks              `yaml:"hooks"`
	Backup    *bool              `yaml:"backup"`
	Log       Log                `yaml:"log"`
	Watch     Watch              `yaml:"watch"`
	Exit      Exit               `yaml:"exit"`
	Profiles  map[string]Profile `yaml:"profiles"`

	// Filled by Load, not by YAML.
	Profile string     `yaml:"-"`
	Path    string     `yaml:"-"`
	Sync    []SyncPair `yaml:"-"`
}

// Ready describes the file whose appearance means the program has finished starting.
type Ready struct {
	File    string   `yaml:"file"`
	Timeout Duration `yaml:"timeout"`
}

// Hooks are the actions run around a session.
type Hooks struct {
	Before []Action `yaml:"before"`
	After  []Action `yaml:"after"`
	// Sync lines "a <-> b": a is copied to b before, b back to a after.
	Sync []string `yaml:"sync"`
	// Timeout bounds each exec action (default 10m).
	Timeout Duration `yaml:"timeout"`
}

// Action is exactly one of Copy, Delete or Exec.
type Action struct {
	Copy   *Copy    `yaml:"copy"`
	Delete string   `yaml:"delete"`
	Exec   []string `yaml:"exec"`
}

type Copy struct {
	From string `yaml:"from"`
	To   string `yaml:"to"`
}

// SyncPair is a parsed sync line.
type SyncPair struct {
	Source string // profile side, copied in before the session
	Target string // live side, copied back after the session
}

type Log struct {
	Dir  string `yaml:"dir"`
	Keep int    `yaml:"keep"`
}

type Watch struct {
	Restart     string   `yaml:"restart"`
	Delay       Duration `yaml:"delay"`
	MaxDelay    Duration `yaml:"max_delay"`
	MaxRestarts int      `yaml:"max_restarts"`
}

type Exit struct {
	Delay   Duration `yaml:"delay"`
	WaitKey string   `yaml:"wait_key"`
}

// Profile overrides vars and appends hooks.
type Profile struct {
	Vars  map[string]string `yaml:"vars"`
	Hooks Hooks             `yaml:"hooks"`
}

const (
	IfRunningAbort = "abort"
	IfRunningKill  = "kill"

	RestartAlways    = "always"
	RestartOnFailure = "on-failure"
	RestartNever     = "never"

	WaitKeyNever   = "never"
	WaitKeyOnError = "on-error"
	WaitKeyAlways  = "always"
)

// Duration accepts Go duration strings ("2s", "1m30s") in YAML.
type Duration time.Duration

func (d *Duration) UnmarshalYAML(n *yaml.Node) error {
	v, err := time.ParseDuration(n.Value)
	if err != nil {
		return fmt.Errorf("ligne %d : durée invalide %q (exemples : 500ms, 2s, 1m)", n.Line, n.Value)
	}
	*d = Duration(v)
	return nil
}

func (d Duration) D() time.Duration { return time.Duration(d) }

// Load reads path, applies profile (may be empty) and returns a validated config.
func Load(path, profile string) (*Config, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return nil, err
	}
	var c Config
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&c); err != nil {
		return nil, fmt.Errorf("%s : %w", abs, err)
	}
	var extra yaml.Node
	if err := dec.Decode(&extra); err != io.EOF {
		return nil, fmt.Errorf("%s : un seul document YAML attendu (séparateur --- en trop ?)", abs)
	}
	c.Path = abs
	if err := c.applyProfile(profile); err != nil {
		return nil, err
	}
	c.defaults()
	if err := c.expand(); err != nil {
		return nil, err
	}
	if err := c.validate(); err != nil {
		return nil, fmt.Errorf("%s : %w", abs, err)
	}
	return &c, nil
}

func (c *Config) applyProfile(name string) error {
	if len(c.Profiles) == 0 {
		if name != "" {
			return fmt.Errorf("profil %q demandé, mais %s ne définit aucun profil", name, c.Path)
		}
		return nil
	}
	if name == "" {
		return fmt.Errorf("%s définit des profils, choisis-en un avec --profile : %s", c.Path, strings.Join(c.ProfileNames(), ", "))
	}
	p, ok := c.Profiles[name]
	if !ok {
		return fmt.Errorf("profil %q inconnu, profils disponibles : %s", name, strings.Join(c.ProfileNames(), ", "))
	}
	if c.Vars == nil {
		c.Vars = map[string]string{}
	}
	for k, v := range p.Vars {
		c.Vars[k] = v
	}
	c.Hooks.Before = append(c.Hooks.Before, p.Hooks.Before...)
	c.Hooks.After = append(c.Hooks.After, p.Hooks.After...)
	c.Hooks.Sync = append(c.Hooks.Sync, p.Hooks.Sync...)
	c.Profile = name
	return nil
}

// ProfileNames returns the sorted profile names.
func (c *Config) ProfileNames() []string {
	names := make([]string, 0, len(c.Profiles))
	for n := range c.Profiles {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

func (c *Config) defaults() {
	if c.Name == "" {
		c.Name = strings.TrimSuffix(filepath.Base(c.Path), filepath.Ext(c.Path))
	}
	if c.IfRunning == "" {
		c.IfRunning = IfRunningAbort
	}
	if c.Log.Dir == "" {
		c.Log.Dir = "${config_dir}/logs"
	}
	if c.Log.Keep == 0 {
		c.Log.Keep = 5
	}
	if c.Watch.Restart == "" {
		c.Watch.Restart = RestartAlways
	}
	if c.Watch.Delay == 0 {
		c.Watch.Delay = Duration(2 * time.Second)
	}
	if c.Watch.MaxDelay == 0 {
		c.Watch.MaxDelay = Duration(time.Minute)
	}
	if c.Hooks.Timeout == 0 {
		c.Hooks.Timeout = Duration(10 * time.Minute)
	}
	if c.Exit.WaitKey == "" {
		c.Exit.WaitKey = WaitKeyOnError
	}
	if c.Ready != nil && c.Ready.Timeout == 0 {
		c.Ready.Timeout = Duration(2 * time.Minute)
	}
}

// BackupEnabled reports whether overwritten files get a .bak copy (default true).
func (c *Config) BackupEnabled() bool { return c.Backup == nil || *c.Backup }

func (c *Config) validate() error {
	var errs []error
	if c.Command == "" {
		errs = append(errs, errors.New("command est obligatoire"))
	}
	switch c.IfRunning {
	case IfRunningAbort, IfRunningKill:
	default:
		errs = append(errs, fmt.Errorf("if_running : %q invalide (abort ou kill)", c.IfRunning))
	}
	switch c.Watch.Restart {
	case RestartAlways, RestartOnFailure, RestartNever:
	default:
		errs = append(errs, fmt.Errorf("watch.restart : %q invalide (always, on-failure ou never)", c.Watch.Restart))
	}
	switch c.Exit.WaitKey {
	case WaitKeyNever, WaitKeyOnError, WaitKeyAlways:
	default:
		errs = append(errs, fmt.Errorf("exit.wait_key : %q invalide (never, on-error ou always)", c.Exit.WaitKey))
	}
	for name, d := range map[string]Duration{"watch.delay": c.Watch.Delay, "watch.max_delay": c.Watch.MaxDelay,
		"exit.delay": c.Exit.Delay, "hooks.timeout": c.Hooks.Timeout} {
		if d < 0 {
			errs = append(errs, fmt.Errorf("%s : durée négative", name))
		}
	}
	if c.Ready != nil && c.Ready.Timeout < 0 {
		errs = append(errs, errors.New("ready.timeout : durée négative"))
	}
	if c.Ready != nil && c.Ready.File == "" {
		errs = append(errs, errors.New("ready.file est obligatoire quand ready est présent"))
	}
	for i, a := range append(append([]Action{}, c.Hooks.Before...), c.Hooks.After...) {
		if err := a.validate(); err != nil {
			errs = append(errs, fmt.Errorf("action %d : %w", i+1, err))
		}
	}
	if c.Track != "" && (strings.ContainsAny(c.Track, `\/`) || filepath.Ext(c.Track) == "") {
		errs = append(errs, fmt.Errorf("track : %q doit être un nom de process seul, extension comprise (ex. dcs.exe)", c.Track))
	}
	c.Sync = nil
	targets, sources := map[string]string{}, map[string]string{}
	for _, line := range c.Hooks.Sync {
		a, b, ok := strings.Cut(line, "<->")
		a, b = strings.TrimSpace(a), strings.TrimSpace(b)
		if !ok || a == "" || b == "" {
			errs = append(errs, fmt.Errorf("sync : %q n'a pas la forme \"profil <-> live\"", line))
			continue
		}
		pair := SyncPair{Source: c.abs(a), Target: c.abs(b)}
		// Two sources for one live file: the copy back would overwrite both with the same content.
		key := strings.ToLower(pair.Target)
		if prev, dup := targets[key]; dup {
			errs = append(errs, fmt.Errorf("sync : %s est la cible de deux lignes (%s et %s)", pair.Target, prev, pair.Source))
		}
		targets[key] = pair.Source
		// Two live files for one profile file: the copy back would keep only the last one.
		if prev, dup := sources[strings.ToLower(pair.Source)]; dup {
			errs = append(errs, fmt.Errorf("sync : %s est la source de deux lignes (vers %s et %s)", pair.Source, prev, pair.Target))
		}
		sources[strings.ToLower(pair.Source)] = pair.Target
		c.Sync = append(c.Sync, pair)
	}
	return errors.Join(errs...)
}

func (a Action) validate() error {
	n := 0
	if a.Copy != nil {
		n++
		if a.Copy.From == "" || a.Copy.To == "" {
			return errors.New("copy demande from et to")
		}
	}
	if a.Delete != "" {
		n++
	}
	if len(a.Exec) > 0 {
		n++
	}
	if n != 1 {
		return errors.New("une action contient exactement une clé parmi copy, delete, exec")
	}
	return nil
}

// ProcessName is the name used to detect an already running instance:
// the tracked process if any, else the executable's file name.
func (c *Config) ProcessName() string {
	if c.Track != "" {
		return c.Track
	}
	return c.LauncherName()
}

// LauncherName is the image name of the launched executable. Windows lists
// processes with their extension, so "notepad" is looked up as "notepad.exe".
func (c *Config) LauncherName() string {
	name := filepath.Base(c.Command)
	if filepath.Ext(name) == "" {
		name += ".exe"
	}
	return name
}
