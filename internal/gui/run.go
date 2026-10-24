package gui

import (
	"encoding/json"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"fyne.io/systray"

	"github.com/davidp57/process-monitor/internal/control"
	"github.com/davidp57/process-monitor/internal/i18n"
	"github.com/davidp57/process-monitor/internal/tray"
)

// instance is what the running interface writes, so a second pm gui only
// opens the page: %LOCALAPPDATA%\pm\gui.json, readable by this account only.
type instance struct {
	Port  int    `json:"port"`
	Token string `json:"token"`
	PID   int    `json:"pid"`
	Exe   string `json:"exe"` // the pm.exe serving it
}

func instanceFile() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "pm", "gui.json"), nil
}

func (in instance) url() string {
	return "http://127.0.0.1:" + strconv.Itoa(in.Port) + "/?token=" + in.Token
}

// running finds the interface already running, if any.
func running() (instance, bool) {
	var in instance
	path, err := instanceFile()
	if err != nil {
		return in, false
	}
	data, err := os.ReadFile(path)
	if err != nil || json.Unmarshal(data, &in) != nil {
		return in, false
	}
	req, _ := http.NewRequest("GET", "http://127.0.0.1:"+strconv.Itoa(in.Port)+"/api/ping", nil)
	req.AddCookie(&http.Cookie{Name: cookie, Value: in.Token})
	resp, err := (&http.Client{Timeout: 2 * time.Second}).Do(req)
	if err != nil {
		return in, false
	}
	resp.Body.Close()
	return in, resp.StatusCode == http.StatusOK
}

// sameExe tells whether the interface running is served by self: after an
// update, another pm.exe still serves the old page.
func sameExe(in instance, self string) bool {
	return in.Exe != "" && strings.EqualFold(filepath.Clean(in.Exe), filepath.Clean(self))
}

// replace asks the interface running to stop and waits for it to be gone;
// false when it refuses (a wizard under way, or a pm too old to be asked).
func replace(in instance) bool {
	req, _ := http.NewRequest("POST", "http://127.0.0.1:"+strconv.Itoa(in.Port)+"/api/quit", nil)
	req.AddCookie(&http.Cookie{Name: cookie, Value: in.Token})
	req.Header.Set("X-PM", "1")
	resp, err := (&http.Client{Timeout: 2 * time.Second}).Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false
	}
	for deadline := time.Now().Add(10 * time.Second); time.Now().Before(deadline); time.Sleep(100 * time.Millisecond) {
		if _, ok := running(); !ok {
			return true
		}
	}
	return false
}

// Run is pm gui: the interface's server and its notification icon, until
// "quit" from the icon. When one already runs, it only opens its page; one
// served by another pm.exe is replaced, so an update shows at once.
func Run(self string) error {
	if in, ok := running(); ok {
		if sameExe(in, self) || !replace(in) {
			return open(in.url())
		}
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	token, err := control.Token()
	if err != nil {
		return err
	}
	in := instance{Port: ln.Addr().(*net.TCPAddr).Port, Token: token, PID: os.Getpid(), Exe: self}
	srv := New(Options{Port: in.Port, Token: token, Self: self, Quit: systray.Quit})
	go http.Serve(ln, srv.Handler())
	path, err := instanceFile()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, _ := json.Marshal(in)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return err
	}
	defer removeIfOurs(path, in.PID)
	open(in.url())

	// No page open for a while, nothing under way: the interface goes.
	go func() {
		for range time.Tick(5 * time.Second) {
			if srv.Idle(time.Now(), 30*time.Second) {
				systray.Quit()
				return
			}
		}
	}()

	systray.Run(func() {
		systray.SetIcon(tray.Icon(tray.Green))
		systray.SetTooltip(i18n.T("gui.tray_tooltip"))
		openItem := systray.AddMenuItem(i18n.T("gui.tray_open"), "")
		systray.AddSeparator()
		quit := systray.AddMenuItem(i18n.T("gui.tray_quit"), i18n.T("gui.tray_quit_hint"))
		go func() {
			for {
				select {
				case <-openItem.ClickedCh:
					open(in.url())
				case <-quit.ClickedCh:
					// Not in the middle of a wizard: the game's original
					// files would stay set aside. Its page says how to give up.
					if srv.wizardUnderWay() {
						open(in.url())
						tray.Inform(i18n.T("gui.tray_quit"), i18n.T("gui.quit_refused")+"\n"+i18n.T("gui.quit_refused_hint"))
						continue
					}
					systray.Quit()
					return
				}
			}
		}()
	}, func() {})
	return nil
}

// removeIfOurs removes the instance file unless another interface, the
// one that replaced this one, has written its own there since.
func removeIfOurs(path string, pid int) {
	var in instance
	if data, err := os.ReadFile(path); err == nil && json.Unmarshal(data, &in) == nil && in.PID != pid {
		return
	}
	os.Remove(path)
}

// open shows url in the default browser.
func open(url string) error { return tray.Open(url) }
