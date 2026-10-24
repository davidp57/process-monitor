package wizard

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/davidp57/process-monitor/internal/i18n"
)

// noIcon and otherIcon answer "which icon for this profile" with none, or
// with a path typed next. Icon paths are absolute, so neither collides.
const (
	noIcon    = "none"
	otherIcon = "other"

	IconNone  = noIcon
	IconOther = otherIcon
)

// ProfileIcon is the icon chosen for profile, or "".
func (w *Wizard) ProfileIcon(profile string) string {
	switch v := w.one("icon:" + profile); v {
	case noIcon, "":
		return ""
	case otherIcon:
		return w.one("icon:" + profile + ":other")
	default:
		return v
	}
}

// iconSteps asks for profile's icon among the .ico files found next to the
// config, the one named after the profile suggested (dcs-dev.ico for DEV).
func (w *Wizard) iconSteps(p string, ask func(string, Question) bool) bool {
	var found []string
	if w.env.Icons != nil {
		found = w.env.Icons(filepath.Dir(w.one("config")))
	}
	if !ask("icon:"+p, IconQuestion(p, w.one("name"), filepath.Dir(w.one("config")), found)) {
		return false
	}
	if w.one("icon:"+p) == otherIcon {
		return ask("icon:"+p+":other", IconPath(p, w.env.Exists))
	}
	return true
}

// IconQuestion asks which icon profile shows, among the icons found in dir
// (the config's folder), the one named after the profile suggested; or
// none (IconNone), or a file typed next (IconOther, then IconPath).
func IconQuestion(profile, config, dir string, found []string) Question {
	opts := make([]Option, 0, len(found)+2)
	for _, f := range found {
		label := f
		if rel, err := filepath.Rel(dir, f); err == nil && !strings.HasPrefix(rel, "..") {
			label = rel
		}
		opts = append(opts, Option{f, label})
	}
	opts = append(opts, Option{noIcon, i18n.T("wizard.icon.none")}, Option{otherIcon, i18n.T("wizard.icon.other")})
	return Question{Kind: Choice, Prompt: i18n.T("wizard.q.icon", profile), Help: i18n.T("wizard.h.icon"),
		Options: opts, Default: []string{iconFor(profile, config, found)}}
}

// IconPath asks for an icon file typed by hand.
func IconPath(profile string, exists func(string) bool) Question {
	return Question{Kind: Path, Prompt: i18n.T("wizard.q.icon", profile), Pick: "icon", check: func(values []string) error {
		if !filepath.IsAbs(values[0]) {
			return i18n.Errorf("wizard.not_absolute", values[0])
		}
		if !exists(values[0]) {
			return i18n.Errorf("wizard.not_found", values[0])
		}
		if ext := strings.ToLower(filepath.Ext(values[0])); ext != ".ico" && ext != ".png" {
			return i18n.Errorf("wizard.bad_icon", values[0])
		}
		return nil
	}}
}

// Check runs a question's own check on an answer, for a front that asks
// it outside the wizard.
func (q Question) Check(values []string) error {
	if q.check == nil {
		return nil
	}
	return q.check(values)
}

// iconFor is the icon named after the profile: the profile's name is a
// word of the icon's (dcs-dev.ico for DEV), and when some icons also carry
// the config's name, only those count (sound-VR.ico is not DCS's VR). The
// shortest wins (dcs-dev.ico before dcs-dev-old.ico); none when no icon fits.
func iconFor(profile, config string, icons []string) string {
	words := func(f string) map[string]bool {
		set := map[string]bool{}
		for _, w := range strings.FieldsFunc(strings.ToLower(strings.TrimSuffix(filepath.Base(f), filepath.Ext(f))),
			func(r rune) bool { return r == '-' || r == '_' || r == '.' || r == ' ' }) {
			set[w] = true
		}
		return set
	}
	profile, config = strings.ToLower(profile), strings.ToLower(config)
	named := false
	for _, f := range icons {
		named = named || words(f)[config]
	}
	var match []string
	for _, f := range icons {
		if w := words(f); w[profile] && (!named || w[config]) {
			match = append(match, f)
		}
	}
	if len(match) == 0 {
		return noIcon
	}
	sort.SliceStable(match, func(i, j int) bool { return len(filepath.Base(match[i])) < len(filepath.Base(match[j])) })
	return match[0]
}
