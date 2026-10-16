package reporter

import (
	"context"
	"io"

	tea "charm.land/bubbletea/v2"
)

// SpinWith runs the spinner path with the program options a headless test needs, since a test has no terminal.
func SpinWith(ctx context.Context, out io.Writer, fn func(context.Context) error, opts ...tea.ProgramOption) error {
	return New(out).spin(ctx, fn, opts...)
}
