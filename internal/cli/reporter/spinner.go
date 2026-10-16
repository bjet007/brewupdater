package reporter

import (
	"context"
	"errors"
	"fmt"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
)

// spin runs fn in the background while a spinner animates, and returns fn's error once both are done.
// The program does not read input, so Ctrl-C reaches it as SIGINT, cancels fn and returns context.Canceled.
func (r Reporter) spin(ctx context.Context, fn func(context.Context) error, opts ...tea.ProgramOption) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var fnErr error
	done := make(chan struct{})
	go func() {
		defer close(done)
		fnErr = fn(ctx)
	}()

	opts = append([]tea.ProgramOption{tea.WithContext(ctx), tea.WithOutput(r.out), tea.WithInput(nil)}, opts...)
	_, err := tea.NewProgram(newSpinnerModel(done), opts...).Run()
	cancel()
	<-done
	if errors.Is(err, tea.ErrInterrupted) {
		return fmt.Errorf("refresh interrupted: %w", context.Canceled)
	}
	if err != nil {
		return fmt.Errorf("running spinner: %w", err)
	}
	return fnErr
}

type doneMsg struct{}

type spinnerModel struct {
	spinner  spinner.Model
	done     <-chan struct{}
	finished bool
}

func newSpinnerModel(done <-chan struct{}) *spinnerModel {
	return &spinnerModel{spinner: spinner.New(spinner.WithSpinner(spinner.Dot)), done: done}
}

func (m *spinnerModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, func() tea.Msg {
		<-m.done
		return doneMsg{}
	})
}

func (m *spinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case doneMsg:
		m.finished = true
		return m, tea.Quit
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

// View is empty once fn is done, so the spinner line does not stay on screen.
func (m *spinnerModel) View() tea.View {
	if m.finished {
		return tea.NewView("")
	}
	return tea.NewView(m.spinner.View() + " " + refreshingText)
}
