package wizard

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/davidp57/process-monitor/internal/config"
)

// fakeEnv knows the files of a pretend machine.
func fakeEnv(files ...string) Env {
	set := map[string]bool{}
	for _, f := range files {
		set[strings.ToLower(f)] = true
	}
	return Env{Home: `C:\Users\Jo`, Documents: `C:\Users\Jo\Documents`, Lang: "fr",
		Exists: func(p string) bool { return set[strings.ToLower(p)] },
		Now:    func() time.Time { return time.Date(2026, 10, 24, 0, 0, 0, 0, time.UTC) },
		Profiles: func(p string) ([]string, error) {
			if !set[strings.ToLower(p)] {
				return nil, os.ErrNotExist
			}
			return []string{"DEV", "VR"}, nil
		}}
}

// drive answers questions and runs actions with results from the test.
type drive struct {
	t *testing.T
	w *Wizard
}

func (d drive) answer(id string, v ...string) {
	d.t.Helper()
	s := d.w.Next()
	if s.ID != id || s.Question == nil {
		d.t.Fatalf("étape %q, voulu la question %q", s.ID, id)
	}
	if err := d.w.Answer(v); err != nil {
		d.t.Fatalf("%s : %v", id, err)
	}
}

func (d drive) act(id string, kind ActionKind, result any) {
	d.t.Helper()
	s := d.w.Next()
	if s.ID != id || s.Action == nil || s.Action.Kind != kind {
		d.t.Fatalf("étape %+v, voulu l'action %q", s, id)
	}
	d.w.Done(result)
}

const (
	saved   = `C:\Users\Jo\Saved Games\DCS`
	options = saved + `\Config\options.lua`
	joy     = saved + `\Config\Input\F-16C_50\joystick\Throttle {AB-12}.diff.lua`
	export  = saved + `\Scripts\Export.lua`
)

// dcsUpToProfiles answers the DCS questions and saves the original state.
func dcsUpToProfiles(t *testing.T) drive {
	d := drive{t, New(fakeEnv(dcsLauncher, saved), "")}
	d.answer("case", CaseDCS)
	d.answer("launcher", "")
	d.answer("saved", "")
	d.answer("srs", "")
	d.answer("name", "")
	d.answer("config", "")
	if got := d.w.one("config"); got != saved+`\Profiles\pm-dcs.yaml` {
		t.Fatalf("config = %q", got)
	}
	d.answer("profiles", "DEV", "VR", "STD")
	d.act("save_original", ActSaveOriginal, Snapshot{options: "o0", export: "e0"})
	return d
}

// dcsPlayed plays the three profiles: DEV changes options and adds a
// joystick file, VR changes options only, STD is the current setup.
// keepDEV is what the player keeps in DEV's report; none: everything.
func dcsPlayed(t *testing.T, keepDEV ...string) drive {
	d := dcsUpToProfiles(t)
	d.answer("current", "STD")
	d.answer("seed:DEV", "")
	d.act("learn:DEV", ActLearn, map[string]string{options: "o1", joy: "j1"})
	if keepDEV == nil {
		keepDEV = d.w.Next().Question.Default
	}
	d.answer("review:DEV", keepDEV...)
	d.answer("seed:VR", "")
	d.act("learn:VR", ActLearn, map[string]string{options: "o2"})
	d.answer("review:VR", d.w.Next().Question.Default...)
	d.act("restore_end", ActRestoreOriginal, nil)
	for _, p := range []string{"DEV", "VR", "STD"} {
		d.answer("icon:"+p, "")
	}
	return d
}

