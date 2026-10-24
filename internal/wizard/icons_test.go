package wizard

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// The icons a player keeps next to the config are found, down to three
// levels; the wizard's copies and other images are not.
func TestFindIcons(t *testing.T) {
	d := t.TempDir()
	for _, f := range []string{`Icons\dcs-dev.ico`, `Icons\DCS.ICO`, `a\b\deep.ico`, `a\b\c\too-deep.ico`, `.pm-new\copy.ico`, `Icons\photo.png`} {
		put(t, filepath.Join(d, f), "")
	}
	got := FindIcons(d)
	want := []string{filepath.Join(d, `Icons\DCS.ICO`), filepath.Join(d, `Icons\dcs-dev.ico`), filepath.Join(d, `a\b\deep.ico`)}
	if len(got) != len(want) {
		t.Fatalf("%v", got)
	}
	for _, w := range want {
		found := false
		for _, g := range got {
			found = found || g == w
		}
		if !found {
			t.Errorf("%s absent de %v", w, got)
		}
	}
	if FindIcons(filepath.Join(d, "missing")) != nil {
		t.Error("dossier absent")
	}
}

func TestIconForProfile(t *testing.T) {
	icons := []string{`C:\p\Icons\dcs.ico`, `C:\p\Icons\dcs-dev-old.ico`, `C:\p\Icons\dcs-dev.ico`, `C:\p\Icons\dcs-vr.ico`, `C:\p\Audio\sound-VR.ico`, `C:\p\Icons\devtools.ico`}
	for profile, want := range map[string]string{"DEV": icons[2], "vr": icons[3], "STD": noIcon} {
		if got := iconFor(profile, "dcs", icons); got != want {
			t.Errorf("%s : %q, voulu %q", profile, got, want)
		}
	}
	// David's folder: no dcs-vr.ico, and the sound profile's VR icon is not DCS's.
	if got := iconFor("VR", "dcs", []string{`C:\p\Icons\dcs-dev.ico`, `C:\p\Audio\sound-VR.ico`}); got != noIcon {
		t.Errorf("VR : %q", got)
	}
	// No icon carries the config's name: the profile's name alone decides.
	if got := iconFor("VR", "dcs", []string{`C:\p\vr.ico`}); got != `C:\p\vr.ico` {
		t.Errorf("sans nom de config : %q", got)
	}
}

// Each profile gets the icon chosen, as its icon key; none leaves it out.
func TestProfileIconsInTheConfig(t *testing.T) {
	icons := []string{saved + `\Profiles\Icons\dcs-dev.ico`, saved + `\Profiles\Icons\dcs.ico`}
	d := dcsUpToProfiles(t)
	d.w.env.Icons = func(string) []string { return icons }
	d.answer("current", "STD")
	d.answer("seed:DEV", "")
	d.act("learn:DEV", ActLearn, map[string]string{options: "o1"})
	d.answer("review:DEV", options)
	d.answer("seed:VR", "")
	d.act("learn:VR", ActLearn, map[string]string{})
	d.act("restore_end", ActRestoreOriginal, nil)

	q := d.w.Next().Question
	if q.Default[0] != icons[0] || q.Options[0].Label != `Icons\dcs-dev.ico` || len(q.Options) != 4 {
		t.Fatalf("DEV : %+v", q)
	}
	d.answer("icon:DEV", "")
	d.answer("icon:VR", noIcon)
	d.answer("icon:STD", otherIcon)
	if err := d.w.Answer([]string{saved + `\x.txt`}); err == nil {
		t.Error("un .txt accepté comme icône")
	}
	d.w.env.Exists = func(string) bool { return true }
	d.answer("icon:STD:other", `D:\icons\std.png`)
	d.answer("shortcuts", "${config_dir}")
	d.answer("confirm", "yes")

	p := d.w.Plan()
	for _, want := range []string{
		// options.lua is in every profile: common, so DEV has its icon and no hooks of its own.
		"  'DEV':\n    icon: '${env:USERPROFILE}\\Saved Games\\DCS\\Profiles\\Icons\\dcs-dev.ico'\n  'VR'",
		"  'VR': {}",
		"  'STD':\n    icon: 'D:\\icons\\std.png'\n",
	} {
		if !strings.Contains(p.YAML, want) {
			t.Errorf("YAML sans %q :\n%s", want, p.YAML)
		}
	}
	if got := []string{d.w.ProfileIcon("DEV"), d.w.ProfileIcon("VR")}; !reflect.DeepEqual(got, []string{icons[0], ""}) {
		t.Errorf("%v", got)
	}
	checkLoads(t, p)
}
