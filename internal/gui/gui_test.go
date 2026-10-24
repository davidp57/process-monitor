package gui

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/davidp57/process-monitor/internal/config"
	"github.com/davidp57/process-monitor/internal/control"
	"github.com/davidp57/process-monitor/internal/i18n"
	"github.com/davidp57/process-monitor/internal/logx"
	"github.com/davidp57/process-monitor/internal/registry"
	"github.com/davidp57/process-monitor/internal/session"
	"github.com/davidp57/process-monitor/internal/shortcut"
	"github.com/davidp57/process-monitor/internal/textdiff"
	"github.com/davidp57/process-monitor/internal/wizard"
)

// The test binary doubles as the supervised program, driven by PM_HELPER:
//
//	sleep:MS   sleep MS milliseconds, exit 0
func TestMain(m *testing.M) {
	mode, arg, _ := strings.Cut(os.Getenv("PM_HELPER"), ":")
	switch mode {
	case "":
		i18n.Set("fr")
		os.Exit(m.Run())
	case "sleep":
		ms, _ := strconv.Atoi(arg)
		time.Sleep(time.Duration(ms) * time.Millisecond)
		os.Exit(0)
	}
	os.Exit(98)
}

// harness is a running interface with a browser that opened its page.
type harness struct {
	t        *testing.T
	base     string
	client   *http.Client
	launched [][]string
}

func start(t *testing.T) *harness {
	t.Helper()
	t.Cleanup(registry.UseFile(filepath.Join(t.TempDir(), "configs.json")))
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	h := &harness{t: t, base: "http://127.0.0.1:" + strconv.Itoa(port)}
	srv := New(Options{Port: port, Token: "secret", Self: `C:\pm\pm.exe`,
		Launch:   func(args []string) error { h.launched = append(h.launched, args); return nil },
		Defaults: func() []string { return nil }})
	ts := &httptest.Server{Listener: ln, Config: &http.Server{Handler: srv.Handler()}}
	ts.Start()
	t.Cleanup(ts.Close)
	jar, _ := cookiejar.New(nil)
	h.client = &http.Client{Jar: jar}
	resp, err := h.client.Get(h.base + "/?token=secret") // the page opened as pm gui opens it
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("ouverture : %v %v", resp, err)
	}
	resp.Body.Close()
	return h
}

func (h *harness) call(method, path string, body any, out any) error {
	h.t.Helper()
	var r io.Reader
	if body != nil {
		data, _ := json.Marshal(body)
		r = strings.NewReader(string(data))
	}
	req, _ := http.NewRequest(method, h.base+path, r)
	req.Header.Set("X-PM", "1")
	resp, err := h.client.Do(req)
	if err != nil {
		h.t.Fatal(err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		var e struct{ Error string }
		json.Unmarshal(data, &e)
		return fmt.Errorf("%d %s", resp.StatusCode, e.Error)
	}
	if out != nil {
		return json.Unmarshal(data, out)
	}
	return nil
}

func put(t *testing.T, p, s string) {
	t.Helper()
	os.MkdirAll(filepath.Dir(p), 0o755)
	if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
		t.Fatal(err)
	}
}

// Only the browser pm opened gets in: with the token first, then its
// cookie; writes need the header a form from another site cannot carry.
func TestGuard(t *testing.T) {
	h := start(t)
	status := func(req *http.Request, client *http.Client) int {
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		return resp.StatusCode
	}
	get := func(path string) *http.Request { r, _ := http.NewRequest("GET", h.base+path, nil); return r }
	if s := status(get("/api/ping"), http.DefaultClient); s != http.StatusForbidden {
		t.Errorf("sans cookie : %d", s)
	}
	if s := status(get("/?token=wrong"), http.DefaultClient); s != http.StatusForbidden {
		t.Errorf("mauvais jeton : %d", s)
	}
	if s := status(get("/api/ping"), h.client); s != http.StatusOK {
		t.Errorf("avec cookie : %d", s)
	}
	evil := get("/api/ping")
	evil.Host = "evil.example:" + strings.Split(h.base, ":")[2]
	if s := status(evil, h.client); s != http.StatusForbidden {
		t.Errorf("autre nom : %d", s)
	}
	post, _ := http.NewRequest("POST", h.base+"/api/configs/add", strings.NewReader(`{"path":"x"}`))
	if s := status(post, h.client); s != http.StatusForbidden {
		t.Errorf("écriture sans en-tête : %d", s)
	}
	u, _ := url.Parse(h.base)
	for _, c := range h.client.Jar.Cookies(u) {
		if c.Name == cookie && c.Value != "secret" {
			t.Errorf("cookie %q", c.Value)
		}
	}
}