// The report comes right after each game, with that profile's files only.
func TestReportAfterEachProfile(t *testing.T) {
	d := dcsUpToProfiles(t)
	s := d.w.Next()
	if s.ID != "current" || len(s.Question.Options) != 4 || s.Question.Default[0] != noProfile {
		t.Fatalf("question : %+v", s)
	}
	// The front sends the value of the option chosen, as it is.
	d.answer("current", s.Question.Options[3].Value)
	d.answer("seed:DEV", "")
	d.act("learn:DEV", ActLearn, map[string]string{options: "o1", joy: "j1"})
	s = d.w.Next()
	if s.ID != "review:DEV" || len(s.Question.Options) != 2 || !strings.Contains(s.Question.Prompt, "DEV") {
		t.Fatalf("rapport : %+v", s)
	}
	if s.Question.Options[0].Label != "commandes de Throttle (F-16C_50)" || s.Question.Options[1].Label != "options du jeu" {
		t.Errorf("libellés : %+v", s.Question.Options)
	}
	d.answer("review:DEV", s.Question.Default...)
	d.answer("seed:VR", "")
	d.act("learn:VR", ActLearn, map[string]string{options: "o2"})
	if s := d.w.Next(); s.ID != "review:VR" || len(s.Question.Options) != 1 {
		t.Fatalf("rapport VR : %+v", s)
	}
}

func TestDCSProfilesLearnedByPlaying(t *testing.T) {
	d := dcsPlayed(t)
	d.answer("shortcuts", "${start_menu}")
	if h := d.w.Next().Question.Help; !strings.Contains(h, "sans : commandes de Throttle") || !strings.Contains(h, "Profil STD (réglages actuels)") {
		t.Errorf("résumé final :\n%s", h)
	}
	d.answer("confirm", "yes")
	d.act("apply", ActApply, nil)
	if !d.w.Applied() || d.w.Next().ID != "" {
		t.Fatalf("fin : %+v", d.w.Next())
	}

	p := d.w.Plan()
	t.Log("\n" + p.YAML)
	for _, want := range []string{
		`- '${prof}\Config\options.lua <-> ${env:USERPROFILE}\Saved Games\DCS\Config\options.lua'`,
		`track: 'dcs.exe'`, `kill_on_exit: [SR-ClientRadio.exe]`, `folder: '${start_menu}'`,
		`.\pm.exe run "` + saved + `\Profiles\pm-dcs.yaml" --profile DEV`,
	} {
		if !strings.Contains(p.YAML, want) {
			t.Errorf("YAML sans %q :\n%s", want, p.YAML)
		}
	}
	// The joystick file exists in DEV only: synced there, deleted before VR and STD.
	if strings.Count(p.YAML, "Throttle {AB-12}.diff.lua <->") != 1 || strings.Count(p.YAML, "- delete:") != 2 {
		t.Errorf("fichier propre à DEV :\n%s", p.YAML)
	}
	// Each profile's version of each file: DEV and VR their own, STD the original.
	prof := saved + `\Profiles`
	want := map[string]string{
		prof + `\DEV\Config\options.lua`:                                       storePath(d.w.ProfileStore("DEV"), options),
		prof + `\VR\Config\options.lua`:                                        storePath(d.w.ProfileStore("VR"), options),
		prof + `\STD\Config\options.lua`:                                       storePath(d.w.originalStore(), options),
		prof + `\DEV\Config\Input\F-16C_50\joystick\Throttle {AB-12}.diff.lua`: storePath(d.w.ProfileStore("DEV"), joy),
	}
	if len(p.Files) != len(want) {
		t.Errorf("fichiers : %+v", p.Files)
	}
	for _, f := range p.Files {
		if want[f.To] != f.From {
			t.Errorf("%s ← %s, voulu %s", f.To, f.From, want[f.To])
		}
	}
	// export.lua was touched by no profile: not carried.
	if strings.Contains(p.YAML, "Export.lua") {
		t.Errorf("Export.lua emporté :\n%s", p.YAML)
	}
	checkLoads(t, p)
}

