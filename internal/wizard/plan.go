package wizard

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/davidp57/process-monitor/internal/i18n"
	"github.com/davidp57/process-monitor/internal/shortcut"
)

// Plan is what Apply writes.
type Plan struct {
	ConfigPath string
	YAML       string
	Profiles   []string   // the profiles to check the config with; none without profiles
	Files      []FileCopy // the profile files
	WizardDir  string     // removed once done; "" when the wizard kept nothing
}

// FileCopy writes one profile file from a copy the wizard kept.
type FileCopy struct{ From, To string }

// ProfileStore is where Learn keeps the files a profile changed.
func (w *Wizard) ProfileStore(profile string) string {
	return filepath.Join(w.WizardDir(), "profiles", profile)
}

func (w *Wizard) originalStore() string { return filepath.Join(w.WizardDir(), "original") }

// changes is what a profile changed and the user kept: a file unticked in
// its report keeps the original version in that profile.
func (w *Wizard) changes(profile string) map[string]string {
	all := w.learned[profile]
	if !w.has("review:" + profile) {
		return all
	}
	kept := map[string]string{}
	for _, f := range w.answers["review:"+profile] {
		if h, ok := all[f]; ok {
			kept[f] = h
		}
	}
	return kept
}

// synced is every file a profile changed, created or removed, and kept: a
// file carried in one profile is carried in all, each with its version.
func (w *Wizard) synced() []string {
	set := map[string]string{}
	for _, p := range w.answers["profiles"] {
		for f, h := range w.changes(p) {
			set[f] = h
		}
	}
	return sortedKeys(set)
}

func sortedKeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// hash is a profile's version of a file: what it left after playing, else
// the original; "" when the profile has no such file.
func (w *Wizard) hash(profile, f string) string {
	if h, ok := w.changes(profile)[f]; ok {
		return h
	}
	return w.original[f]
}

// source is the kept copy of a profile's version of a file.
func (w *Wizard) source(profile, f string) string {
	if _, ok := w.changes(profile)[f]; ok {
		return storePath(w.ProfileStore(profile), f)
	}
	return storePath(w.originalStore(), f)
}

// sub is where a live file goes inside a profile folder: under the name of
// the watched folder it is in, Config\options.lua.
func (w *Wizard) sub(f string) string {
	roots := w.Roots()
	used := map[string]int{}
	for _, r := range roots {
		name := filepath.Base(r)
		used[strings.ToLower(name)]++
		if n := used[strings.ToLower(name)]; n > 1 {
			name = fmt.Sprintf("%s-%d", name, n)
		}
		if rel, err := filepath.Rel(r, f); err == nil && !strings.HasPrefix(rel, "..") {
			return filepath.Join(name, rel)
		}
	}
	return filepath.Base(f)
}

var guid = regexp.MustCompile(`\s*\{[0-9A-Fa-f-]+\}`)

// label is a file's name for a player: the DCS files pm knows by what they
// hold, the others by their place.
func (w *Wizard) label(f string) string {
	sub := w.sub(f)
	if w.one("case") == CaseDCS {
		parts := strings.Split(sub, string(filepath.Separator))
		switch {
		case strings.EqualFold(sub, `Config\options.lua`):
			return i18n.T("wizard.file.options")
		case strings.EqualFold(sub, `Scripts\Export.lua`):
			return i18n.T("wizard.file.export")
		case len(parts) == 5 && strings.EqualFold(parts[1], "Input") && strings.HasSuffix(strings.ToLower(parts[4]), ".diff.lua"):
			device := guid.ReplaceAllString(parts[4][:len(parts[4])-len(".diff.lua")], "")
			return i18n.T("wizard.file.input", device, parts[2])
		}
	}
	return sub
}

