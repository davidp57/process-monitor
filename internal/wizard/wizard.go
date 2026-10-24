// Package wizard is pm new: questions in, a working config out. The engine
// here touches no file and no terminal: it says what to ask, or what to do
// (save the original files, learn a profile by playing, write the result),
// and the front runs it and hands the result back. The terminal is one
// front; a graphical interface can be another on the same engine.
package wizard

import (
	"errors"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/davidp57/process-monitor/internal/i18n"
)

// Kind is the shape of an answer.
type Kind int

const (
	Text   Kind = iota
	Path        // a file or folder path
	YesNo       // "yes" or "no"
	Choice      // one Option value
	List        // several free entries
	Multi       // several Option values
)

type Option struct{ Value, Label string }

// Question is one thing to ask. Answers are always a list of strings: one
// value, or several for List and Multi.
type Question struct {
	Kind     Kind
	Prompt   string
	Help     string // shown before the prompt
	Options  []Option
	Default  []string
	Optional bool // List and Multi: an empty answer is fine

	check func([]string) error
}

type ActionKind int

const (
	// ActRecover puts back the original files a previous, interrupted pm new left saved.
	ActRecover ActionKind = iota
	// ActSaveOriginal saves the watched folders as they are now; the result is a Snapshot.
	ActSaveOriginal
	// ActLearn puts the original files back, starts the game for Profile, waits
	// for it to end and saves what changed; the result is Changes as hashes
	// (path → hash, "" for a removed file).
	ActLearn
	// ActRestoreOriginal puts the original files back.
	ActRestoreOriginal
	// ActApply writes the Plan; Discard removes what the wizard kept, when nothing is written.
	ActApply
	ActDiscard
)

type Action struct {
	Kind    ActionKind
	Profile string
	Message string // for Learn: what the player has to do
}

// Step is the next thing the front has to do: ask a Question, run an
// Action, or nothing more when both are nil.
type Step struct {
	ID       string
	Question *Question
	Action   *Action
}

// Env is what the engine knows of the machine.
type Env struct {
	Home      string // %USERPROFILE%
	Documents string
	Lang      string
	Exists    func(path string) bool
	Now       func() time.Time
	// Profiles lists the profiles of a pm config, to start a profile from one.
	Profiles func(path string) ([]string, error)
	// Icons lists the .ico files under a folder, to offer one per profile.
	Icons func(dir string) []string
}

// Case values.
const (
	CaseDCS    = "dcs"
	CaseGame   = "game"
	CaseWatch  = "watch"
	CaseBackup = "backup"
)

// Wizard holds the answers and the results of the actions so far.
type Wizard struct {
	env     Env
	path    string // the config path given on the command line, a default
	answers map[string][]string
	history []string // answered question IDs, in order
	barrier int      // Back never goes before an action that ran
	done    map[string]bool

	original Snapshot
	learned  map[string]map[string]string
	// shown: what each answered question asked and what was answered, in
	// words, in the order of history: the "your answers" of a front.
	shown []Answered

	// leftover: a previous pm new on this config was interrupted and left
	// the original files saved. Seen once, when the config is chosen: the
	// copies this run saves must not look like a leftover.
	leftover bool
}

func New(env Env, path string) *Wizard {
	return &Wizard{env: env, path: path, answers: map[string][]string{}, done: map[string]bool{},
		learned: map[string]map[string]string{}}
}

// Next is the first step not done yet.
func (w *Wizard) Next() Step {
	for _, s := range w.steps() {
		if !w.done[s.ID] {
			return s
		}
	}
	return Step{}
}

