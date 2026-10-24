package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOverview(t *testing.T) {
	p := write(t, `
command: C:\game\updater.exe
track: game.exe
hooks:
  sync:
    - ${config_dir}\prof\options.lua <-> ${config_dir}\live\options.lua
    - ${config_dir}\prof\Input <-> ${config_dir}\live\Input
    - ${config_dir}\prof\Kneeboard <-> ${config_dir}\live\Kneeboard
  before:
    - copy: { from: '${config_dir}\absent.txt', to: '${config_dir}\x.txt' }
    - delete: ${config_dir}\Temp
  after:
    - exec: [ backup.cmd, --all ]
`)
	dir := filepath.Dir(p)
	for _, d := range []string{`prof\Input`, `live\Input`} {
		os.MkdirAll(filepath.Join(dir, d), 0o755)
	}
	os.WriteFile(filepath.Join(dir, `prof\options.lua`), nil, 0o644)
	c, err := Load(p, "")
	if err != nil {
		t.Fatal(err)
	}
	o := c.Overview()

	want := []SyncView{
		{Source: dir + `\prof\options.lua`, Target: dir + `\live\options.lua`},
		{Source: dir + `\prof\Input`, Target: dir + `\live\Input`, SourceDir: true, TargetDir: true},
		{Source: dir + `\prof\Kneeboard`, Target: dir + `\live\Kneeboard`, Missing: true},
	}
	if len(o.Sync) != len(want) {
		t.Fatalf("sync = %+v", o.Sync)
	}
	for i := range want {
		if o.Sync[i] != want[i] {
			t.Errorf("sync[%d] = %+v, attendu %+v", i, o.Sync[i], want[i])
		}
	}
	if len(o.Before) != 2 || o.Before[0].Kind != "copy" || !o.Before[0].Missing || o.Before[1].Kind != "delete" || o.Before[1].Path != dir+`\Temp` {
		t.Errorf("before = %+v", o.Before)
	}
	if len(o.After) != 1 || o.After[0].Kind != "exec" || strings.Join(o.After[0].Args, " ") != "backup.cmd --all" {
		t.Errorf("after = %+v", o.After)
	}
	if o.Missing != 2 || o.Follows != "game.exe" || o.IfRunning != IfRunningAbort {
		t.Errorf("missing = %d, follows = %q, if running = %q", o.Missing, o.Follows, o.IfRunning)
	}
	// pm check prints the same overview: both missing sources marked.
	if n := strings.Count(c.Plan(), "SOURCE ABSENTE"); n != 2 {
		t.Errorf("%d sources marquées absentes dans :\n%s", n, c.Plan())
	}
}
