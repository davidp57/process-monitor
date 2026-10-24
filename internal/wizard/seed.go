package wizard

import (
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/davidp57/process-monitor/internal/i18n"
)

// Ways to start a profile from files the player already has, before the
// game starts for it.
const (
	seedNone   = "none"
	seedFolder = "folder" // a folder laid out like the watched folders (Config\..., Scripts\...)
	seedFiles  = "files"  // loose files, each put where the player says
	seedConfig = "config" // an older pm config: its sync lines say where each file goes
)

// Seed is what to copy into the game before it starts for a profile.
type Seed struct {
	Dir             string     // seedFolder
	Copies          []FileCopy // seedFiles
	Config, Profile string     // seedConfig
}

// Seed is the seed the player chose for profile.
func (w *Wizard) Seed(profile string) Seed {
	switch w.one("seed:" + profile) {
	case seedFolder:
		return Seed{Dir: w.one("seed_dir:" + profile)}
	case seedFiles:
		var s Seed
		for i, f := range w.answers["seed_files:"+profile] {
			to := w.one(seedTo(profile, i))
			if to == otherPath {
				to = w.one(seedTo(profile, i) + ":other")
			}
			s.Copies = append(s.Copies, FileCopy{From: f, To: to})
		}
		return s
	case seedConfig:
		return Seed{Config: w.one("seed_config:" + profile), Profile: w.one("seed_profile:" + profile)}
	}
	return Seed{}
}

func seedTo(profile string, i int) string { return "seed_to:" + profile + ":" + strconv.Itoa(i) }

// otherPath answers "where does this file go" with a path typed by hand
// instead of one of the files listed. A path is absolute, so it never collides.
const otherPath = "other"

// seedSteps asks how profile starts. ask returns whether the question is
// answered; seedSteps returns false while one is not.
func (w *Wizard) seedSteps(p string, ask func(string, Question) bool) bool {
	if !ask("seed:"+p, Question{Kind: Choice, Prompt: i18n.T("wizard.q.seed", p), Help: i18n.T("wizard.h.seed"),
		Options: []Option{
			{seedNone, i18n.T("wizard.seed.none")},
			{seedConfig, i18n.T("wizard.seed.config")},
			{seedFolder, i18n.T("wizard.seed.folder", strings.Join(rootNames(w.Roots()), ", "))},
			{seedFiles, i18n.T("wizard.seed.files")},
		}, Default: []string{seedNone}}) {
		return false
	}
	switch w.one("seed:" + p) {
	case seedFolder:
		return ask("seed_dir:"+p, w.existing(i18n.T("wizard.q.seed_dir"), "", ""))
	case seedConfig:
		if !ask("seed_config:"+p, Question{Kind: Path, Prompt: i18n.T("wizard.q.seed_config"), check: w.checkOldConfig}) {
			return false
		}
		profiles, _ := w.env.Profiles(w.one("seed_config:" + p))
		if len(profiles) == 0 {
			return true
		}
		opts := make([]Option, len(profiles))
		for i, name := range profiles {
			opts[i] = Option{name, name}
		}
		def := profiles[0]
		for _, name := range profiles {
			if strings.EqualFold(name, p) {
				def = name
			}
		}
		return ask("seed_profile:"+p, Question{Kind: Choice, Prompt: i18n.T("wizard.q.seed_profile"), Options: opts, Default: []string{def}})
	case seedFiles:
		if !ask("seed_files:"+p, Question{Kind: List, Prompt: i18n.T("wizard.q.seed_files"), check: w.checkExisting}) {
			return false
		}
		for i, f := range w.answers["seed_files:"+p] {
			id, prompt := seedTo(p, i), i18n.T("wizard.q.seed_to", filepath.Base(f))
			candidates := w.candidates(f)
			if len(candidates) == 0 {
				// Nothing of that name in the game: a path to type, the first watched folder suggested.
				def := []string{filepath.Join(w.Roots()[0], filepath.Base(f))}
				if !ask(id, Question{Kind: Path, Prompt: prompt, Help: i18n.T("wizard.h.seed_to_new"), Default: def, check: w.checkInRoots}) {
					return false
				}
				continue
			}
			opts := make([]Option, 0, len(candidates)+1)
			for _, c := range candidates {
				opts = append(opts, Option{c, c})
			}
			opts = append(opts, Option{otherPath, i18n.T("wizard.seed.other_path")})
			if !ask(id, Question{Kind: Choice, Prompt: prompt, Options: opts, Default: candidates[:1]}) {
				return false
			}
			if w.one(id) == otherPath {
				if !ask(id+":other", Question{Kind: Path, Prompt: prompt, check: w.checkInRoots}) {
					return false
				}
			}
		}
	}
	return true
}

func rootNames(roots []string) []string {
	out := make([]string, len(roots))
	for i, r := range roots {
		out[i] = filepath.Base(r)
	}
	return out
}

// candidates are the files of the same name already in the game, the
// likeliest first: the one whose folders match the file's own best
// (…\F-16C_50\joystick\x.diff.lua goes to the F-16's, not the A-10's), then
// the name with the same case, then the shallowest path (Config\options.lua
// before Config\Presets\Backup\Options.lua).
func (w *Wizard) candidates(f string) []string {
	name := filepath.Base(f)
	from := strings.Split(strings.ToLower(filepath.Dir(f)), string(filepath.Separator))
	type ranked struct {
		path         string
		score, depth int
		sameCase     bool
	}
	var found []ranked
	for _, p := range w.original.Paths() {
		if !strings.EqualFold(filepath.Base(p), name) {
			continue
		}
		to := strings.Split(strings.ToLower(filepath.Dir(p)), string(filepath.Separator))
		score := 0
		for i := 1; i <= len(from) && i <= len(to) && from[len(from)-i] == to[len(to)-i]; i++ {
			score++
		}
		found = append(found, ranked{p, score, len(to), filepath.Base(p) == name})
	}
	sort.SliceStable(found, func(i, j int) bool {
		a, b := found[i], found[j]
		if a.score != b.score {
			return a.score > b.score
		}
		if a.sameCase != b.sameCase {
			return a.sameCase
		}
		return a.depth < b.depth
	})
	out := make([]string, len(found))
	for i, r := range found {
		out[i] = r.path
	}
	return out
}

// inRoots reports whether p is a file the wizard watches.
func (w *Wizard) inRoots(p string) bool {
	for _, ig := range w.Ignored() {
		if strings.EqualFold(filepath.Clean(p), filepath.Clean(ig)) {
			return false
		}
	}
	for _, r := range w.Roots() {
		if rel, err := filepath.Rel(r, p); err == nil && rel != "." && !strings.HasPrefix(rel, "..") {
			return true
		}
	}
	return false
}

func (w *Wizard) checkInRoots(values []string) error {
	if !filepath.IsAbs(values[0]) || !w.inRoots(values[0]) {
		return i18n.Errorf("wizard.not_in_roots", values[0], strings.Join(w.Roots(), ", "))
	}
	return nil
}

func (w *Wizard) checkOldConfig(values []string) error {
	if err := w.checkExisting(values); err != nil {
		return err
	}
	if _, err := w.env.Profiles(values[0]); err != nil {
		return i18n.Errorf("wizard.bad_old_config", values[0], err)
	}
	return nil
}
