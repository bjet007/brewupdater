package checklist

import (
	tea "charm.land/bubbletea/v2"

	"github.com/bjet007/brewupdater/internal/plan"
)

type phase int

const (
	phaseSelecting phase = iota
	phaseDone
	phaseAborted
)

type model struct {
	rows []plan.Row
	// favorites is how many rows, from the top, are favorites; the separator is drawn after them.
	favorites int
	ticked    []bool
	cursor    int
	phase     phase
}

func newModel(p plan.Plan) *model {
	m := &model{
		rows:      p.Rows(),
		favorites: len(p.Favorites),
	}
	m.ticked = make([]bool, len(m.rows))
	for i, r := range m.rows {
		m.ticked[i] = r.Preselected
	}
	for i, r := range m.rows {
		if r.Selectable() {
			m.cursor = i
			break
		}
	}
	return m
}

func (m *model) Init() tea.Cmd { return nil }

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyPressMsg)
	if !ok || m.phase != phaseSelecting {
		return m, nil
	}

	switch key.String() {
	case "esc", "ctrl+c":
		m.phase = phaseAborted
		return m, tea.Quit
	case "enter":
		m.phase = phaseDone
		return m, tea.Quit
	case "up", "k":
		m.cursor = max(m.cursor-1, 0)
	case "down", "j":
		m.cursor = min(m.cursor+1, len(m.rows)-1)
	case "space":
		if m.rows[m.cursor].Selectable() {
			m.ticked[m.cursor] = !m.ticked[m.cursor]
		}
	case "a":
		m.toggleAll()
	}
	return m, nil
}

// toggleAll ticks every selectable row, or unticks them all when they already are.
func (m *model) toggleAll() {
	all := true
	for i, r := range m.rows {
		if r.Selectable() && !m.ticked[i] {
			all = false
			break
		}
	}
	for i, r := range m.rows {
		if r.Selectable() {
			m.ticked[i] = !all
		}
	}
}

func (m *model) View() tea.View {
	if m.phase != phaseSelecting {
		return tea.NewView("")
	}
	return tea.NewView(m.render())
}

func (m *model) selection() []plan.Package {
	var pkgs []plan.Package
	for i, r := range m.rows {
		if m.ticked[i] {
			pkgs = append(pkgs, r.Package)
		}
	}
	return pkgs
}

var _ tea.Model = (*model)(nil)
