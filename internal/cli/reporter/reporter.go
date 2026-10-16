// Package reporter shows the refresh progress and the final summary.
package reporter

import (
	"context"
	"fmt"
	"io"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/term"

	"github.com/bjet007/brewupdater/internal/plan"
	"github.com/bjet007/brewupdater/internal/upgrade"
)

const (
	refreshingText = "Refreshing taps…"
	nothingText    = "Nothing upgraded"
	okGlyph        = "✔"
	failedGlyph    = "✘"
)

var (
	styleOK     = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	styleFailed = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
)

// Reporter writes progress and the summary to out. Colours and the spinner are only used when out is a terminal.
type Reporter struct {
	out io.Writer
}

// New returns a Reporter writing to out.
func New(out io.Writer) Reporter {
	return Reporter{out: out}
}

// Refreshing runs fn behind a spinner, or after a plain line when out is not a terminal.
func (r Reporter) Refreshing(ctx context.Context, fn func(context.Context) error) error {
	if !isTerminal(r.out) {
		_, _ = fmt.Fprintln(r.out, refreshingText)
		return fn(ctx)
	}
	return r.spin(ctx, fn)
}

// Summary prints which selected packages were upgraded and which are still outdated.
func (r Reporter) Summary(res upgrade.Result) {
	styled := isTerminal(r.out)

	var parts []string
	if len(res.Upgraded) > 0 {
		parts = append(parts, glyph(styleOK, okGlyph, styled)+" "+names(res.Upgraded))
	}
	if len(res.Failed) > 0 {
		parts = append(parts, glyph(styleFailed, failedGlyph, styled)+" "+names(res.Failed))
	}
	if len(parts) == 0 {
		parts = append(parts, nothingText)
	}
	_, _ = fmt.Fprintln(r.out, strings.Join(parts, "  "))
}

func glyph(style lipgloss.Style, g string, styled bool) string {
	if !styled {
		return g
	}
	return style.Render(g)
}

func names(pkgs []plan.Package) string {
	n := make([]string, len(pkgs))
	for i, p := range pkgs {
		n[i] = p.Name
	}
	return strings.Join(n, ", ")
}

func isTerminal(out io.Writer) bool {
	f, ok := out.(interface{ Fd() uintptr })
	return ok && term.IsTerminal(f.Fd())
}