const dcsConfig = `# DCS by profile
name: dcs
command: C:\Windows\System32\cmd.exe
profiles:
  DEV: {}
  VR: {}
`

func TestConfigs(t *testing.T) {
	h := start(t)
	d := t.TempDir()
	cfg := filepath.Join(d, "pm-dcs.yaml")
	put(t, cfg, dcsConfig)
	put(t, filepath.Join(d, "Icons", "dcs-dev.ico"), "")
	folder := filepath.Join(d, "more")
	put(t, filepath.Join(folder, "sim.yaml"), "command: C:\\Windows\\System32\\cmd.exe\nshortcuts: { mode: watch }\n")
	put(t, filepath.Join(folder, "foreign.yaml"), "something: else\n")

	if err := h.call("POST", "/api/configs/add", map[string]string{"path": cfg}, nil); err != nil {
		t.Fatal(err)
	}
	if err := h.call("POST", "/api/folders/add", map[string]string{"path": folder}, nil); err != nil {
		t.Fatal(err)
	}
	if err := h.call("POST", "/api/configs/add", map[string]string{"path": filepath.Join(d, "nope.yaml")}, nil); err == nil {
		t.Error("config absente ajoutée")
	}
	var list struct {
		Configs []configView
		Folders []string
	}
	h.call("GET", "/api/configs", nil, &list)
	// The foreign .yaml of the folder is not pm's: not listed.
	byName := map[string]configView{}
	for _, c := range list.Configs {
		byName[c.Name] = c
	}
	if len(list.Configs) != 2 || len(byName["dcs"].Profiles) != 2 || !byName["dcs"].Remembered || byName["sim"].Mode != "watch" || byName["sim"].Remembered {
		t.Fatalf("%+v", list)
	}

	// Launch: pm.exe run, as its own process.
	if err := h.call("POST", "/api/configs/launch", map[string]string{"path": cfg, "profile": "VR", "mode": "run"}, nil); err != nil {
		t.Fatal(err)
	}
	if len(h.launched) != 1 || strings.Join(h.launched[0], " ") != `C:\pm\pm.exe run `+cfg+` --profile VR` {
		t.Errorf("lancé : %q", h.launched)
	}
	var plan struct{ Plan string }
	if err := h.call("GET", "/api/configs/check?"+url.Values{"path": {cfg}, "profile": {"DEV"}}.Encode(), nil, &plan); err != nil || !strings.Contains(plan.Plan, "cmd.exe") {
		t.Errorf("check : %v %q", err, plan.Plan)
	}

	// Shortcuts: what is missing, then all of it at once.
	var sp shortcutPlan
	h.call("GET", "/api/configs/shortcuts?path="+url.QueryEscape(cfg), nil, &sp)
	if sp.HasSection || sp.HasUI || len(sp.Icons) != 1 || sp.Profiles[0].Suggest != filepath.Join(d, "Icons", "dcs-dev.ico") || sp.Profiles[1].Suggest != "" {
		t.Fatalf("%+v", sp)
	}
	links := filepath.Join(d, "links")
	var made struct{ Written []shortcut.Written }
	err := h.call("POST", "/api/configs/shortcuts", map[string]any{"path": cfg, "mode": "run", "folders": []string{links},
		"ui": "tray", "icons": map[string]string{"DEV": sp.Profiles[0].Suggest}}, &made)
	if err != nil || len(made.Written) != 2 || made.Written[0].Err != "" {
		t.Fatalf("%+v %v", made, err)
	}
	if l, err := shortcut.Read(filepath.Join(links, "dcs - DEV.lnk")); err != nil || !l.Minimized || l.Icon != sp.Profiles[0].Suggest || l.Target != `C:\pm\pm.exe` {
		t.Errorf("%+v %v", l, err)
	}

	// Editing: a valid change is saved with a .bak; a change pm refuses is not.
	var ed struct{ Doc map[string]any }
	h.call("GET", "/api/configs/edit?path="+url.QueryEscape(cfg), nil, &ed)
	ed.Doc["track"] = "dcs.exe"
	if err := h.call("POST", "/api/configs/edit", map[string]any{"path": cfg, "doc": ed.Doc}, nil); err != nil {
		t.Fatal(err)
	}
	if c, err := config.Load(cfg, "DEV"); err != nil || c.Track != "dcs.exe" {
		t.Errorf("après édition : %+v %v", c, err)
	}
	if b, _ := os.ReadFile(cfg + ".bak"); !strings.Contains(string(b), "# DCS by profile") || strings.Contains(string(b), "track") {
		t.Errorf(".bak : %s", b)
	}
	if b, _ := os.ReadFile(cfg); !strings.Contains(string(b), "# DCS by profile") {
		t.Errorf("commentaire perdu : %s", b)
	}
	before, _ := os.ReadFile(cfg)
	ed.Doc["if_running"] = "explode"
	if err := h.call("POST", "/api/configs/edit", map[string]any{"path": cfg, "doc": ed.Doc}, nil); err == nil || !strings.Contains(err.Error(), "if_running") {
		t.Errorf("config invalide acceptée : %v", err)
	}
	if after, _ := os.ReadFile(cfg); string(after) != string(before) {
		t.Error("fichier changé malgré le refus")
	}

	// Forget and remove a folder: the files stay.
	h.call("POST", "/api/configs/forget", map[string]string{"path": cfg}, nil)
	h.call("POST", "/api/folders/remove", map[string]string{"path": folder}, nil)
	h.call("GET", "/api/configs", nil, &list)
	if len(list.Configs) != 0 {
		t.Errorf("%+v", list.Configs)
	}
	if _, err := os.Stat(cfg); err != nil {
		t.Error("config supprimée")
	}
}