// profileSummary says, profile by profile, which files it carries: the
// ones the same for every profile, then each profile's own.
func (w *Wizard) profileSummary() string {
	profiles := w.answers["profiles"]
	var common []string
	own := map[string][]string{}
	without := map[string][]string{}
	for _, f := range w.synced() {
		same := true
		for _, p := range profiles {
			if w.hash(p, f) != w.hash(profiles[0], f) {
				same = false
			}
		}
		if same && w.hash(profiles[0], f) != "" {
			common = append(common, w.label(f))
			continue
		}
		for _, p := range profiles {
			if w.hash(p, f) == "" {
				without[p] = append(without[p], w.label(f))
			} else {
				own[p] = append(own[p], w.label(f))
			}
		}
	}
	var b strings.Builder
	if len(common) > 0 {
		b.WriteString(i18n.T("wizard.sum.common", strings.Join(common, ", ")) + "\n")
	}
	for _, p := range profiles {
		how := i18n.T("wizard.sum.played")
		if w.one("current") == p {
			how = i18n.T("wizard.sum.current")
		}
		b.WriteString(i18n.T("wizard.sum.profile", p, how) + "\n")
		if len(own[p]) > 0 {
			b.WriteString(i18n.T("wizard.sum.own", strings.Join(own[p], ", ")) + "\n")
		}
		if len(without[p]) > 0 {
			b.WriteString(i18n.T("wizard.sum.without", strings.Join(without[p], ", ")) + "\n")
		}
	}
	return b.String()
}

// Summary is the whole config in plain words, for the last confirmation.
func (w *Wizard) Summary() string {
	var b strings.Builder
	line := func(key string, args ...any) { b.WriteString(i18n.T(key, args...) + "\n") }
	line(i18n.T("wizard.sum.config", w.one("config")))
	switch w.one("case") {
	case CaseDCS, CaseGame:
		cmd, track := w.Launch()
		line(i18n.T("wizard.sum.launches", cmd))
		if track != "" {
			line(i18n.T("wizard.sum.follows", track))
		}
		if w.one("srs") == "yes" {
			line(i18n.T("wizard.sum.srs"))
		}
		line(i18n.T("wizard.sum.profiles", strings.Join(w.answers["profiles"], ", "), len(w.synced())))
		b.WriteString(w.profileSummary())
	case CaseWatch:
		line(i18n.T("wizard.sum.keeps", w.one("program")))
		if w.one("kill_running") == "yes" {
			line(i18n.T("wizard.sum.kill_running"))
		}
		for _, c := range w.answers["clean"] {
			line(i18n.T("wizard.sum.cleans", c))
		}
	case CaseBackup:
		line(i18n.T("wizard.sum.launches", w.one("program")))
		line(i18n.T("wizard.sum.backup", w.one("saves"), w.one("archives"), w.one("keep")))
	}
	line(i18n.T("wizard.sum.shortcuts", strings.Join(w.shortcutFolders(), ", ")))
	return b.String()
}

func (w *Wizard) shortcutFolders() []string {
	var out []string
	for _, f := range w.answers["shortcuts"] {
		if f == "other" {
			f = w.one("shortcuts_folder")
		}
		out = append(out, w.portable(f))
	}
	return out
}

// Plan is the config and the profile files to write.
func (w *Wizard) Plan() Plan {
	p := Plan{ConfigPath: w.one("config")}
	var y yamlWriter
	mode := "run"
	if w.one("case") == CaseWatch {
		mode = "watch"
	}
	launch := fmt.Sprintf(`.\pm.exe %s "%s"`, mode, p.ConfigPath)
	if w.profiled() {
		launch += " --profile " + w.answers["profiles"][0]
	}
	y.raw(i18n.T("wizard.yaml_header", w.env.Now().Format("2006-01-02"), launch))
	y.kv("name", w.one("name"))
	if w.env.Lang != "" {
		y.kv("lang", w.env.Lang)
	}
	switch w.one("case") {
	case CaseDCS, CaseGame:
		w.profileYAML(&y, &p)
	case CaseWatch:
		prog := w.one("program")
		y.kv("command", w.portable(prog))
		y.kv("workdir", w.portable(filepath.Dir(prog)))
		if w.one("kill_running") == "yes" {
			y.raw("if_running: kill\n")
		}
		y.raw("ui: tray\n")
		y.raw("watch: { delay: " + w.one("delay") + " }\n")
		if clean := w.answers["clean"]; len(clean) > 0 {
			y.raw("hooks:\n  before:\n")
			for _, c := range clean {
				y.raw("    - delete: " + quote(w.portable(c)) + "\n")
			}
		}
	case CaseBackup:
		y.raw("vars:\n")
		y.raw("  saves: " + quote(w.portable(w.one("saves"))) + "\n")
		y.raw("  archives: " + quote(w.portable(w.one("archives"))) + "\n")
		y.kv("command", w.portable(w.one("program")))
		y.raw("hooks:\n  before:\n")
		y.raw(`    - exec: [powershell, -NoProfile, -Command, 'New-Item -ItemType Directory -Force ''${archives}'' | Out-Null; Compress-Archive -Path ''${saves}\*'' -DestinationPath (''${archives}\saves-'' + (Get-Date -Format yyyyMMdd-HHmmss) + ''.zip'')']` + "\n")
		y.raw("  after:\n")
		y.raw(fmt.Sprintf(`    - exec: [powershell, -NoProfile, -Command, 'Get-ChildItem ''${archives}\saves-*.zip'' | Sort-Object Name -Descending | Select-Object -Skip %s | Remove-Item']`, w.one("keep")) + "\n")
		y.raw("  timeout: 5m\n")
	}
	y.raw("shortcuts:\n")
	y.raw("  mode: " + mode + "\n")
	y.raw("  folder: " + shortcut.FolderYAML(w.shortcutFolders()) + "\n")
	p.YAML = y.String()
	return p
}

