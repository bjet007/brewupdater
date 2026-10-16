// Package checklist lets the user pick, from a plan, the packages to upgrade.
package checklist

import (
	"context"
	"errors"
	"fmt"

	tea "charm.land/bubbletea/v2"

	"github.com/bjet007/brewupdater/internal/plan"
)

// ErrAborted means the user cancelled the checklist. It is a decision, not a failure.
var ErrAborted = errors.New("checklist cancelled")

// Prompt runs the checklist in the terminal.
type Prompt struct{}

// Select shows the plan and returns the ticked packages in display order. An empty plan returns at once with nothing selected.
func (Prompt) Select(ctx context.Context, p plan.Plan) ([]plan.Package, error) {
	return run(ctx, p)
}

func run(ctx context.Context, p plan.Plan, opts ...tea.ProgramOption) ([]plan.Package, error) {
	if len(p.Rows()) == 0 {
		return nil, nil
	}

	m := newModel(p)
	opts = append([]tea.ProgramOption{tea.WithContext(ctx)}, opts...)
	if _, err := tea.NewProgram(m, opts...).Run(); err != nil {
		if errors.Is(err, tea.ErrInterrupted) {
			return nil, ErrAborted
		}
		return nil, fmt.Errorf("running checklist: %w", err)
	}
	if m.phase == phaseAborted {
		return nil, ErrAborted
	}
	return m.selection(), nil
}