// A session started anywhere shows in the interface, and stops from it.
func TestSessions(t *testing.T) {
	h := start(t)
	self, _ := os.Executable()
	exe := filepath.Join(t.TempDir(), fmt.Sprintf("pm-gui-%d.exe", os.Getpid()))
	data, _ := os.ReadFile(self)
	os.WriteFile(exe, data, 0o755)
	c := &config.Config{Name: filepath.Base(exe), Command: exe, Env: map[string]string{"PM_HELPER": "sleep:30000"}, IfRunning: config.IfRunningAbort}
	log, _ := logx.New("", "", 0, nil)
	s := session.New(c, log, config.RestartNever)
	ctl, err := control.Serve(s, log, control.Info{Name: c.Name, Config: `C:\cfg\x.yaml`, Mode: "run"})
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() { s.Run(context.Background()); ctl.Close(); close(done) }()
	t.Cleanup(func() { s.Kill() })

	var list []sessionView
	for deadline := time.Now().Add(10 * time.Second); ; time.Sleep(100 * time.Millisecond) {
		list = nil
		h.call("GET", "/api/sessions", nil, &list)
		found := false
		for _, v := range list {
			found = found || (v.Name == c.Name && v.State.Status == session.Running)
		}
		if found {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("session pas listée : %+v", list)
		}
	}
	var lines []logx.Line
	if err := h.call("GET", "/api/sessions/log?name="+url.QueryEscape(c.Name), nil, &lines); err != nil || len(lines) == 0 {
		t.Errorf("journal : %v %v", lines, err)
	}
	if err := h.call("POST", "/api/sessions/command", map[string]string{"name": c.Name, "command": "stop"}, nil); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(15 * time.Second):
		t.Fatal("session pas arrêtée")
	}
	if err := h.call("POST", "/api/sessions/command", map[string]string{"name": c.Name, "command": "stop"}, nil); err == nil {
		t.Error("commande à une session finie acceptée")
	}
}

