package checklist

import (
	"context"

	tea "charm.land/bubbletea/v2"

	"github.com/bjet007/brewupdater/internal/plan"
)

// RunWith is Prompt.Select with the program options a headless test needs to stand in for a terminal.
func RunWith(ctx context.Context, p plan.Plan, opts ...tea.ProgramOption) ([]plan.Package, error) {
	return run(ctx, p, opts...)
}

// NewModel exposes the model so tests drive its keys without a terminal.
func NewModel(p plan.Plan) tea.Model { return newModel(p) }

// Outcome reads a driven model's selection, and whether the user aborted.
func Outcome(driven tea.Model) ([]plan.Package, bool) {
	m, ok := driven.(*model)
	if !ok {
		return nil, false
	}
	return m.selection(), m.phase == phaseAborted
}

// Cursor is the index into plan.Rows() of the row under the cursor.
func Cursor(driven tea.Model) int {
	m, ok := driven.(*model)
	if !ok {
		return -1
	}
	return m.cursor
}
