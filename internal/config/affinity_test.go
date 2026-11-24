package config

import (
	"math/bits"
	"strconv"
	"strings"
	"testing"

	"github.com/davidp57/process-monitor/internal/affinity"
)

func needCores(t *testing.T, n int) affinity.Mask {
	t.Helper()
	sys, err := affinity.System()
	if err != nil {
		t.Fatal(err)
	}
	if sys.Count() < n {
		t.Skipf("needs %d cores", n)
	}
	return sys
}

func TestAffinityLoads(t *testing.T) {
	needCores(t, 2)
	c, err := Load(write(t, `
command: a.exe
affinity:
  cores: 0
  others:
    cores: 1
    processes: [vrserver.exe, vrcompositor.exe]
`), "")
	if err != nil {
		t.Fatal(err)
	}
	a := c.Affinity
	if a == nil || a.Mask != 1 || a.Others == nil || a.Others.Mask != 2 ||
		strings.Join(a.Others.Processes, " ") != "vrserver.exe vrcompositor.exe" {
		t.Fatalf("affinity = %+v, others = %+v", a, a.Others)
	}
}

func TestAffinityPerProfile(t *testing.T) {
	needCores(t, 2)
	p := write(t, `
command: a.exe
affinity: { cores: 0-1 }
profiles:
  VR:
    affinity:
      cores: 0
      others: { cores: 1, processes: [vrserver.exe] }
  STD: {}
`)
	vr, err := Load(p, "VR")
	if err != nil {
		t.Fatal(err)
	}
	if vr.Affinity.Mask != 1 || vr.Affinity.Others == nil || vr.Affinity.Others.Mask != 2 {
		t.Errorf("VR: the profile's block must replace the top one: %+v", vr.Affinity)
	}
	std, err := Load(p, "STD")
	if err != nil {
		t.Fatal(err)
	}
	if std.Affinity.Mask != 3 || std.Affinity.Others != nil {
		t.Errorf("STD keeps the top block: %+v", std.Affinity)
	}
}

func TestAffinityOthersAlone(t *testing.T) {
	needCores(t, 2)
	c, err := Load(write(t, "command: a.exe\naffinity: { others: { cores: 1, processes: [x.exe] } }\n"), "")
	if err != nil {
		t.Fatal(err)
	}
	if c.Affinity.Mask != 0 || c.Affinity.Others.Mask != 2 {
		t.Errorf("affinity = %+v", c.Affinity)
	}
}

func TestAffinityGuards(t *testing.T) {
	sys := needCores(t, 1)
	missing := bits.Len64(uint64(sys)) // the first core past this machine's
	cases := map[string]struct{ yml, want string }{
		"empty block":   {"affinity: {}", "affinity :"},
		"empty cores":   {"affinity: { cores: '' }", "affinity :"},
		"bad list":      {"affinity: { cores: 3-1 }", "affinity.cores"},
		"hex mask":      {"affinity: { cores: '0x0FFF' }", "affinity.cores"},
		"others alone":  {"affinity: { others: { cores: 0 } }", "affinity.others"},
		"others no cpu": {"affinity: { others: { processes: [x.exe] } }", "affinity.others"},
		"others list":   {"affinity: { others: { cores: 'a', processes: [x.exe] } }", "affinity.others.cores"},
		"process path":  {"affinity: { others: { cores: 0, processes: ['C:\\x\\vr.exe'] } }", "nom de process seul"},
		"process ext":   {"affinity: { others: { cores: 0, processes: [vrserver] } }", "nom de process seul"},
		"the launcher":  {"affinity: { others: { cores: 0, processes: [A.exe] } }", "le programme"},
		"the followed":  {"track: game.exe\naffinity: { others: { cores: 0, processes: [game.exe] } }", "le programme"},
	}
	if missing < 64 {
		cases["missing core"] = struct{ yml, want string }{"affinity: { cores: " + strconv.Itoa(missing) + " }", "absent de cette machine"}
	}
	for name, tc := range cases {
		_, err := Load(write(t, "command: a.exe\n"+tc.yml+"\n"), "")
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s : err = %v, attendu %q", name, err, tc.want)
		}
	}
}

func TestAffinityInOverview(t *testing.T) {
	needCores(t, 2)
	c, err := Load(write(t, "command: a.exe\naffinity: { cores: 0-1, others: { cores: 1, processes: [vr.exe] } }\n"), "")
	if err != nil {
		t.Fatal(err)
	}
	o := c.Overview()
	if o.Affinity == nil || o.Affinity.Cores != "0-1" || o.Affinity.OthersCores != "1" || o.Affinity.Shared != "1" {
		t.Fatalf("overview = %+v", o.Affinity)
	}
	plan := c.Plan()
	for _, want := range []string{"cœurs   : 0-1", "vr.exe", "partagés"} {
		if !strings.Contains(plan, want) {
			t.Errorf("%q absent de :\n%s", want, plan)
		}
	}
}