// The wizard from the page: questions answered, actions run in the
// background, the config and its shortcuts written.
func TestWizard(t *testing.T) {
	h := start(t)
	d := t.TempDir()
	self, _ := os.Executable()
	cfg := filepath.Join(d, "pm", "sim.yaml")
	var v wizardView
	if err := h.call("POST", "/api/wizard/start", map[string]string{"path": cfg}, &v); err != nil || v.Step.ID != "case" {
		t.Fatalf("%+v %v", v, err)
	}
	if err := h.call("POST", "/api/wizard/start", map[string]string{}, nil); err == nil {
		t.Error("second assistant accepté")
	}
	// The page reads answers.length: no answer yet is an empty list, not null.
	var raw map[string]any
	if err := h.call("GET", "/api/wizard", nil, &raw); err != nil {
		t.Fatal(err)
	}
	if a, ok := raw["answers"].([]any); !ok || len(a) != 0 {
		t.Errorf("answers = %#v, attendu []", raw["answers"])
	}
	answer := func(values ...string) {
		t.Helper()
		if err := h.call("POST", "/api/wizard/answer", map[string]any{"values": values}, &v); err != nil {
			t.Fatalf("%s : %v", v.Step.ID, err)
		}
	}
	answer("watch")
	answer(self)
	answer("no")
	answer()
	answer("2s")
	answer("sim")
	answer("")
	if err := h.call("POST", "/api/wizard/back", nil, &v); err != nil || v.Step.ID != "config" {
		t.Fatalf("retour : %+v %v", v.Step, err)
	}
	answer("")
	answer("${config_dir}")
	answer("yes")
	if v.Step.ID != "apply" {
		t.Fatalf("%+v", v.Step)
	}
	h.call("POST", "/api/wizard/run", nil, &v)
	for deadline := time.Now().Add(10 * time.Second); v.Busy || !v.Done; time.Sleep(50 * time.Millisecond) {
		h.call("GET", "/api/wizard", nil, &v)
		if time.Now().After(deadline) {
			t.Fatalf("%+v", v)
		}
	}
	if !v.Applied || v.Config != cfg || len(v.Written) != 1 || v.Written[0].Err != "" {
		t.Fatalf("%+v", v)
	}
	if _, err := config.Load(cfg, ""); err != nil {
		t.Error(err)
	}
	if l, err := shortcut.Read(filepath.Join(d, "pm", "sim.lnk")); err != nil || l.Target != `C:\pm\pm.exe` {
		t.Errorf("%+v %v", l, err)
	}
	// The new config is remembered for the list.
	var list struct{ Configs []configView }
	h.call("GET", "/api/configs", nil, &list)
	if len(list.Configs) != 1 || list.Configs[0].Path != cfg {
		t.Errorf("%+v", list.Configs)
	}
	h.call("POST", "/api/wizard/close", nil, &v)
	if v.Active {
		t.Error("assistant pas fermé")
	}
}

// The report's comparison: the lines a profile changed in a text file,
// a binary file by its sizes.
func TestDiffOfAChangedFile(t *testing.T) {
	d := t.TempDir()
	saved := filepath.Join(d, "Saved Games", "DCS")
	opts, auth := filepath.Join(saved, "Config", "options.lua"), filepath.Join(saved, "Config", "keys.bin")
	env := wizard.Env{Home: d, Documents: d, Now: time.Now,
		Exists: func(p string) bool {
			return !strings.HasSuffix(p, `\DEV`) && !strings.HasSuffix(p, ".yaml") && !strings.Contains(p, ".pm-new")
		}}
	w := wizard.New(env, filepath.Join(d, "Profiles", "pm-dcs.yaml"))
	answer := func(v ...string) {
		t.Helper()
		if err := w.Answer(v); err != nil {
			t.Fatalf("%s : %v", w.Next().ID, err)
		}
	}
	answer("dcs")
	answer("")
	answer(saved)
	answer("")
	answer("")
	answer("")
	answer("DEV")
	w.Done(wizard.Snapshot{opts: "o0", auth: "k0"})
	answer(w.Next().Question.Options[1].Value) // no profile is the current setup
	answer("")
	w.Done(map[string]string{opts: "o1", auth: "k1"})
	if w.Next().ID != "review:DEV" {
		t.Fatalf("%+v", w.Next())
	}
	for f, versions := range map[string][2]string{opts: {"a = 1\nb = 2\nc = 3\n", "a = 1\nb = 5\nc = 3\n"}, auth: {"x\x00y", "z\x00w"}} {
		before, after := w.Versions("DEV", f)
		put(t, before, versions[0])
		put(t, after, versions[1])
	}
	got := diffOf(w, "DEV", opts)
	want := []textdiff.Op{{Kind: '=', Text: "a = 1"}, {Kind: '-', Text: "b = 2"}, {Kind: '+', Text: "b = 5"}, {Kind: '=', Text: "c = 3"}}
	if got.Binary || fmt.Sprint(got.Ops) != fmt.Sprint(want) {
		t.Errorf("%+v", got)
	}
	if got := diffOf(w, "DEV", auth); !got.Binary || got.Before != 3 || got.After != 3 || got.Ops != nil {
		t.Errorf("binaire : %+v", got)
	}
}

