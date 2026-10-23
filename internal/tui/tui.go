// Package tui is the terminal interface of pm watch.
package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/davidp57/process-monitor/internal/config"
	"github.com/davidp57/process-monitor/internal/logx"
	"github.com/davidp57/process-monitor/internal/session"
)

const maxLines = 2000

var (
	title   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("13"))
	key     = lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true)
	label   = lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Width(14)
	dim     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	box     = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("8")).Padding(0, 1)
	byLevel = map[logx.Level]lipgloss.Style{
		logx.Warn:  lipgloss.NewStyle().Foreground(lipgloss.Color("11")),
		logx.Error: lipgloss.NewStyle().Foreground(lipgloss.Color("9")),
		logx.Out:   lipgloss.NewStyle().Foreground(lipgloss.Color("7")),
	}
	byStatus = map[session.Status]lipgloss.Style{
		session.Running:    lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true),
		session.Paused:     lipgloss.NewStyle().Foreground(lipgloss.Color("208")).Bold(true),
		session.Restarting: lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Bold(true),
	}
)

type lineMsg logx.Line
type tickMsg time.Time
type doneMsg struct{ err error }

type model struct {
	s       *session.Session
	cfg     *config.Config
	logPath string
	lines   []string
	view    viewport.Model
	width   int
	height  int
	ready   bool

	finished bool
	err      error
	errors   func() int
	closeAt  time.Time
	waitKey  bool
	confirmQ bool
	follow   bool // keep the log scrolled to the newest line
}

// Run shows the interface until the session ends (or the user detaches).
func Run(ctx context.Context, s *session.Session, log *logx.Logger) error {
	m := &model{s: s, cfg: s.Config(), logPath: log.Path, errors: log.Errors, follow: true}
	prog := tea.NewProgram(m, tea.WithAltScreen())
	log.SetSink(func(l logx.Line) { prog.Send(lineMsg(l)) })
	ended := make(chan struct{})
	go func() {
		defer close(ended)
		err := s.Run(ctx)
		if err != nil {
			log.Errorf("session terminée en erreur : %v", err) // reaches both the file and the screen
		}
		prog.Send(doneMsg{err})
	}()
	_, err := prog.Run()
	log.SetSink(nil)
	if err != nil {
		// The interface is gone: do not leave the session running behind pm's back.
		log.Errorf("interface en échec (%v) : session détachée", err)
		s.Detach()
		<-ended
		return err
	}
	return m.err
}

func tick() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (m *model) Init() tea.Cmd { return tick() }

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		if !m.ready {
			m.view = viewport.New(msg.Width, 3)
			m.ready = true
		}
		m.view.Width = msg.Width
		m.refresh()
	case lineMsg:
		l := logx.Line(msg)
		text := l.String()
		if st, ok := byLevel[l.Level]; ok {
			text = st.Render(text)
		}
		m.lines = append(m.lines, text)
		if len(m.lines) > maxLines {
			m.lines = m.lines[len(m.lines)-maxLines:]
		}
		m.refresh()
	case tickMsg:
		if m.finished && !m.waitKey && time.Now().After(m.closeAt) {
			return m, tea.Quit
		}
		return m, tick()
	case doneMsg:
		m.finished, m.err = true, msg.err
		failed := msg.err != nil || m.errors() > 0
		m.waitKey = m.cfg.Exit.WaitKey == config.WaitKeyAlways || (m.cfg.Exit.WaitKey == config.WaitKeyOnError && failed)
		m.closeAt = time.Now().Add(m.cfg.Exit.Delay.D())
		m.refresh()
	case tea.KeyMsg:
		if cmd, handled := m.onKey(msg.String()); handled {
			return m, cmd
		}
		var cmd tea.Cmd
		m.view, cmd = m.view.Update(msg) // arrows, page up/down: scroll the log
		m.follow = m.view.AtBottom()
		return m, cmd
	}
	var cmd tea.Cmd
	m.view, cmd = m.view.Update(msg)
	return m, cmd
}