// Unticking in a profile's report gives that profile the original back,
// and nothing else: VR's options stay VR's.
func TestUntickKeepsTheOriginalInThatProfile(t *testing.T) {
	d := dcsPlayed(t, joy) // DEV keeps its joystick file, not its options
	d.answer("shortcuts", "${config_dir}")
	d.answer("confirm", "yes")
	p := d.w.Plan()
	prof := saved + `\Profiles`
	for _, f := range p.Files {
		switch f.To {
		case prof + `\DEV\Config\options.lua`:
			if f.From != storePath(d.w.originalStore(), options) {
				t.Errorf("DEV options ← %s, voulu l'original", f.From)
			}
		case prof + `\VR\Config\options.lua`:
			if f.From != storePath(d.w.ProfileStore("VR"), options) {
				t.Errorf("VR options ← %s, voulu celles de VR", f.From)
			}
		}
	}
	checkLoads(t, p)

	// Unticked everywhere it changed: not carried at all.
	d = dcsUpToProfiles(t)
	d.answer("current", "VR")
	d.answer("seed:DEV", "")
	d.act("learn:DEV", ActLearn, map[string]string{options: "o1"})
	d.answer("review:DEV")
	d.answer("seed:STD", "")
	d.act("learn:STD", ActLearn, map[string]string{}) // nothing changed: no report
	d.act("restore_end", ActRestoreOriginal, nil)
	for _, p := range []string{"DEV", "VR", "STD"} {
		d.answer("icon:"+p, "")
	}
	d.answer("shortcuts", "${config_dir}")
	d.answer("confirm", "yes")
	if p := d.w.Plan(); strings.Contains(p.YAML, "options.lua") || len(p.Files) != 0 {
		t.Errorf("décoché partout, encore emporté :\n%s", p.YAML)
	}
}

func TestBackStopsAtAnActionThatRan(t *testing.T) {
	d := dcsUpToProfiles(t)
	d.answer("current", "STD")
	d.answer("seed:DEV", "")
	d.act("learn:DEV", ActLearn, map[string]string{options: "o1"})
	if d.w.Back() {
		t.Error("retour au-delà d'une action : les fichiers ne reviennent pas avec")
	}
	d.answer("review:DEV", options)
	if !d.w.Back() || d.w.Next().ID != "review:DEV" {
		t.Errorf("retour : %+v", d.w.Next())
	}
}

func TestSayingNoDiscards(t *testing.T) {
	d := dcsPlayed(t)
	d.answer("shortcuts", "${config_dir}")
	d.answer("confirm", "no")
	d.act("discard", ActDiscard, nil)
	if d.w.Applied() || d.w.Next().ID != "" {
		t.Errorf("fin : %+v", d.w.Next())
	}
}

func TestDCSIgnoresLoginData(t *testing.T) {
	d := dcsUpToProfiles(t)
	if got := d.w.Ignored(); len(got) != 1 || got[0] != saved+`\Config\authdata.bin` {
		t.Errorf("ignorés : %v", got)
	}
}

func TestChecks(t *testing.T) {
	cfg := `C:\Users\Jo\Documents\pm\sim\sim.yaml`
	d := drive{t, New(fakeEnv(`C:\Sim\sim.exe`, cfg), "")}
	d.answer("case", CaseWatch)
	for _, bad := range [][]string{{`C:\Nope\x.exe`}, {`sim.exe`}} {
		if err := d.w.Answer(bad); err == nil {
			t.Errorf("%v accepté", bad)
		}
	}
	d.answer("program", `C:\Sim\sim.exe`)
	d.answer("kill_running", "")
	d.answer("clean", `C:\Sim\Temp`)
	if err := d.w.Answer([]string{"soon"}); err == nil {
		t.Error("délai invalide accepté")
	}
	d.answer("delay", "")
	d.answer("name", "")
	if err := d.w.Answer(nil); err == nil || !strings.Contains(err.Error(), "existe déjà") {
		t.Errorf("config existante : %v", err)
	}
}

func TestProfileNames(t *testing.T) {
	d := drive{t, New(fakeEnv(dcsLauncher, saved, saved+`\Profiles\DEV`), "")}
	d.answer("case", CaseDCS)
	d.answer("launcher", "")
	d.answer("saved", "")
	d.answer("srs", "")
	d.answer("name", "")
	d.answer("config", "")
	for _, bad := range [][]string{{"#1"}, {"-x"}, {"V R"}, {"a:b"}, {"VR", "vr"}, {"DEV"}} {
		if err := d.w.Answer(bad); err == nil {
			t.Errorf("%v accepté", bad)
		}
	}
	d.answer("profiles", "VR", "Écran_2", "v1.2")
}

