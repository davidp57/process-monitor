package shortcut

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/davidp57/process-monitor/internal/config"
	"github.com/davidp57/process-monitor/internal/i18n"
)

// SetIcon gives profile (or the config, for "") the icon it does not have
// yet, touching the file as little as possible: "DEV: {}" becomes "DEV:"
// and an icon line, a profile written as a block gets an icon line, a
// config without profiles gets one at its end. The rest of the file stays
// as it is. If pm does not read the icon back, the file is put back.
func SetIcon(path, profile, icon string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines, ok := insertIcon(strings.Split(string(data), "\n"), profile, "'"+strings.ReplaceAll(icon, "'", "''")+"'")
	if !ok {
		return i18n.Errorf("shortcut.icon_not_written", profile, path)
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		return err
	}
	c, err := config.Load(path, profile)
	if err == nil && !strings.EqualFold(c.Icon, filepath.Clean(icon)) {
		err = i18n.Errorf("shortcut.icon_not_written", profile, path)
	}
	if err != nil {
		return errors.Join(err, os.WriteFile(path, data, 0o644))
	}
	return nil
}

// appendKey adds a top-level line at the end of the config, before its
// trailing blank lines.
func appendKey(lines []string, line string) []string {
	end := len(lines)
	for end > 0 && strings.TrimSpace(lines[end-1]) == "" {
		end--
	}
	return append(lines[:end], append([]string{line}, lines[end:]...)...)
}

// HasUI reports whether the config at path says how a session shows itself
// (the ui key). One that does not load says yes: pm says what is wrong.
func HasUI(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return true
	}
	var top map[string]any
	if err := yaml.Unmarshal(data, &top); err != nil {
		return true
	}
	_, ok := top["ui"]
	return ok
}

// SetUI adds the ui key at the end of the config at path, the rest of the
// file as it is; put back if pm does not read it back.
func SetUI(path, ui string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := appendKey(strings.Split(string(data), "\n"), "ui: "+ui)
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		return err
	}
	profiles, _ := config.Profiles(path)
	first := ""
	if len(profiles) > 0 {
		first = profiles[0]
	}
	c, err := config.Load(path, first)
	if err == nil && c.UI != ui {
		err = i18n.Errorf("shortcut.ui_not_written", path)
	}
	if err != nil {
		return errors.Join(err, os.WriteFile(path, data, 0o644))
	}
	return nil
}

func indent(line string) int { return len(line) - len(strings.TrimLeft(line, " ")) }

// insertIcon adds "icon: value" to profile in the lines of a config.
func insertIcon(lines []string, profile, value string) ([]string, bool) {
	insert := func(at int, line string) []string {
		return append(lines[:at], append([]string{line}, lines[at:]...)...)
	}
	if profile == "" {
		return appendKey(lines, "icon: "+value), true
	}
	in := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if indent(line) == 0 {
			in = strings.HasPrefix(line, "profiles:")
			continue
		}
		key, rest, found := strings.Cut(trimmed, ":")
		if !in || !found || strings.Trim(key, `'"`) != profile {
			continue
		}
		ind := indent(line)
		switch rest = strings.TrimSpace(rest); rest {
		case "{}":
			lines[i] = line[:ind] + key + ":"
			return insert(i+1, strings.Repeat(" ", ind+2)+"icon: "+value), true
		case "":
			child := ind + 2
			for _, next := range lines[i+1:] {
				if t := strings.TrimSpace(next); t != "" && !strings.HasPrefix(t, "#") {
					if indent(next) > ind {
						child = indent(next)
					}
					break
				}
			}
			return insert(i+1, strings.Repeat(" ", child)+"icon: "+value), true
		}
		return lines, false // a profile written inline: by hand
	}
	return lines, false
}
