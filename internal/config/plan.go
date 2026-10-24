package config

import (
	"os"
	"strings"

	"github.com/davidp57/process-monitor/internal/i18n"
)

// Overview is what a session of this config would do, as data: what pm
// check prints and what the interface draws.
type Overview struct {
	Config    string       `json:"config"`
	Profile   string       `json:"profile"`
	Name      string       `json:"name"`
	Shared    bool         `json:"shared"` // one lock for every profile
	Command   string       `json:"command"`
	Args      []string     `json:"args"`
	Workdir   string       `json:"workdir"`
	Follows   string       `json:"follows"`
	IfRunning string       `json:"ifRunning"`
	Ready     *ReadyView   `json:"ready"`
	LogDir    string       `json:"logDir"`
	LogKeep   int          `json:"logKeep"`
	Sync      []SyncView   `json:"sync"`
	Before    []ActionView `json:"before"`
	After     []ActionView `json:"after"`
	// Missing counts the sources missing: pm run would fail before
	// launching anything.
	Missing int `json:"missing"`
}

type ReadyView struct {
	File    string `json:"file"`
	Timeout string `json:"timeout"`
}

// SyncView is one sync pair: Source copied to Target before the session,
// Target back to Source after.
type SyncView struct {
	Source    string `json:"source"`
	Target    string `json:"target"`
	SourceDir bool   `json:"sourceDir"`
	TargetDir bool   `json:"targetDir"`
	Missing   bool   `json:"missing"`
}

// ActionView is one before or after action. Kind is copy (From, To),
// delete (Path) or exec (Args).
type ActionView struct {
	Kind    string   `json:"kind"`
	From    string   `json:"from,omitempty"`
	To      string   `json:"to,omitempty"`
	Path    string   `json:"path,omitempty"`
	Args    []string `json:"args,omitempty"`
	Missing bool     `json:"missing,omitempty"`
}

// Overview looks at the files the session would copy, without touching
// them.
func (c *Config) Overview() Overview {
	o := Overview{Config: c.Path, Profile: c.Profile, Name: c.Name, Shared: len(c.Profiles) > 0,
		Command: c.Command, Args: c.Args, Workdir: c.Workdir, Follows: c.ProcessName(), IfRunning: c.IfRunning,
		LogDir: c.Log.Dir, LogKeep: c.Log.Keep, Sync: []SyncView{}, Before: []ActionView{}, After: []ActionView{}}
	if c.Ready != nil {
		o.Ready = &ReadyView{File: c.Ready.File, Timeout: c.Ready.Timeout.D().String()}
	}
	for _, p := range c.Sync {
		v := SyncView{Source: p.Source, Target: p.Target}
		if st, err := os.Stat(p.Source); err != nil {
			v.Missing = true
			o.Missing++
		} else {
			v.SourceDir = st.IsDir()
		}
		if st, err := os.Stat(p.Target); err == nil {
			v.TargetDir = st.IsDir()
		}
		o.Sync = append(o.Sync, v)
	}
	for _, a := range c.Hooks.Before {
		v := actionView(a)
		if a.Copy != nil {
			if _, err := os.Stat(a.Copy.From); err != nil {
				v.Missing = true
				o.Missing++
			}
		}
		o.Before = append(o.Before, v)
	}
	for _, a := range c.Hooks.After {
		o.After = append(o.After, actionView(a))
	}
	return o
}

func actionView(a Action) ActionView {
	switch {
	case a.Copy != nil:
		return ActionView{Kind: "copy", From: a.Copy.From, To: a.Copy.To}
	case a.Delete != "":
		return ActionView{Kind: "delete", Path: a.Delete}
	default:
		return ActionView{Kind: "exec", Args: a.Exec}
	}
}

// Plan is what pm check shows: what the session would copy, delete and
// run, the source files missing marked.
func (c *Config) Plan() string { return c.Overview().Text() }

// Text is the overview as pm check prints it.
func (o Overview) Text() string {
	var b strings.Builder
	b.WriteString(i18n.T("plan.config", o.Config))
	if o.Profile != "" {
		b.WriteString(i18n.T("plan.profile", o.Profile))
	}
	if o.Shared {
		b.WriteString(i18n.T("plan.session_shared", o.Name))
	} else {
		b.WriteString(i18n.T("plan.session", o.Name))
	}
	b.WriteString(i18n.T("plan.launches", o.Command, strings.Join(o.Args, " ")))
	if o.Workdir != "" {
		b.WriteString(i18n.T("plan.workdir", o.Workdir))
	}
	b.WriteString(i18n.T("plan.follows", o.Follows))
	b.WriteString(i18n.T("plan.if_running", o.IfRunning))
	if o.Ready != nil {
		b.WriteString(i18n.T("plan.ready", o.Ready.File, o.Ready.Timeout))
	}
	b.WriteString(i18n.T("plan.logs", o.LogDir, o.LogKeep))
	b.WriteString("\n" + i18n.T("plan.before") + "\n")
	for _, p := range o.Sync {
		b.WriteString(i18n.T("plan.copy", p.Source, p.Target) + missingMark(p.Missing) + "\n")
	}
	for _, a := range o.Before {
		b.WriteString("  " + a.text() + missingMark(a.Missing) + "\n")
	}
	b.WriteString(i18n.T("plan.after") + "\n")
	for _, p := range o.Sync {
		b.WriteString(i18n.T("plan.copy", p.Target, p.Source) + "\n")
	}
	for _, a := range o.After {
		b.WriteString("  " + a.text() + "\n")
	}
	if o.Missing > 0 {
		b.WriteString(i18n.T("plan.missing_total", o.Missing))
	}
	return b.String()
}

func missingMark(missing bool) string {
	if missing {
		return i18n.T("plan.source_missing")
	}
	return ""
}

func (a ActionView) text() string {
	switch a.Kind {
	case "copy":
		return strings.TrimPrefix(i18n.T("plan.copy", a.From, a.To), "  ")
	case "delete":
		return i18n.T("plan.delete", a.Path)
	default:
		return "exec   " + strings.Join(a.Args, " ")
	}
}