// Answer records the answer to the current question, after checking it.
func (w *Wizard) Answer(values []string) error {
	s := w.Next()
	if s.Question == nil {
		panic("wizard: Answer with no question pending") // a front bug
	}
	var kept []string
	for _, v := range values {
		if v = strings.TrimSpace(v); v != "" {
			kept = append(kept, v)
		}
	}
	q := s.Question
	if len(kept) == 0 && len(q.Default) > 0 && q.Kind != List && q.Kind != Multi {
		kept = q.Default
	}
	if len(kept) == 0 && !q.Optional {
		return errors.New(i18n.T("wizard.answer_required"))
	}
	if q.Kind != List && q.Kind != Multi && len(kept) > 1 {
		kept = kept[:1]
	}
	if q.Kind == Choice || q.Kind == Multi || q.Kind == YesNo {
		for _, v := range kept {
			if !hasOption(q.Options, v) {
				return i18n.Errorf("wizard.bad_choice", v)
			}
		}
	}
	if q.check != nil {
		if err := q.check(kept); err != nil {
			return err
		}
	}
	w.answers[s.ID] = kept
	w.done[s.ID] = true
	w.shown = append(w.shown, Answered{ID: s.ID, Prompt: q.Prompt, Answer: shownAnswer(q, kept)})
	if s.ID == "config" {
		w.leftover = w.env.Exists(filepath.Join(w.WizardDir(), manifestName))
	}
	w.history = append(w.history, s.ID)
	return nil
}

// Answered is one question answered, as a front recalls it.
type Answered struct {
	ID     string `json:"id"`
	Prompt string `json:"prompt"`
	Answer string `json:"answer"`
}

// Answers are the questions answered so far, in order.
func (w *Wizard) Answers() []Answered { return append([]Answered{}, w.shown...) }

// shownAnswer is an answer in words: the labels of the options chosen.
func shownAnswer(q *Question, values []string) string {
	var out []string
	for _, v := range values {
		label := v
		for _, o := range q.Options {
			if o.Value == v && o.Label != "" {
				label = o.Label
			}
		}
		out = append(out, label)
	}
	return strings.Join(out, ", ")
}

func hasOption(opts []Option, v string) bool {
	for _, o := range opts {
		if o.Value == v {
			return true
		}
	}
	return false
}

// ActionsRan reports whether an action already touched the files: Back
// then stops there.
func (w *Wizard) ActionsRan() bool { return w.barrier > 0 }

// Back forgets the last answer, unless an action ran since: the files it
// touched are not undone by going back.
func (w *Wizard) Back() bool {
	if len(w.history) <= w.barrier {
		return false
	}
	id := w.history[len(w.history)-1]
	w.history = w.history[:len(w.history)-1]
	w.shown = w.shown[:len(w.shown)-1]
	delete(w.answers, id)
	delete(w.done, id)
	return true
}

// Done records that the current action ran, with its result.
func (w *Wizard) Done(result any) {
	s := w.Next()
	if s.Action == nil {
		return
	}
	switch s.Action.Kind {
	case ActSaveOriginal:
		w.original, _ = result.(Snapshot)
	case ActLearn:
		w.learned[s.Action.Profile], _ = result.(map[string]string)
	}
	w.done[s.ID] = true
	w.barrier = len(w.history)
}

// Applied reports whether the wizard ended by writing the config, rather
// than by the user saying no at the summary.
func (w *Wizard) Applied() bool { return w.done["apply"] }

func (w *Wizard) one(id string) string {
	if v := w.answers[id]; len(v) > 0 {
		return v[0]
	}
	return ""
}

func (w *Wizard) has(id string) bool { return w.done[id] }

func yn() []Option {
	return []Option{{"yes", i18n.T("wizard.yes")}, {"no", i18n.T("wizard.no")}}
}