// Requests at once (a double click on "next", the page polling while an
// answer goes) must not race on the wizard: under -race this fails, and
// without -race Go would kill the interface on concurrent map writes.
func TestWizardRequestsAtOnce(t *testing.T) {
	h := start(t)
	if err := h.call("POST", "/api/wizard/start", map[string]string{}, nil); err != nil {
		t.Fatal(err)
	}
	// Answers and backs at once: each one that succeeds writes the engine.
	done := make(chan struct{})
	for i := 0; i < 40; i++ {
		go func() {
			defer func() { done <- struct{}{} }()
			if i%2 == 0 {
				h.call("POST", "/api/wizard/answer", map[string]any{"values": []string{"watch"}}, nil)
			} else {
				h.call("POST", "/api/wizard/back", nil, nil)
			}
			h.call("GET", "/api/wizard", nil, nil)
		}()
	}
	for i := 0; i < 40; i++ {
		<-done
	}
	var v wizardView
	if err := h.call("GET", "/api/wizard", nil, &v); err != nil || v.Step == nil {
		t.Errorf("%+v %v", v, err)
	}
}

// The language set in the page is kept, and the page speaks it at once.
func TestLanguageSetting(t *testing.T) {
	h := start(t)
	defer i18n.Set("fr")
	if err := h.call("POST", "/api/lang", map[string]string{"lang": "en"}, nil); err != nil {
		t.Fatal(err)
	}
	var i struct {
		Lang, Setting string
		Messages      map[string]string
	}
	h.call("GET", "/api/i18n", nil, &i)
	if i.Lang != "en" || i.Setting != "en" || i.Messages["gui.save"] != "Save" || registry.Lang() != "en" {
		t.Errorf("%s %s %q", i.Lang, i.Setting, i.Messages["gui.save"])
	}
	if err := h.call("POST", "/api/lang", map[string]string{"lang": "de"}, nil); err == nil {
		t.Error("langue inconnue acceptée")
	}
}

// The folders looked into without being given: next to pm.exe and above,
// and where the wizard puts configs (Saved Games\DCS\Profiles).
func TestDefaultFolders(t *testing.T) {
	d := t.TempDir()
	t.Setenv("USERPROFILE", d)
	exe := filepath.Join(d, "Saved Games", "DCS", "Profiles", "pm", "pm.exe")
	put(t, exe, "")
	got := defaultFolders(exe)
	for _, want := range []string{filepath.Dir(exe), filepath.Join(d, "Saved Games", "DCS", "Profiles")} {
		found := false
		for _, g := range got {
			found = found || g == want
		}
		if !found {
			t.Errorf("%s absent de %v", want, got)
		}
	}
	for _, g := range got {
		if strings.Contains(g, "DCS.openbeta") {
			t.Errorf("dossier absent proposé : %s", g)
		}
	}
}

// The interface may stop once no page has been open for the grace, unless
// a wizard is under way.
func TestIdle(t *testing.T) {
	t.Cleanup(registry.UseFile(filepath.Join(t.TempDir(), "configs.json")))
	srv := New(Options{Port: 1, Token: "x", Defaults: func() []string { return nil }})
	ts := httptest.NewServer(http.HandlerFunc(srv.alive))
	defer ts.Close()
	later := time.Now().Add(time.Hour)
	if srv.Idle(time.Now(), 30*time.Second) {
		t.Error("au démarrage, avant la première page")
	}
	ctx, cancel := context.WithCancel(context.Background())
	req, _ := http.NewRequestWithContext(ctx, "GET", ts.URL, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if srv.Idle(later, 30*time.Second) {
		t.Error("une page est ouverte")
	}
	cancel()
	resp.Body.Close()
	for deadline := time.Now().Add(5 * time.Second); !srv.Idle(later, 30*time.Second); time.Sleep(20 * time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatal("jamais inactive après la fermeture de la page")
		}
	}
	if srv.Idle(time.Now(), 30*time.Second) {
		t.Error("avant la fin du délai")
	}
	wr, err := newWizard("", "")
	if err != nil {
		t.Fatal(err)
	}
	srv.wizard = wr
	if srv.Idle(later, 30*time.Second) {
		t.Error("un assistant est en cours")
	}
}
