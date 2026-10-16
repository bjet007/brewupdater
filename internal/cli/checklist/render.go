package checklist

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/bjet007/brewupdater/internal/plan"
)

const (
	title    = "Select the packages to upgrade"
	helpText = "space toggle · a all · ↑/↓ move · enter upgrade · esc cancel"

	cursorMark    = ">"
	checkboxOn    = "[x]"
	checkboxOff   = "[ ]"
	checkboxInert = "·"
	favoriteMark  = "★"
	pinnedNote    = "(pinned)"
	arrow         = "→"
)

var (
	styleTitle    = lipgloss.NewStyle().Bold(true)
	styleCursor   = lipgloss.NewStyle().Bold(true)
	styleTicked   = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	styleFavorite = lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
	styleDim      = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Faint(true)
	styleMuted    = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
)

// widths holds each column's width, so every row lines up.
type widths struct {
	name, kind, installed, latest int
}

func measure(rows []plan.Row) widths {
	var w widths
	for _, r := range rows {
		w.name = max(w.name, lipgloss.Width(r.Name))
		w.kind = max(w.kind, lipgloss.Width(kindText(r.Kind)))
		w.installed = max(w.installed, lipgloss.Width(r.Installed))
		w.latest = max(w.latest, lipgloss.Width(r.Latest))
	}
	return w
}

func (m *model) render() string {
	w := measure(m.rows)

	end := m.offset + m.windowSize()
	lines := []string{styleTitle.Render(title), ""}
	for i := m.offset; i < end; i++ {
		if i == m.favorites && m.favorites > 0 {
			lines = append(lines, m.separator(w))
		}
		lines = append(lines, m.renderRow(i, m.rows[i], w))
	}

	help := helpText
	if end-m.offset < len(m.rows) {
		help = fmt.Sprintf("%d-%d of %d · ", m.offset+1, end, len(m.rows)) + help
	}
	lines = append(lines, "", styleMuted.Render(help))
	return strings.Join(lines, "\n")
}

func (m *model) renderRow(i int, r plan.Row, w widths) string {
	cursor := " "
	if i == m.cursor {
		cursor = styleCursor.Render(cursorMark)
	}

	cells := []string{cursor, m.checkbox(i, r)}
	if m.favorites > 0 {
		star := " "
		if r.Favorite {
			star = styleFavorite.Render(favoriteMark)
		}
		cells = append(cells, star)
	}

	text := columnsText(r, w)
	if r.Pinned {
		text = styleDim.Render(text + " " + pinnedNote)
	}
	cells = append(cells, text)

	return strings.TrimRight(strings.Join(cells, " "), " ")
}

func (m *model) checkbox(i int, r plan.Row) string {
	switch {
	case !r.Selectable():
		return styleDim.Render(pad(checkboxInert, lipgloss.Width(checkboxOff)))
	case m.ticked[i]:
		return styleTicked.Render(checkboxOn)
	default:
		return checkboxOff
	}
}

func columnsText(r plan.Row, w widths) string {
	return pad(r.Name, w.name) + "  " + pad(kindText(r.Kind), w.kind) + "  " + pad(r.Installed, w.installed) + " " + arrow + " " +
		pad(r.Latest, w.latest)
}

// separator runs from the checkbox to the end of the latest column. It is only drawn when there are favorites,
// so the star column is always there.
func (m *model) separator(w widths) string {
	width := lipgloss.Width(checkboxOff) + 1 + lipgloss.Width(favoriteMark) + 1 + lipgloss.Width(columnsText(m.rows[0], w))
	indent := strings.Repeat(" ", lipgloss.Width(cursorMark)+1)
	return indent + styleMuted.Render(strings.Repeat("─", width))
}

func kindText(k plan.Kind) string {
	switch k {
	case plan.KindFormula:
		return "[formula]"
	case plan.KindCask:
		return "[cask]"
	}
	return ""
}

func pad(s string, width int) string {
	return s + strings.Repeat(" ", max(width-lipgloss.Width(s), 0))
}