// steps lists the steps known so far: each one that depends on an answer
// not given yet stops the list there.
func (w *Wizard) steps() []Step {
	var out []Step
	ask := func(id string, q Question) bool {
		out = append(out, Step{ID: id, Question: &q})
		return w.has(id)
	}
	act := func(id string, a Action) bool {
		out = append(out, Step{ID: id, Action: &a})
		return w.has(id)
	}

	if !ask("case", Question{Kind: Choice, Prompt: i18n.T("wizard.q.case"), Options: []Option{
		{CaseDCS, i18n.T("wizard.case.dcs")},
		{CaseGame, i18n.T("wizard.case.game")},
		{CaseWatch, i18n.T("wizard.case.watch")},
		{CaseBackup, i18n.T("wizard.case.backup")},
	}}) {
		return out
	}
	kase := w.one("case")
	switch kase {
	case CaseDCS:
		if !ask("launcher", w.existing(i18n.T("wizard.q.dcs_launcher"), i18n.T("wizard.h.dcs_launcher"), dcsLauncher)) {
			return out
		}
		if !ask("saved", w.existing(i18n.T("wizard.q.dcs_saved"), "", w.dcsSaved())) {
			return out
		}
		if !ask("srs", Question{Kind: YesNo, Prompt: i18n.T("wizard.q.srs"), Options: yn(), Default: []string{"yes"}}) {
			return out
		}
	case CaseGame:
		if !ask("launcher", w.existing(i18n.T("wizard.q.launcher"), "", "")) {
			return out
		}
		if !ask("track", Question{Kind: Text, Prompt: i18n.T("wizard.q.track"), Help: i18n.T("wizard.h.track"),
			Optional: true, check: checkProcessName}) {
			return out
		}
		if !ask("roots", Question{Kind: List, Prompt: i18n.T("wizard.q.roots"), Help: i18n.T("wizard.h.roots"),
			check: w.checkExisting}) {
			return out
		}
	case CaseWatch, CaseBackup:
		if !ask("program", w.existing(i18n.T("wizard.q.program"), "", "")) {
			return out
		}
		if kase == CaseWatch {
			if !ask("kill_running", Question{Kind: YesNo, Prompt: i18n.T("wizard.q.kill_running"), Options: yn(), Default: []string{"yes"}}) {
				return out
			}
			if !ask("clean", Question{Kind: List, Prompt: i18n.T("wizard.q.clean"), Help: i18n.T("wizard.h.clean"),
				Optional: true, check: checkAbsolute}) {
				return out
			}
			if !ask("delay", Question{Kind: Text, Prompt: i18n.T("wizard.q.delay"), Default: []string{"2s"}, check: checkDuration}) {
				return out
			}
		} else {
			if !ask("saves", w.existing(i18n.T("wizard.q.saves"), "", "")) {
				return out
			}
			if !ask("archives", Question{Kind: Path, Prompt: i18n.T("wizard.q.archives"), check: checkAbsolute}) {
				return out
			}
			if !ask("keep", Question{Kind: Text, Prompt: i18n.T("wizard.q.keep"), Default: []string{"10"}, check: checkCount}) {
				return out
			}
		}
	}

	if !ask("name", Question{Kind: Text, Prompt: i18n.T("wizard.q.name"), Help: i18n.T("wizard.h.name"),
		Default: []string{w.defaultName()}, check: checkFileName}) {
		return out
	}
	if !ask("config", Question{Kind: Path, Prompt: i18n.T("wizard.q.config"), Default: []string{w.defaultConfig()},
		check: w.checkNewConfig}) {
		return out
	}

	if w.profiled() {
		if w.leftover {
			if !act("recover", Action{Kind: ActRecover}) {
				return out
			}
		}
		if !ask("profiles", Question{Kind: List, Prompt: i18n.T("wizard.q.profiles"), Help: i18n.T("wizard.h.profiles"),
			check: w.checkProfiles}) {
			return out
		}
		if !act("save_original", Action{Kind: ActSaveOriginal}) {
			return out
		}
		// One profile at most is the setup as it is now; the others are played.
		current := []Option{}
		for _, p := range w.answers["profiles"] {
			current = append(current, Option{p, p})
		}
		current = append(current, Option{noProfile, i18n.T("wizard.current_none")})
		if !ask("current", Question{Kind: Choice, Prompt: i18n.T("wizard.q.current"), Help: i18n.T("wizard.h.current"),
			Options: current, Default: []string{noProfile}}) {
			return out
		}
		for _, p := range w.answers["profiles"] {
			if p != w.one("current") {
				if !w.seedSteps(p, ask) {
					return out
				}
				if !act("learn:"+p, Action{Kind: ActLearn, Profile: p, Message: i18n.T("wizard.learn", p)}) {
					return out
				}
				// Right after the game: what changed is still fresh in mind.
				if files := sortedKeys(w.learned[p]); len(files) > 0 {
					opts := make([]Option, len(files))
					for i, f := range files {
						opts[i] = Option{f, w.label(f)}
					}
					if !ask("review:"+p, Question{Kind: Multi, Prompt: i18n.T("wizard.q.review", p), Help: i18n.T("wizard.h.review"),
						Options: opts, Default: files, Optional: true}) {
						return out
					}
				}
			}
		}
		if !act("restore_end", Action{Kind: ActRestoreOriginal}) {
			return out
		}
		for _, p := range w.answers["profiles"] {
			if !w.iconSteps(p, ask) {
				return out
			}
		}
	}

	if !ask("shortcuts", ShortcutFolders()) {
		return out
	}
	if hasOption(optionsOf(w.answers["shortcuts"]), "other") {
		if !ask("shortcuts_folder", Question{Kind: Path, Prompt: i18n.T("wizard.q.shortcuts_folder"), check: checkAbsolute}) {
			return out
		}
	}
	if !ask("confirm", Question{Kind: YesNo, Prompt: i18n.T("wizard.q.confirm"), Help: w.Summary(),
		Options: yn(), Default: []string{"yes"}}) {
		return out
	}
	if w.one("confirm") == "yes" {
		act("apply", Action{Kind: ActApply})
	} else if w.profiled() {
		act("discard", Action{Kind: ActDiscard})
	}
	return out
}

