// Package tray shows a session as an icon in the Windows notification area,
// for pm run and pm watch without a console window.
package tray

import (
	"time"

	"github.com/davidp57/process-monitor/internal/session"

	"github.com/davidp57/process-monitor/internal/i18n"
)

// Color is the state the icon shows at a glance.
type Color int

const (
	Green  Color = iota // the program runs
	Orange              // copying, waiting, restarting: something is in progress
	Grey                // paused
	Red                 // the session ended in error
)

func (c Color) String() string {
	return [...]string{"green", "orange", "grey", "red"}[c]
}

// View is what the icon and its menu show for one session state.
type View struct {
	Color   Color
	Tooltip string // at most 127 characters, the Windows limit
	Status  string // the first, disabled line of the menu

	CanPause, CanResume, CanRestart, CanStop, CanDetach bool
}

// ViewOf maps a session state to the icon. watch enables the pause and
// restart entries; failed (the session ended in error) turns it red.
func ViewOf(name, profile string, st session.State, watch, failed bool) View {
	v := View{Color: Orange}
	active := st.Status == session.Running || st.Status == session.Paused || st.Status == session.Restarting
	switch st.Status {
	case session.Starting:
		v.Status = i18n.T("tray.status_starting")
	case session.Running:
		v.Color = Green
		v.Status = i18n.T("tray.status_running", st.PID)
		if st.Ready != "" {
			v.Status += " (" + i18n.T(st.Ready) + ")"
		}
	case session.Paused:
		v.Color = Grey
		v.Status = i18n.T("tray.status_paused", st.PID)
	case session.Restarting:
		v.Status = i18n.T("status.restarting")
		if !st.NextStart.IsZero() {
			v.Status += i18n.T("tray.restart_in", time.Until(st.NextStart).Round(time.Second))
		}
	case session.Stopping:
		v.Status = i18n.T("tray.status_stopping")
	case session.Finished, session.Detached:
		v.Color = Green
		v.Status = i18n.T("status." + string(st.Status))
	default:
		v.Status = i18n.T("status." + string(st.Status))
	}
	if failed { // whatever the state the session stopped in
		v.Color = Red
		v.Status += i18n.T("tray.in_error")
	}
	if st.Restarts > 0 {
		v.Status += i18n.T("tray.restarts", st.Restarts)
	}

	v.CanPause = watch && st.Status == session.Running
	v.CanResume = watch && st.Status == session.Paused
	v.CanRestart = watch && active
	v.CanDetach = active || st.Status == session.Starting
	// While stopping, a stop may be retried: Windows can refuse the first.
	v.CanStop = v.CanDetach || st.Status == session.Stopping

	title := "pm — " + name
	if profile != "" {
		title += " / " + profile
	}
	v.Tooltip = truncate(title+"\n"+v.Status, 127)
	return v
}

// DetachQuestion is the confirmation asked before detaching: what it does
// depends on the moment, as in the terminal interface.
func DetachQuestion(st session.Status) string {
	switch st {
	case session.Starting:
		return i18n.T("tray.ask_cancel")
	case session.Restarting:
		return i18n.T("tray.ask_end")
	default:
		return i18n.T("tray.ask_detach")
	}
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}
