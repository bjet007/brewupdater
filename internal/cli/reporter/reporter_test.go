package reporter_test

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bjet007/brewupdater/internal/cli/reporter"
	"github.com/bjet007/brewupdater/internal/plan"
	"github.com/bjet007/brewupdater/internal/upgrade"
)

var _ upgrade.Reporter = reporter.Reporter{}

var errRefresh = errors.New("brew update failed")

var (
	node    = plan.Package{Name: "node", Kind: plan.KindFormula}
	kubectl = plan.Package{Name: "kubectl", Kind: plan.KindFormula}
	jq      = plan.Package{Name: "jq", Kind: plan.KindFormula}
)

func TestSummary(t *testing.T) {
	tests := map[string]struct {
		result upgrade.Result
		want   string
	}{
		"all ok": {
			result: upgrade.Result{Upgraded: []plan.Package{node, kubectl}},
			want:   "✔ node, kubectl\n",
		},
		"with failures": {
			result: upgrade.Result{Upgraded: []plan.Package{node}, Failed: []plan.Package{jq}},
			want:   "✔ node  ✘ jq\n",
		},
		"all failed": {
			result: upgrade.Result{Failed: []plan.Package{node, jq}},
			want:   "✘ node, jq\n",
		},
		"nothing selected": {
			result: upgrade.Result{},
			want:   "Nothing upgraded\n",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var out bytes.Buffer

			reporter.New(&out).Summary(tc.result)

			assert.Equal(t, tc.want, out.String())
		})
	}
}

func TestRefreshing(t *testing.T) {
	tests := map[string]struct {
		fnErr   error
		wantErr error
	}{
		"fn succeeds": {},
		"fn fails":    {fnErr: errRefresh, wantErr: errRefresh},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var out bytes.Buffer
			called := false

			err := reporter.New(&out).Refreshing(context.Background(), func(context.Context) error {
				called = true
				return tc.fnErr
			})

			require.ErrorIs(t, err, tc.wantErr)
			assert.True(t, called)
			assert.Equal(t, "Refreshing taps…\n", out.String())
		})
	}
}

func TestSpin(t *testing.T) {
	tests := map[string]struct {
		fnErr   error
		wantErr error
	}{
		"fn succeeds": {},
		"fn fails":    {fnErr: errRefresh, wantErr: errRefresh},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var out bytes.Buffer
			called := false

			err := reporter.SpinWith(context.Background(), &out, func(context.Context) error {
				called = true
				return tc.fnErr
			}, tea.WithInput(nil), tea.WithoutRenderer(), tea.WithoutSignals())

			require.ErrorIs(t, err, tc.wantErr)
			assert.True(t, called)
		})
	}
}

func TestSpinInterruptCancelsFn(t *testing.T) {
	tests := map[string]struct {
		opts []tea.ProgramOption
	}{
		"interrupt signal": {
			opts: []tea.ProgramOption{
				tea.WithInput(nil),
				tea.WithFilter(func(tea.Model, tea.Msg) tea.Msg { return tea.InterruptMsg{} }),
			},
		},
		"ctrl+c key": {
			opts: []tea.ProgramOption{tea.WithInput(strings.NewReader("\x03"))},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var out bytes.Buffer
			opts := append([]tea.ProgramOption{tea.WithoutRenderer(), tea.WithoutSignals()}, tc.opts...)

			err := reporter.SpinWith(context.Background(), &out, func(ctx context.Context) error {
				<-ctx.Done()
				return ctx.Err()
			}, opts...)

			require.ErrorIs(t, err, context.Canceled)
		})
	}
}

func TestSpinCancelsFnWhenContextIsCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var out bytes.Buffer

	err := reporter.SpinWith(ctx, &out, func(ctx context.Context) error {
		cancel()
		<-ctx.Done()
		return ctx.Err()
	}, tea.WithInput(nil), tea.WithoutRenderer(), tea.WithoutSignals())

	require.ErrorIs(t, err, context.Canceled)
}