// noProfile answers "which profile is the current setup": none of them.
// A profile name cannot start with -, so it never collides; and answers
// are trimmed, so it must not hold spaces either.
const noProfile = "-"

// ShortcutFolders asks where the shortcuts go, one place or several: folder
// variables pm resolves, and "other" for a path typed next.
func ShortcutFolders() Question {
	return Question{Kind: Multi, Prompt: i18n.T("wizard.q.shortcuts"), Options: []Option{
		{"${config_dir}", i18n.T("wizard.sc.config_dir")},
		{"${desktop}", i18n.T("wizard.sc.desktop")},
		{"${start_menu}", i18n.T("wizard.sc.start_menu")},
		{"other", i18n.T("wizard.sc.other")},
	}, Default: []string{"${config_dir}"}}
}

func optionsOf(values []string) []Option {
	out := make([]Option, len(values))
	for i, v := range values {
		out[i] = Option{Value: v}
	}
	return out
}

func (w *Wizard) profiled() bool {
	k := w.one("case")
	return k == CaseDCS || k == CaseGame
}

// WizardDir is where the wizard keeps the original files while it runs:
// next to the config, so a later pm new on the same config finds them.
func (w *Wizard) WizardDir() string {
	return filepath.Join(filepath.Dir(w.one("config")), ".pm-new")
}

// Roots are the folders watched while learning profiles.
func (w *Wizard) Roots() []string {
	if w.one("case") == CaseDCS {
		s := w.one("saved")
		return []string{filepath.Join(s, "Config"), filepath.Join(s, "Scripts")}
	}
	return w.answers["roots"]
}

// Ignored are files under the roots the game rewrites on its own and that
// must never be carried nor put back: DCS's login data changes at every
// launch, and an old copy would replace a fresh one.
func (w *Wizard) Ignored() []string {
	if w.one("case") == CaseDCS {
		return []string{filepath.Join(w.one("saved"), "Config", "authdata.bin")}
	}
	return nil
}

// Launch is what Learn starts: the launcher and the process to follow.
func (w *Wizard) Launch() (command, track string) {
	if w.one("case") == CaseDCS {
		return w.one("launcher"), "dcs.exe"
	}
	return w.one("launcher"), w.one("track")
}

const dcsLauncher = `C:\Program Files\Eagle Dynamics\DCS World\bin\dcs_updater.exe`

// dcsSaved is the first DCS folder of Saved Games found.
func (w *Wizard) dcsSaved() string {
	for _, d := range []string{"DCS", "DCS.openbeta"} {
		if p := filepath.Join(w.env.Home, "Saved Games", d); w.env.Exists(p) {
			return p
		}
	}
	return filepath.Join(w.env.Home, "Saved Games", "DCS")
}

func (w *Wizard) existing(prompt, help, def string) Question {
	q := Question{Kind: Path, Prompt: prompt, Help: help, check: w.checkExisting}
	if def != "" {
		q.Default = []string{def}
	}
	return q
}

