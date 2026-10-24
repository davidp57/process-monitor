package gui

import (
	"encoding/json"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
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

// Run is pm gui: the interface's server and its notification icon, until
// "quit" from the icon. When one already runs, it only opens its page.
func Run(self string) error {
	if in, ok := running(); ok {
		return open(in.url())
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	token, err := control.Token()
	if err != nil {
		return err
	}
	in := instance{Port: ln.Addr().(*net.TCPAddr).Port, Token: token, PID: os.Getpid()}
	srv := New(Options{Port: in.Port, Token: token, Self: self})
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
	defer os.Remove(path)
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
					systray.Quit()
					return
				}
			}
		}()
	}, func() {})
	return nil
}

// open shows url in the default browser.
func open(url string) error { return tray.Open(url) }