func TestConfigNotInAWatchedFolder(t *testing.T) {
	d := drive{t, New(fakeEnv(`C:\Game\game.exe`, `C:\Game\Settings`), "")}
	d.answer("case", CaseGame)
	d.answer("launcher", `C:\Game\game.exe`)
	d.answer("track", "")
	d.answer("roots", `C:\Game\Settings`)
	d.answer("name", "")
	if err := d.w.Answer([]string{`C:\Game\Settings\pm\game.yaml`}); err == nil {
		t.Error("config dans un dossier surveillé acceptée")
	}
	d.answer("config", `C:\Game\pm\game.yaml`)
}

func TestWatchCase(t *testing.T) {
	d := drive{t, New(fakeEnv(`C:\Users\Jo\AppData\Local\Sim\sim.exe`), "")}
	d.answer("case", CaseWatch)
	d.answer("program", `C:\Users\Jo\AppData\Local\Sim\sim.exe`)
	d.answer("kill_running", "yes")
	d.answer("clean", `C:\Users\Jo\AppData\Local\Sim\cache`)
	d.answer("delay", "")
	d.answer("name", "")
	d.answer("config", "")
	// Several places at once, one of them typed.
	d.answer("shortcuts", "${desktop}", "${start_menu}", "other")
	d.answer("shortcuts_folder", `C:\Users\Jo\Links`)
	if !strings.Contains(d.w.Next().Question.Help, `sim.exe`) {
		t.Errorf("résumé : %q", d.w.Next().Question.Help)
	}
	d.answer("confirm", "yes")
	d.act("apply", ActApply, nil)
	p := d.w.Plan()
	for _, want := range []string{"if_running: kill", "watch: { delay: 2s }", "mode: watch",
		`- delete: '${env:USERPROFILE}\AppData\Local\Sim\cache'`,
		`folder: ['${desktop}', '${start_menu}', '${env:USERPROFILE}\Links']`} {
		if !strings.Contains(p.YAML, want) {
			t.Errorf("YAML sans %q :\n%s", want, p.YAML)
		}
	}
	if p.WizardDir != "" || len(p.Profiles) != 0 {
		t.Errorf("plan : %+v", p)
	}
	checkLoads(t, p)
}

func TestBackupCase(t *testing.T) {
	d := drive{t, New(fakeEnv(`C:\Game\game.exe`, `C:\Saves`), `C:\cfg\game.yaml`)}
	d.answer("case", CaseBackup)
	d.answer("program", `C:\Game\game.exe`)
	d.answer("saves", `C:\Saves`)
	d.answer("archives", `C:\Archives`)
	d.answer("keep", "4")
	d.answer("name", "")
	d.answer("config", "")
	if d.w.one("config") != `C:\cfg\game.yaml` {
		t.Errorf("config = %q", d.w.one("config"))
	}
	d.answer("shortcuts", "${config_dir}")
	d.answer("confirm", "yes")
	p := d.w.Plan()
	if !strings.Contains(p.YAML, "Select-Object -Skip 4") || !strings.Contains(p.YAML, "archives: 'C:\\Archives'") {
		t.Errorf("YAML :\n%s", p.YAML)
	}
	checkLoads(t, p)
}

// checkLoads writes the plan's config into a temp folder and loads it with
// every profile: a wizard that writes a config pm refuses is a bug.
func checkLoads(t *testing.T, p Plan) {
	t.Helper()
	f := filepath.Join(t.TempDir(), "check.yaml")
	if err := os.WriteFile(f, []byte(p.YAML), 0o644); err != nil {
		t.Fatal(err)
	}
	profiles := p.Profiles
	if len(profiles) == 0 {
		profiles = []string{""}
	}
	for _, prof := range profiles {
		if _, err := config.Load(f, prof); err != nil {
			t.Errorf("profil %q : %v\n%s", prof, err, p.YAML)
		}
	}
}