func (w *Wizard) profileYAML(y *yamlWriter, p *Plan) {
	profiles := w.answers["profiles"]
	p.Profiles = profiles
	p.WizardDir = w.WizardDir()
	cmd, track := w.Launch()
	y.raw("vars:\n  prof: '${config_dir}\\${profile}'\n")
	y.kv("command", w.portable(cmd))
	if track != "" {
		y.kv("track", track)
	}
	if w.one("srs") == "yes" {
		y.raw("kill_on_exit: [SR-ClientRadio.exe]\n")
	}
	y.raw("ui: tray\n")
	y.raw("log: { dir: '${prof}\\logs', keep: 3 }\n")

	var common []string
	sync := map[string][]string{}
	del := map[string][]string{}
	for _, f := range w.synced() {
		line := quote(`${prof}\` + w.sub(f) + " <-> " + w.portable(f))
		everywhere := true
		for _, prof := range profiles {
			if w.hash(prof, f) == "" {
				everywhere = false
			}
		}
		if everywhere {
			common = append(common, line)
		}
		for _, prof := range profiles {
			if w.hash(prof, f) == "" {
				del[prof] = append(del[prof], quote(w.portable(f)))
				continue
			}
			if !everywhere {
				sync[prof] = append(sync[prof], line)
			}
			p.Files = append(p.Files, FileCopy{From: w.source(prof, f), To: filepath.Join(w.profileDir(prof), w.sub(f))})
		}
	}
	if len(common) > 0 {
		y.raw("hooks:\n  sync:\n")
		for _, l := range common {
			y.raw("    - " + l + "\n")
		}
	}
	y.raw("profiles:\n")
	for _, prof := range profiles {
		icon := w.ProfileIcon(prof)
		if len(sync[prof]) == 0 && len(del[prof]) == 0 && icon == "" {
			y.raw("  " + quote(prof) + ": {}\n")
			continue
		}
		y.raw("  " + quote(prof) + ":\n")
		if icon != "" {
			y.raw("    icon: " + quote(w.portable(icon)) + "\n")
		}
		if len(sync[prof]) == 0 && len(del[prof]) == 0 {
			continue
		}
		y.raw("    hooks:\n")
		if len(sync[prof]) > 0 {
			y.raw("      sync:\n")
			for _, l := range sync[prof] {
				y.raw("        - " + l + "\n")
			}
		}
		if len(del[prof]) > 0 {
			// The file does not exist in this profile: the one another profile left must go.
			y.raw("      before:\n")
			for _, l := range del[prof] {
				y.raw("        - delete: " + l + "\n")
			}
		}
	}
}

// portable writes a path under the user's folder as ${env:USERPROFILE}\…,
// so the config still reads right if the folder is renamed.
func (w *Wizard) portable(p string) string {
	h := w.env.Home
	if h != "" && len(p) > len(h) && strings.EqualFold(p[:len(h)], h) && p[len(h)] == '\\' {
		return "${env:USERPROFILE}" + p[len(h):]
	}
	return p
}

type yamlWriter struct{ strings.Builder }

func (y *yamlWriter) raw(s string) { y.WriteString(s) }

func (y *yamlWriter) kv(k, v string) { y.WriteString(k + ": " + quote(v) + "\n") }

// quote is a YAML single-quoted string: backslashes stay as they are.
func quote(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }
