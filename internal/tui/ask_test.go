package tui

import (
	"reflect"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/davidp57/process-monitor/internal/wizard"
)

func press(m *ask, keys ...tea.KeyMsg) *ask {
	for _, k := range keys {
		next, _ := m.Update(k)
		m = next.(*ask)
	}
	return m
}

var (
	enter = tea.KeyMsg{Type: tea.KeyEnter}
	down  = tea.KeyMsg{Type: tea.KeyDown}
	space = tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}}
	bs    = tea.KeyMsg{Type: tea.KeyBackspace}
	esc   = tea.KeyMsg{Type: tea.KeyEsc}
)

func typed(s string) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)} }

func TestMultiUntick(t *testing.T) {
	q := &wizard.Question{Kind: wizard.Multi, Options: []wizard.Option{{Value: "a", Label: "A"}, {Value: "b", Label: "B"}}, Default: []string{"a", "b"}}
	m := press(newAsk(q, ""), down, space, enter)
	if m.result != Answered || !reflect.DeepEqual(m.values, []string{"a"}) {
		t.Errorf("%v %v", m.result, m.values)
	}
	m = press(newAsk(q, ""), space, down, space, enter)
	if m.values == nil || len(m.values) != 0 {
		t.Errorf("tout décoché : %#v", m.values)
	}
}

func TestChoiceStartsOnDefault(t *testing.T) {
	q := &wizard.Question{Kind: wizard.YesNo, Options: []wizard.Option{{Value: "yes", Label: "oui"}, {Value: "no", Label: "non"}}, Default: []string{"no"}}
	if m := press(newAsk(q, ""), enter); !reflect.DeepEqual(m.values, []string{"no"}) {
		t.Errorf("%v", m.values)
	}
}

func TestListEntries(t *testing.T) {
	q := &wizard.Question{Kind: wizard.List}
	m := press(newAsk(q, ""), typed("DEV"), enter, typed("VR"), enter, bs, typed("STD"), enter, enter)
	if !reflect.DeepEqual(m.values, []string{"DEV", "STD"}) {
		t.Errorf("%v", m.values)
	}
}

// A question that ends without Enter, as when the window is closed, gives up:
// it must never count as an answer with the suggested value.
func TestEndedWithoutEnterGivesUp(t *testing.T) {
	q := &wizard.Question{Kind: wizard.YesNo, Options: []wizard.Option{{Value: "yes", Label: "oui"}, {Value: "no", Label: "non"}}, Default: []string{"no"}}
	m := press(newAsk(q, ""), down)
	next, _ := m.Update(tea.QuitMsg{})
	if a := next.(*ask); a.result != Quit || a.values != nil {
		t.Errorf("%v %v", a.result, a.values)
	}
}

func TestEscIsBack(t *testing.T) {
	m := press(newAsk(&wizard.Question{Kind: wizard.Text}, ""), typed("x"), esc)
	if m.result != Back {
		t.Errorf("%v", m.result)
	}
}