// Loose files: each one's place is chosen among the files of the same name
// already in the game, the likeliest first, or typed by hand.
func TestSeedFilesChoosesWhereEachGoes(t *testing.T) {
	f16 := saved + `\Config\Input\F-16C_50\joystick\Throttle.diff.lua`
	a10 := saved + `\Config\Input\A-10C\joystick\Throttle.diff.lua`
	preset := saved + `\Config\Presets\Backup\Options.lua`
	d := dcsUpToProfiles(t)
	d.w.original = Snapshot{options: "o0", preset: "p", f16: "f", a10: "a"}
	d.answer("current", "STD")
	d.answer("seed:DEV", seedFiles)
	old := []string{`D:\old\F-16C_50\joystick\Throttle.diff.lua`, `D:\old\STD\options.lua`, `D:\old\x\Throttle.diff.lua`, `D:\old\new.lua`}
	d.w.env.Exists = func(string) bool { return true }
	d.answer("seed_files:DEV", old...)

	// The F-16's file, where the folders match; the A-10's offered too, and a path of one's own.
	q := d.w.Next().Question
	if q.Kind != Choice || q.Default[0] != f16 || len(q.Options) != 3 || q.Options[1].Value != a10 || q.Options[2].Value != otherPath {
		t.Fatalf("Throttle : %+v", q)
	}
	d.answer(seedTo("DEV", 0), "")
	// No folder in common: the same case and the shallowest path win over the presets.
	if q := d.w.Next().Question; q.Default[0] != options || q.Options[1].Value != preset {
		t.Errorf("options.lua : %+v", q)
	}
	d.answer(seedTo("DEV", 1), "")
	// Typed by hand: inside the watched folders only.
	d.answer(seedTo("DEV", 2), otherPath)
	for _, bad := range []string{`D:\elsewhere\Throttle.diff.lua`, saved + `\Config\authdata.bin`} {
		if err := d.w.Answer([]string{bad}); err == nil {
			t.Errorf("%s accepté", bad)
		}
	}
	mine := saved + `\Config\Input\Ka-50\joystick\Throttle.diff.lua`
	d.answer(seedTo("DEV", 2)+":other", mine)
	// Nothing of that name in the game: a path, the first watched folder suggested.
	if q := d.w.Next().Question; q.Kind != Path || q.Default[0] != saved+`\Config\new.lua` {
		t.Errorf("new.lua : %+v", q)
	}
	d.answer(seedTo("DEV", 3), "")

	want := []FileCopy{{old[0], f16}, {old[1], options}, {old[2], mine}, {old[3], saved + `\Config\new.lua`}}
	if got := d.w.Seed("DEV").Copies; !reflect.DeepEqual(got, want) {
		t.Errorf("copies : %+v", got)
	}
	if s := d.w.Next(); s.ID != "learn:DEV" {
		t.Errorf("étape suivante : %q", s.ID)
	}
}

func TestSeedFromAnOldConfig(t *testing.T) {
	d := dcsUpToProfiles(t)
	d.answer("current", "STD")
	d.answer("seed:DEV", seedConfig)
	if err := d.w.Answer([]string{`C:\nope\pm-dcs.yaml`}); err == nil {
		t.Error("config absente acceptée")
	}
	d.w.env.Exists = func(string) bool { return true }
	d.w.env.Profiles = func(string) ([]string, error) { return []string{"DEV", "VR"}, nil }
	d.answer("seed_config:DEV", `C:\old\pm-dcs.yaml`)
	if q := d.w.Next().Question; q.Default[0] != "DEV" || len(q.Options) != 2 {
		t.Errorf("profil proposé : %+v", q)
	}
	d.answer("seed_profile:DEV", "")
	if s := d.w.Seed("DEV"); s.Config != `C:\old\pm-dcs.yaml` || s.Profile != "DEV" {
		t.Errorf("seed : %+v", s)
	}
}
