// Package tray shows a session as an icon in the Windows notification area,
// for pm run and pm watch without a console window.
package tray

import (
	"fmt"
	"time"

	"github.com/davidp57/process-monitor/internal/session"
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
	return [...]string{"vert", "orange", "gris", "rouge"}[c]
}

// View is what the icon and its menu show for one session state.
type View struct {
	Color   Color
	Tooltip string // at most 127 characters, the Windows limit
	Status  string // the first, disabled line of the menu

	CanPause, CanResume, CanRestart, CanStop, CanDetach bool
}

// ViewOf maps a session state to the icon. watch enables the pause and
// restart entries; failed turns a finished session red.
func ViewOf(name, profile string, st session.State, watch, failed bool) View {
	v := View{Color: Orange}
	active := st.Status == session.Running || st.Status == session.Paused || st.Status == session.Restarting
	switch st.Status {
	case session.Starting:
		v.Status = "démarrage : copie des fichiers"
	case session.Running:
		v.Color = Green
		v.Status = fmt.Sprintf("en cours, pid %d", st.PID)
		if st.Ready != "" {
			v.Status += " (" + st.Ready + ")"
		}
	case session.Paused:
		v.Color = Grey
		v.Status = fmt.Sprintf("en pause, pid %d", st.PID)
	case session.Restarting:
		v.Status = "relance"
		if !st.NextStart.IsZero() {
			v.Status += " dans " + time.Until(st.NextStart).Round(time.Second).String()
		}
	case session.Stopping:
		v.Status = "actions de fin en cours"
	case session.Finished, session.Detached:
		v.Color = Green
		v.Status = string(st.Status)
	default:
		v.Status = string(st.Status)
	}
	if failed && st.Status == session.Finished {
		v.Color = Red
		v.Status += " en erreur"
	}
	if st.Restarts > 0 {
		v.Status += fmt.Sprintf(", %d relance(s)", st.Restarts)
	}

	v.CanPause = watch && st.Status == session.Running
	v.CanResume = watch && st.Status == session.Paused
	v.CanRestart = watch && active
	v.CanStop = active || st.Status == session.Starting
	v.CanDetach = v.CanStop

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
		return "Annuler la session ? Rien ne sera lancé, les fichiers live seront remis comme avant."
	case session.Restarting:
		return "Terminer sans relancer ? Rien ne tourne : les actions de fin seront exécutées."
	default:
		return "Détacher pm ? Le programme continue sans surveillance, et les actions de fin (copies retour comprises) ne seront PAS exécutées."
	}
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}
