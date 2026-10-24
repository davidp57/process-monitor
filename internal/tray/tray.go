package tray

import (
	"context"
	"errors"
	"image"
	"sync"
	"time"

	"fyne.io/systray"
	"golang.org/x/sys/windows"

	"github.com/davidp57/process-monitor/internal/config"
	"github.com/davidp57/process-monitor/internal/logx"
	"github.com/davidp57/process-monitor/internal/session"

	"github.com/davidp57/process-monitor/internal/i18n"
)

// Run shows the session as a notification-area icon until it ends. It must
// be called from the main goroutine. base is the custom image (see
// LoadImage), with a state dot drawn over it; nil shows a coloured disc.
// A failing session ends with a message box: there is no console left.
func Run(ctx context.Context, s *session.Session, log *logx.Logger, watch bool, base image.Image) error {
	cfg := s.Config()
	icons := map[Color][]byte{}
	for _, c := range []Color{Green, Orange, Grey, Red} {
		if base != nil {
			icons[c] = Badged(base, c)
		} else {
			icons[c] = Icon(c)
		}
	}
	var (
		mu     sync.Mutex
		runErr error
		done   bool
	)
	result := func() (error, bool) {
		mu.Lock()
		defer mu.Unlock()
		return runErr, done
	}
	// failed judges the outcome once the session has ended: an error other
	// than the user's own cancellation, or, for pm run, a program that
	// exited non-zero on its own (not stopped from the menu). Errors logged
	// along the way and recovered from (a watch restart) do not count.
	failed := func() bool {
		err, ended := result()
		switch {
		case !ended:
			return false
		case err != nil:
			return !errors.Is(err, session.ErrCancelled)
		default:
			return !watch && s.State().ExitCode != 0 && !s.State().Stopped
		}
	}

	onReady := func() {
		systray.SetTitle("pm")
		status := systray.AddMenuItem("", "")
		status.Disable()
		systray.AddSeparator()
		openLog := systray.AddMenuItem(i18n.T("tray.open_log"), log.Path)
		if log.Path == "" {
			openLog.Disable()
		}
		var pause, resume, restart *systray.MenuItem
		if watch {
			pause = systray.AddMenuItem(i18n.T("tray.pause"), "")
			resume = systray.AddMenuItem(i18n.T("tray.resume"), "")
			restart = systray.AddMenuItem(i18n.T("tray.restart"), "")
		}
		systray.AddSeparator()
		stop := systray.AddMenuItem(i18n.T("tray.stop"), i18n.T("tray.stop_hint"))
		detach := systray.AddMenuItem(i18n.T("tray.detach"), i18n.T("tray.detach_hint"))

		var last View
		show := func() {
			v := ViewOf(cfg.Name, cfg.Profile, s.State(), watch, failed())
			if v.Color != last.Color || last.Tooltip == "" {
				systray.SetIcon(icons[v.Color])
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
				log.Error(i18n.T("run.ended_error", err))
			}
			mu.Lock()
			runErr, done = err, true
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
					// The box stays open as long as the user likes; if the
					// session moved on meanwhile, the answer was to another
					// question: ask again.
					for asked := s.State().Status; ask(cfg.Name, DetachQuestion(asked)); asked = s.State().Status {
						if s.State().Status == asked {
							s.Detach()
							break
						}
					}
				case <-ended:
					show()
					err, _ := result()
					switch {
					case failed():
						msg := i18n.T("tray.exit_code", s.State().ExitCode)
						if err != nil {
							msg = err.Error()
						}
						if log.Path != "" {
							msg += "\n\n" + i18n.T("tray.log_path", log.Path)
						}
						alert(cfg.Name, msg)
					case err != nil: // cancelled by the user: nothing to report
					case cfg.Exit.WaitKey == config.WaitKeyAlways:
						inform(cfg.Name, i18n.T("tray.session_over"))
					default:
						time.Sleep(cfg.Exit.Delay.D()) // the final state stays visible meanwhile
					}
					systray.Quit()
					return
				}
			}
		}()
	}
	systray.Run(onReady, func() {})
	err, _ := result()
	return err
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
	mbYesNo           = 0x00000004
	mbIconError       = 0x00000010
	mbIconQuestion    = 0x00000020
	mbIconInformation = 0x00000040
	mbSetForeground   = 0x00010000
	mbTopMost         = 0x00040000
	idYes             = 6
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

func inform(title, text string) {
	box(title, text, mbIconInformation)
}

// Alert shows an error in a message box, for when pm has no console.
func Alert(title, text string) { alert(title, text) }

func alert(title, text string) {
	box(title, text, mbIconError)
}

// Open shows a file or a web address with the program Windows has for it.
func Open(target string) error {
	verb, _ := windows.UTF16PtrFromString("open")
	file, _ := windows.UTF16PtrFromString(target)
	return windows.ShellExecute(0, verb, file, nil, nil, windows.SW_SHOWNORMAL)
}

func shellOpen(path string) {
	if path == "" {
		return
	}
	if err := Open(path); err != nil {
		alert("log", i18n.T("tray.open_failed", path, err))
	}
}
