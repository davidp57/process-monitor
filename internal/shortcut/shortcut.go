// Package shortcut writes the Windows shortcuts (.lnk) a config describes
// in its shortcuts section: one per profile, each starting pm on it.
package shortcut

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/davidp57/process-monitor/internal/config"
	"github.com/davidp57/process-monitor/internal/i18n"
)

// Link is one shortcut.
type Link struct {
	Path      string // the .lnk file
	Target    string
	Args      string
	Workdir   string
	Icon      string // a .ico, or a program whose first icon is used; "" for the target's
	Minimized bool
}

// ForConfig lists the shortcuts of the config at path: every profile, or
// only profile when it is not empty. pm is the pm.exe the shortcuts start.
func ForConfig(path, profile, pm string) ([]Link, error) {
	profiles, err := config.Profiles(path)
	if err != nil {
		return nil, err
	}
	if len(profiles) == 0 || profile != "" {
		profiles = []string{profile} // Load says what is wrong with a profile that does not exist
	}
	var links []Link
	for _, p := range profiles {
		c, err := config.Load(path, p)
		if err != nil {
			return nil, err
		}
		if c.Shortcuts == nil {
			return nil, i18n.Errorf("shortcut.no_section", c.Path)
		}
		// Windows command lines quote with "…" and no escapes: %q would double the backslashes.
		args := c.Shortcuts.Mode + ` "` + c.Path + `"`
		if p != "" {
			args += " --profile " + quote(p)
		}
		for _, folder := range c.Shortcuts.Folders {
			links = append(links, Link{
				Path:      filepath.Join(folder, c.Shortcuts.Name+".lnk"),
				Target:    pm,
				Args:      args,
				Workdir:   filepath.Dir(c.Path),
				Icon:      icon(c),
				Minimized: c.UI == config.UITray,
			})
		}
	}
	return links, nil
}

func quote(s string) string {
	if strings.ContainsAny(s, " \t") {
		return `"` + s + `"`
	}
	return s
}

// HasSection reports whether the config at path has a shortcuts section.
// A config that does not load says yes: pm shortcut then says what is wrong.
func HasSection(path string) bool {
	profiles, err := config.Profiles(path)
	if err != nil {
		return true
	}
	first := ""
	if len(profiles) > 0 {
		first = profiles[0]
	}
	c, err := config.Load(path, first)
	return err != nil || c.Shortcuts != nil
}

// AddSection appends a shortcuts section at the end of the config at path,
// leaving the rest of the file as it is, comments included. When pm no
// longer loads the file, it is put back as it was.
func AddSection(path, mode string, folders []string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	text := string(data)
	if text != "" && !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	text += "\nshortcuts:\n  mode: " + mode + "\n  folder: " + FolderYAML(folders) + "\n"
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		return err
	}
	if _, err := ForConfig(path, "", "pm.exe"); err != nil {
		return errors.Join(err, os.WriteFile(path, data, 0o644))
	}
	return nil
}

// FolderYAML writes the folder key's value: one folder alone, several as a
// list, each single-quoted so backslashes stay as they are.
func FolderYAML(folders []string) string {
	quoted := make([]string, len(folders))
	for i, f := range folders {
		quoted[i] = "'" + strings.ReplaceAll(f, "'", "''") + "'"
	}
	if len(quoted) == 1 {
		return quoted[0]
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}

// icon is the config's icon when a shortcut can show it (.ico), else the
// launched program's own icon.
func icon(c *config.Config) string {
	if strings.EqualFold(filepath.Ext(c.Icon), ".ico") {
		return c.Icon
	}
	if p, err := exec.LookPath(c.Command); err == nil {
		if abs, err := filepath.Abs(p); err == nil {
			return abs
		}
	}
	return ""
}

// Write creates l.Path, keeping a shortcut already there as <name>.lnk.old
// (an older .old is overwritten). kept names that copy, or is "".
func Write(l Link) (kept string, err error) {
	if err := os.MkdirAll(filepath.Dir(l.Path), 0o755); err != nil {
		return "", err
	}
	if old, err := os.ReadFile(l.Path); err == nil {
		kept = l.Path + ".old"
		if err := os.WriteFile(kept, old, 0o644); err != nil {
			return "", err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	return kept, save(l)
}
