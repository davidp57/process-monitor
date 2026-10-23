package tray

import (
	"context"
	"fmt"
	"sync"
	"time"

	"fyne.io/systray"
	"golang.org/x/sys/windows"

	"github.com/davidp57/process-monitor/internal/logx"
	"github.com/davidp57/process-monitor/internal/session"
)

// Run shows the session as a notification-area icon until it ends. It must
// be called from the main goroutine. A failing session ends with a message
// box: there is no console left to show the error.
func Run(ctx context.Context, s *session.Session, log *logx.Logger, watch bool) error {
	cfg := s.Config()
	var (
		mu     sync.Mutex
		runErr error
	)
	result := func() error {
		mu.Lock()
		defer mu.Unlock()
		return runErr
	}
	failed := func() bool { return result() != nil || log.Errors() > 0 }

	onReady := func() {
		systray.SetTitle("pm")
		status := systray.AddMenuItem("", "")
		status.Disable()
		systray.AddSeparator()
		openLog := systray.AddMenuItem("Ouvrir le journal", log.Path)
		if log.Path == "" {
			openLog.Disable()
		}
		var pause, resume, restart *systray.MenuItem
		if watch {
			pause = systray.AddMenuItem("Mettre en pause", "")
			resume = systray.AddMenuItem("Reprendre", "")
			restart = systray.AddMenuItem("Relancer", "")
		}
		systray.AddSeparator()
		stop := systray.AddMenuItem("Arrêter et finir", "Arrête le programme, puis exécute les actions de fin")
		detach := systray.AddMenuItem("Détacher…", "pm s'arrête, le programme continue")

		var last View
		show := func() {
			v := ViewOf(cfg.Name, cfg.Profile, s.State(), watch, failed())
			if v.Color != last.Color || last.Tooltip == "" {
				systray.SetIcon(Icon(v.Color))
			}
			systray.SetTooltip(v.Tooltip)
			status.SetTitle(v.Status)
			enable(stop, v.CanStop)
			enable(detach, v.CanDetach)
			if watch {
				enable(pause, v.CanPause)
				enable(resume, v.CanResume)
				enable(restart, v.CanRestart)
			}
			last = v
		}
		show()

		ended := make(chan struct{})
		go func() {
			err := s.Run(ctx)
			if err != nil {
				log.Errorf("session terminée en erreur : %v", err)
			}
			mu.Lock()
			runErr = err
			mu.Unlock()
			close(ended)
		}()

		go func() {
			tick := time.NewTicker(500 * time.Millisecond)
			defer tick.Stop()
			for {
				select {
				case <-tick.C:
					show()
				case <-openLog.ClickedCh:
					shellOpen(log.Path)
				case <-clicked(pause):
					s.Pause()
				case <-clicked(resume):
					s.Resume()
				case <-clicked(restart):
					s.Restart()
				case <-stop.ClickedCh:
					s.Kill()
				case <-detach.ClickedCh:
					if ask(cfg.Name, DetachQuestion(s.State().Status)) {
						s.Detach()
					}
				case <-ended:
					show()
					if failed() {
						msg := "La session s'est terminée en erreur."
						if err := result(); err != nil {
							msg = err.Error()
						}
						if log.Path != "" {
							msg += "\n\nJournal : " + log.Path
						}
						alert(cfg.Name, msg)
					}
					systray.Quit()
					return
				}
			}
		}()
	}
	systray.Run(onReady, func() {})
	return result()
}

func enable(m *systray.MenuItem, on bool) {
	if on {
		m.Enable()
	} else {
		m.Disable()
	}
}

// clicked returns the item's click channel, or nil (never ready) when the
// item does not exist.
func clicked(m *systray.MenuItem) <-chan struct{} {
	if m == nil {
		return nil
	}
	return m.ClickedCh
}

const (
	mbYesNo         = 0x00000004
	mbIconError     = 0x00000010
	mbIconQuestion  = 0x00000020
	mbSetForeground = 0x00010000
	mbTopMost       = 0x00040000
	idYes           = 6
)

func box(title, text string, flags uint32) int32 {
	t, _ := windows.UTF16PtrFromString("pm — " + title)
	m, _ := windows.UTF16PtrFromString(text)
	r, _ := windows.MessageBox(0, m, t, flags|mbSetForeground|mbTopMost)
	return r
}

func ask(title, question string) bool {
	return box(title, question, mbYesNo|mbIconQuestion) == idYes
}

func alert(title, text string) {
	box(title, text, mbIconError)
}

func shellOpen(path string) {
	if path == "" {
		return
	}
	verb, _ := windows.UTF16PtrFromString("open")
	file, _ := windows.UTF16PtrFromString(path)
	if err := windows.ShellExecute(0, verb, file, nil, nil, windows.SW_SHOWNORMAL); err != nil {
		alert("journal", fmt.Sprintf("Impossible d'ouvrir %s : %v", path, err))
	}
}
