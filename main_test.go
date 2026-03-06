package main

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestQDoesNotQuitWhileFiltering(t *testing.T) {
	m := model{
		step:    stepTargets,
		targets: []targetItem{{Label: allTarget}, {Label: "resource.aws_instance.example"}},
	}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if cmd != nil {
		t.Fatal("q should not quit the program")
	}

	got := updated.(model)
	if got.filter != "q" {
		t.Fatalf("expected filter to contain q, got %q", got.filter)
	}
}

func TestEscQuits(t *testing.T) {
	m := model{
		step:    stepTargets,
		targets: []targetItem{{Label: allTarget}, {Label: "resource.aws_instance.example"}},
	}

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("esc should quit the program")
	}
}
