package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// errMsg wraps an error returned from the background function.
type errMsg struct{ err error }

// doneMsg signals the background function completed successfully.
type doneMsg struct{}

// spinnerModel is the Bubble Tea model for the CLI spinner.
type spinnerModel struct {
	spinner  spinner.Model
	message  string
	fn       func() error
	done     bool
	err      error
	quitting bool
}

func newSpinnerModel(msg string, fn func() error) spinnerModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ColorBoba)
	return spinnerModel{
		spinner: s,
		message: msg,
		fn:      fn,
	}
}

func (m spinnerModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.runTask(),
	)
}

// runTask executes the background function and returns a done or error message.
func (m spinnerModel) runTask() tea.Cmd {
	fn := m.fn
	return func() tea.Msg {
		if err := fn(); err != nil {
			return errMsg{err: err}
		}
		return doneMsg{}
	}
}

func (m spinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			m.quitting = true
			return m, tea.Quit
		}
	case doneMsg:
		m.done = true
		m.quitting = true
		return m, tea.Quit
	case errMsg:
		m.err = msg.err
		m.quitting = true
		return m, tea.Quit
	default:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m spinnerModel) View() string {
	if m.quitting {
		if m.err != nil {
			return ErrorStyle.Render("✗") + "  " + m.message + " - " + ErrorStyle.Render("failed") + "\n"
		}
		return SuccessStyle.Render("✓") + "  " + m.message + " - " + SuccessStyle.Render("done") + "\n"
	}
	return m.spinner.View() + "  " + m.message + "\n"
}

// RunWithSpinner displays a spinner while fn executes. It shows a success
// or failure message when the function completes.
func RunWithSpinner(msg string, fn func() error) error {
	model := newSpinnerModel(msg, fn)
	p := tea.NewProgram(model, tea.WithInputTTY())
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("spinner program error: %w", err)
	}
	if m, ok := finalModel.(spinnerModel); ok && m.err != nil {
		return m.err
	}
	return nil
}