func (w *Wizard) defaultName() string {
	switch w.one("case") {
	case CaseDCS:
		return "dcs"
	case CaseGame:
		return baseName(w.one("launcher"))
	}
	return baseName(w.one("program"))
}

func baseName(p string) string {
	b := filepath.Base(p)
	return strings.ToLower(strings.TrimSuffix(b, filepath.Ext(b)))
}

func (w *Wizard) defaultConfig() string {
	if w.path != "" {
		if abs, err := filepath.Abs(w.path); err == nil {
			return abs
		}
	}
	name := w.one("name")
	if w.one("case") == CaseDCS {
		return filepath.Join(w.one("saved"), "Profiles", "pm-"+name+".yaml")
	}
	return filepath.Join(w.env.Documents, "pm", name, name+".yaml")
}

func (w *Wizard) checkExisting(values []string) error {
	for _, v := range values {
		if !filepath.IsAbs(v) {
			return i18n.Errorf("wizard.not_absolute", v)
		}
		if !w.env.Exists(v) {
			return i18n.Errorf("wizard.not_found", v)
		}
	}
	return nil
}

func (w *Wizard) checkNewConfig(values []string) error {
	p := values[0]
	if !filepath.IsAbs(p) {
		return i18n.Errorf("wizard.not_absolute", p)
	}
	if ext := strings.ToLower(filepath.Ext(p)); ext != ".yaml" && ext != ".yml" {
		return i18n.Errorf("wizard.not_yaml", p)
	}
	if w.env.Exists(p) {
		return i18n.Errorf("wizard.config_exists", p)
	}
	// The wizard keeps its copies next to the config: inside a watched
	// folder, they would count as files the game changed.
	if w.profiled() {
		for _, r := range w.Roots() {
			if rel, err := filepath.Rel(r, filepath.Dir(p)); err == nil && !strings.HasPrefix(rel, "..") {
				return i18n.Errorf("wizard.config_in_root", r)
			}
		}
	}
	return nil
}

func checkAbsolute(values []string) error {
	for _, v := range values {
		if !filepath.IsAbs(v) {
			return i18n.Errorf("wizard.not_absolute", v)
		}
	}
	return nil
}

const badFileChars = `\/:*?"<>|`

func checkFileName(values []string) error {
	if strings.ContainsAny(values[0], badFileChars) {
		return i18n.Errorf("wizard.bad_name", values[0])
	}
	return nil
}

// profileName is what a profile may be called: it names a folder, a YAML
// key and a --profile argument, so letters, digits, - _ and ., not first -.
var profileName = regexp.MustCompile(`^[\p{L}\p{N}_][\p{L}\p{N}_.-]*$`)

// checkProfiles also refuses a profile whose folder is already there: the
// wizard would write over files it did not make.
func (w *Wizard) checkProfiles(values []string) error {
	seen := map[string]bool{}
	for _, v := range values {
		if !profileName.MatchString(v) {
			return i18n.Errorf("wizard.bad_profile", v)
		}
		if seen[strings.ToLower(v)] {
			return i18n.Errorf("wizard.dup_profile", v)
		}
		seen[strings.ToLower(v)] = true
		if dir := w.profileDir(v); w.env.Exists(dir) {
			return i18n.Errorf("wizard.profile_exists", dir)
		}
	}
	return nil
}

// profileDir is a profile's folder: next to the config, named after it.
func (w *Wizard) profileDir(profile string) string {
	return filepath.Join(filepath.Dir(w.one("config")), profile)
}

func checkProcessName(values []string) error {
	for _, v := range values {
		if strings.ContainsAny(v, `\/`) || filepath.Ext(v) == "" {
			return i18n.Errorf("wizard.bad_track", v)
		}
	}
	return nil
}

func checkDuration(values []string) error {
	if d, err := time.ParseDuration(values[0]); err != nil || d < 0 {
		return i18n.Errorf("wizard.bad_delay", values[0])
	}
	return nil
}

func checkCount(values []string) error {
	if n, err := strconv.Atoi(values[0]); err != nil || n < 1 {
		return i18n.Errorf("wizard.bad_count", values[0])
	}
	return nil
}
