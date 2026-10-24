package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/davidp57/process-monitor/internal/i18n"
	"github.com/davidp57/process-monitor/internal/wizard"
)

var (
	prompt  = lipgloss.NewStyle().Bold(true)
	errLine = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	chosen  = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
)

// Answer is what the person did with a question.
type Answer int

// Quit is the zero value: a question that ends without Enter (Ctrl+C, the
// window closed) gives up. Taken as answered, its empty answer would pick
// the suggested value and go on, starting a game nobody asked for.
const (
	Quit     Answer = iota // Ctrl+C, or the question ended any other way
	Answered               // Enter
	Back                   // Esc: the previous question
)

// ask is one question, asked in place: what was answered stays on screen
// once the next question comes.
type ask struct {
	q       *wizard.Question
	errMsg  string
	input   textinput.Model
	entries []string // List
	cursor  int
	ticked  map[string]bool // Multi
	result  Answer
	values  []string
	ended   bool
}

// Ask asks q in the terminal. errMsg, when not empty, says why the previous
// answer was refused.
func Ask(q *wizard.Question, errMsg string) ([]string, Answer, error) {
	m := newAsk(q, errMsg)
	final, err := tea.NewProgram(m).Run()
	if err != nil {
		return nil, Quit, err
	}
	a := final.(*ask)
	return a.values, a.result, nil
}

func newAsk(q *wizard.Question, errMsg string) *ask {
	m := &ask{q: q, errMsg: errMsg, ticked: map[string]bool{}}
	m.input = textinput.New()
	m.input.Prompt = "> "
	if len(q.Default) > 0 && (q.Kind == wizard.Text || q.Kind == wizard.Path) {
		m.input.Placeholder = q.Default[0]
	}
	m.input.Focus()
	for _, d := range q.Default {
		m.ticked[d] = true
		for i, o := range q.Options {
			if o.Value == d && q.Kind != wizard.Multi {
				m.cursor = i
			}
		}
	}
	return m
}

func (m *ask) Init() tea.Cmd { return textinput.Blink }

func (m *ask) choosing() bool {
	return m.q.Kind == wizard.Choice || m.q.Kind == wizard.YesNo || m.q.Kind == wizard.Multi
}

func (m *ask) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	k, ok := msg.(tea.KeyMsg)
	if !ok {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
	switch k.Type {
	case tea.KeyCtrlC:
		return m.end(Quit, nil)
	case tea.KeyEsc:
		return m.end(Back, nil)
	}
	if m.choosing() {
		switch k.String() {
		case "up", "k":
			m.cursor = (m.cursor + len(m.q.Options) - 1) % len(m.q.Options)
		case "down", "j", "tab":
			m.cursor = (m.cursor + 1) % len(m.q.Options)
		case " ":
			if m.q.Kind == wizard.Multi {
				v := m.q.Options[m.cursor].Value
				m.ticked[v] = !m.ticked[v]
			}
		case "enter":
			if m.q.Kind != wizard.Multi {
				return m.end(Answered, []string{m.q.Options[m.cursor].Value})
			}
			values := []string{} // not nil: nothing ticked is an answer
			for _, o := range m.q.Options {
				if m.ticked[o.Value] {
					values = append(values, o.Value)
				}
			}
			return m.end(Answered, values)
		}
		return m, nil
	}
	switch k.Type {
	case tea.KeyEnter:
		v := strings.TrimSpace(m.input.Value())
		if m.q.Kind != wizard.List {
			return m.end(Answered, []string{v})
		}
		if v == "" {
			return m.end(Answered, m.entries)
		}
		m.entries = append(m.entries, v)
		m.input.SetValue("")
		return m, nil
	case tea.KeyBackspace:
		if m.q.Kind == wizard.List && m.input.Value() == "" && len(m.entries) > 0 {
			m.entries = m.entries[:len(m.entries)-1]
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m *ask) end(r Answer, values []string) (tea.Model, tea.Cmd) {
	m.result, m.values, m.ended = r, values, true
	return m, tea.Quit
}

func (m *ask) View() string {
	var b strings.Builder
	if m.ended {
		if m.result != Answered {
			return ""
		}
		b.WriteString(prompt.Render(m.q.Prompt) + " : " + chosen.Render(m.shown()) + "\n")
		return b.String()
	}
	b.WriteString("\n")
	if m.q.Help != "" {
		b.WriteString(dim.Render(strings.TrimRight(m.q.Help, "\n")) + "\n")
	}
	b.WriteString(prompt.Render(m.q.Prompt) + "\n")
	if m.choosing() {
		for i, o := range m.q.Options {
			mark := "  "
			if i == m.cursor {
				mark = key.Render("> ")
			}
			if m.q.Kind == wizard.Multi {
				box := "[ ] "
				if m.ticked[o.Value] {
					box = "[x] "
				}
				mark += box
			}
			b.WriteString(mark + o.Label + "\n")
		}
	} else {
		for _, e := range m.entries {
			b.WriteString("  " + e + "\n")
		}
		b.WriteString(m.input.View() + "\n")
	}
	if m.errMsg != "" {
		b.WriteString(errLine.Render(m.errMsg) + "\n")
	}
	b.WriteString(dim.Render(m.keysHelp()) + "\n")
	return b.String()
}

// keysHelp is the help line for this kind of question.
func (m *ask) keysHelp() string {
	switch m.q.Kind {
	case wizard.Multi:
		return i18n.T("ask.keys_multi")
	case wizard.Choice, wizard.YesNo:
		return i18n.T("ask.keys_choice")
	case wizard.List:
		return i18n.T("ask.keys_list")
	}
	return i18n.T("ask.keys_text")
}

// shown is the answer as it stays on screen.
func (m *ask) shown() string {
	if m.choosing() {
		var labels []string
		for _, o := range m.q.Options {
			for _, v := range m.values {
				if o.Value == v {
					labels = append(labels, o.Label)
				}
			}
		}
		return strings.Join(labels, ", ")
	}
	if len(m.values) == 1 && m.values[0] == "" && len(m.q.Default) > 0 {
		return m.q.Default[0]
	}
	return strings.Join(m.values, ", ")
}