// onKey handles the command keys; handled is false for keys meant for the log view.
func (m *model) onKey(k string) (cmd tea.Cmd, handled bool) {
	if m.finished {
		if k == "q" || k == "ctrl+c" || k == "enter" || k == "esc" || m.waitKey {
			return tea.Quit, true
		}
		return nil, false
	}
	if k != "q" && k != "ctrl+c" {
		m.confirmQ = false
	}
	switch k {
	case "p":
		if m.s.State().Status == session.Paused {
			m.s.Resume()
		} else {
			m.s.Pause()
		}
	case "r":
		m.s.Restart()
	case "k":
		m.s.Kill()
	case "q", "ctrl+c":
		// Detaching skips the after hooks: ask twice.
		if !m.confirmQ {
			m.confirmQ = true
			return nil, true
		}
		m.s.Detach()
	default:
		return nil, false
	}
	return nil, true
}

func (m *model) refresh() {
	if !m.ready {
		return
	}
	m.view.SetContent(strings.Join(m.lines, "\n"))
	if m.follow {
		m.view.GotoBottom()
	}
}

func (m *model) header() string {
	st := m.s.State()
	c := m.cfg
	name := c.Name
	if c.Profile != "" {
		name += " / " + c.Profile
	}
	status := string(st.Status)
	if style, ok := byStatus[st.Status]; ok {
		status = style.Render(status)
	}
	rows := [][2]string{{"Statut", status}}
	if st.PID != 0 {
		rows = append(rows, [2]string{"PID", fmt.Sprint(st.PID)})
	}
	if !st.StartedAt.IsZero() && st.Status != session.Restarting {
		rows = append(rows, [2]string{"Depuis", time.Since(st.StartedAt).Round(time.Second).String()})
	}
	if !st.NextStart.IsZero() && st.Status == session.Restarting {
		rows = append(rows, [2]string{"Relance dans", time.Until(st.NextStart).Round(time.Second).String()})
	}
	rows = append(rows, [2]string{"Relances", fmt.Sprint(st.Restarts)})
	if st.Ready != "" {
		rows = append(rows, [2]string{"Démarrage", st.Ready})
	}
	if st.LastExit != "" {
		rows = append(rows, [2]string{"Dernière fin", st.LastExit})
	}
	rows = append(rows, [2]string{"Programme", c.Command})
	if c.Track != "" {
		rows = append(rows, [2]string{"Suivi", c.Track})
	}
	if m.logPath != "" {
		rows = append(rows, [2]string{"Journal", m.logPath})
	}
	var b strings.Builder
	for i, r := range rows {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(label.Render(r[0]) + r[1])
	}
	width := max(m.width-2, 20)
	return title.Render("pm watch — "+name) + "\n" + box.Width(width).Render(b.String())
}

func (m *model) footer() string {
	switch {
	case m.finished && m.waitKey:
		return key.Render("une touche") + dim.Render(" pour fermer")
	case m.finished:
		return dim.Render(fmt.Sprintf("fermeture dans %s — ", time.Until(m.closeAt).Round(time.Second))) + key.Render("q") + dim.Render(" fermer")
	case m.confirmQ && m.s.State().Status == session.Restarting:
		// Between two runs nothing is left to detach from: quitting ends the session.
		return key.Render("q") + dim.Render(" encore une fois pour terminer sans relancer : les actions de fin seront exécutées")
	case m.confirmQ:
		return key.Render("q") + dim.Render(" encore une fois pour détacher : le programme continue, les actions de fin ne seront PAS exécutées")
	}
	pause := "pause"
	if m.s.State().Status == session.Paused {
		pause = "reprendre"
	}
	return strings.Join([]string{
		key.Render("p") + dim.Render(" "+pause),
		key.Render("r") + dim.Render(" relancer"),
		key.Render("k") + dim.Render(" arrêter et finir"),
		key.Render("q") + dim.Render(" détacher"),
	}, dim.Render("  ·  "))
}

func (m *model) View() string {
	if !m.ready {
		return "…"
	}
	header := m.header()
	// The header grows and shrinks with the state; the log takes what is left.
	m.view.Height = max(m.height-lipgloss.Height(header)-2, 3)
	if m.follow {
		m.view.GotoBottom()
	}
	return header + "\n" + m.view.View() + "\n" + m.footer()
}
