package config

import (
	"os"
	"strings"

	"github.com/davidp57/process-monitor/internal/i18n"
)

// Plan is what pm check shows: what the session would copy, delete and
// run, the source files missing marked. The interface shows the same.
func (c *Config) Plan() string {
	var b strings.Builder
	b.WriteString(i18n.T("plan.config", c.Path))
	if c.Profile != "" {
		b.WriteString(i18n.T("plan.profile", c.Profile))
	}
	if len(c.Profiles) > 0 {
		b.WriteString(i18n.T("plan.session_shared", c.Name))
	} else {
		b.WriteString(i18n.T("plan.session", c.Name))
	}
	b.WriteString(i18n.T("plan.launches", c.Command, strings.Join(c.Args, " ")))
	if c.Workdir != "" {
		b.WriteString(i18n.T("plan.workdir", c.Workdir))
	}
	b.WriteString(i18n.T("plan.follows", c.ProcessName()))
	b.WriteString(i18n.T("plan.if_running", c.IfRunning))
	if c.Ready != nil {
		b.WriteString(i18n.T("plan.ready", c.Ready.File, c.Ready.Timeout.D()))
	}
	b.WriteString(i18n.T("plan.logs", c.Log.Dir, c.Log.Keep))
	missing := 0
	b.WriteString("\n" + i18n.T("plan.before") + "\n")
	for _, p := range c.Sync {
		mark := ""
		if _, err := os.Stat(p.Source); err != nil {
			mark = i18n.T("plan.source_missing")
			missing++
		}
		b.WriteString(i18n.T("plan.copy", p.Source, p.Target) + mark + "\n")
	}
	for _, a := range c.Hooks.Before {
		mark := ""
		if a.Copy != nil {
			if _, err := os.Stat(a.Copy.From); err != nil {
				mark = i18n.T("plan.source_missing")
				missing++
			}
		}
		b.WriteString("  " + describe(a) + mark + "\n")
	}
	b.WriteString(i18n.T("plan.after") + "\n")
	for _, p := range c.Sync {
		b.WriteString(i18n.T("plan.copy", p.Target, p.Source) + "\n")
	}
	for _, a := range c.Hooks.After {
		b.WriteString("  " + describe(a) + "\n")
	}
	if missing > 0 {
		b.WriteString(i18n.T("plan.missing_total", missing))
	}
	return b.String()
}

func describe(a Action) string {
	switch {
	case a.Copy != nil:
		return strings.TrimPrefix(i18n.T("plan.copy", a.Copy.From, a.Copy.To), "  ")
	case a.Delete != "":
		return i18n.T("plan.delete", a.Delete)
	default:
		return "exec   " + strings.Join(a.Exec, " ")
	}
}
