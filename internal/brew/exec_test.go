package brew

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bjet007/brewupdater/internal/plan"
)

type fakeRun struct {
	stdout string
	stderr string
	err    error

	calls []*exec.Cmd
}

func (f *fakeRun) run(cmd *exec.Cmd) error {
	f.calls = append(f.calls, cmd)
	if f.stdout != "" {
		_, _ = io.WriteString(cmd.Stdout, f.stdout)
	}
	if f.stderr != "" {
		_, _ = io.WriteString(cmd.Stderr, f.stderr)
	}
	return f.err
}

func TestClient(t *testing.T) {
	exitErr := errors.New("exit status 1")

	tests := map[string]struct {
		fake  fakeRun
		call  func(ctx context.Context, c *Client) (any, error)
		check func(t *testing.T, got any, err error, calls []*exec.Cmd)
	}{
		"Update ok": {
			call: func(ctx context.Context, c *Client) (any, error) {
				return nil, c.Update(ctx)
			},
			check: func(t *testing.T, _ any, err error, calls []*exec.Cmd) {
				require.NoError(t, err)
				require.Len(t, calls, 1)
				assert.Equal(t, []string{"brew", "update"}, calls[0].Args)
			},
		},
		"Update fails": {
			fake: fakeRun{stderr: "Error: no network\n", err: exitErr},
			call: func(ctx context.Context, c *Client) (any, error) {
				return nil, c.Update(ctx)
			},
			check: func(t *testing.T, _ any, err error, _ []*exec.Cmd) {
				require.ErrorIs(t, err, exitErr)
				require.ErrorContains(t, err, "Error: no network")
			},
		},
		"Update fails without stderr": {
			fake: fakeRun{err: exitErr},
			call: func(ctx context.Context, c *Client) (any, error) {
				return nil, c.Update(ctx)
			},
			check: func(t *testing.T, _ any, err error, _ []*exec.Cmd) {
				require.ErrorIs(t, err, exitErr)
				assert.Equal(t, "brew update: exit status 1", err.Error())
			},
		},
		"Outdated ok": {
			fake: fakeRun{stdout: `{"formulae":[{"name":"jq","installed_versions":["1.7"],"current_version":"1.8","pinned":false}],"casks":[]}`},
			call: func(ctx context.Context, c *Client) (any, error) {
				return c.Outdated(ctx, true)
			},
			check: func(t *testing.T, got any, err error, calls []*exec.Cmd) {
				require.NoError(t, err)
				require.Len(t, calls, 1)
				assert.Equal(t, []string{"brew", "outdated", "--json=v2", "--greedy"}, calls[0].Args)
				assert.Equal(t, []plan.Package{
					{Name: "jq", Kind: plan.KindFormula, Installed: "1.7", Latest: "1.8"},
				}, got)
			},
		},
		"Outdated fails": {
			fake: fakeRun{stderr: "Error: boom\n", err: exitErr},
			call: func(ctx context.Context, c *Client) (any, error) {
				return c.Outdated(ctx, false)
			},
			check: func(t *testing.T, got any, err error, _ []*exec.Cmd) {
				require.ErrorIs(t, err, exitErr)
				require.ErrorContains(t, err, "Error: boom")
				assert.Nil(t, got)
			},
		},
		"Installed ok": {
			fake: fakeRun{stdout: "jq\nnode\n\nfirefox\n"},
			call: func(ctx context.Context, c *Client) (any, error) {
				return c.Installed(ctx)
			},
			check: func(t *testing.T, got any, err error, calls []*exec.Cmd) {
				require.NoError(t, err)
				require.Len(t, calls, 1)
				assert.Equal(t, []string{"brew", "list", "--full-name", "-1"}, calls[0].Args)
				assert.Equal(t, []string{"jq", "node", "firefox"}, got)
			},
		},
		"Installed fails": {
			fake: fakeRun{stderr: "Error: boom\n", err: exitErr},
			call: func(ctx context.Context, c *Client) (any, error) {
				return c.Installed(ctx)
			},
			check: func(t *testing.T, got any, err error, _ []*exec.Cmd) {
				require.ErrorIs(t, err, exitErr)
				require.ErrorContains(t, err, "Error: boom")
				assert.Nil(t, got)
			},
		},
		"Upgrade args": {
			call: func(ctx context.Context, c *Client) (any, error) {
				return nil, c.Upgrade(ctx, plan.KindCask, []string{"firefox"}, true)
			},
			check: func(t *testing.T, _ any, err error, calls []*exec.Cmd) {
				require.NoError(t, err)
				require.Len(t, calls, 1)
				assert.Equal(t, []string{"brew", "upgrade", "--cask", "--greedy", "firefox"}, calls[0].Args)
			},
		},
		"Upgrade env": {
			call: func(ctx context.Context, c *Client) (any, error) {
				return nil, c.Upgrade(ctx, plan.KindFormula, []string{"jq"}, false)
			},
			check: func(t *testing.T, _ any, err error, calls []*exec.Cmd) {
				require.NoError(t, err)
				require.Len(t, calls, 1)
				assert.Contains(t, calls[0].Env, "HOMEBREW_NO_AUTO_UPDATE=1")
				assert.Contains(t, calls[0].Env, "BREWUPDATER_TEST_MARKER=1")
			},
		},
		"Upgrade streams": {
			call: func(ctx context.Context, c *Client) (any, error) {
				return nil, c.Upgrade(ctx, plan.KindFormula, []string{"jq"}, false)
			},
			check: func(t *testing.T, _ any, err error, calls []*exec.Cmd) {
				require.NoError(t, err)
				require.Len(t, calls, 1)
				assert.Same(t, os.Stdin, calls[0].Stdin)
				assert.Same(t, os.Stdout, calls[0].Stdout)
				assert.Same(t, os.Stderr, calls[0].Stderr)
			},
		},
		"Upgrade interrupts instead of killing on cancel": {
			call: func(ctx context.Context, c *Client) (any, error) {
				return nil, c.Upgrade(ctx, plan.KindFormula, []string{"jq"}, false)
			},
			check: func(t *testing.T, _ any, err error, calls []*exec.Cmd) {
				require.NoError(t, err)
				require.Len(t, calls, 1)
				assert.NotNil(t, calls[0].Cancel)
				assert.Positive(t, calls[0].WaitDelay)
			},
		},
		"Upgrade empty names": {
			call: func(ctx context.Context, c *Client) (any, error) {
				return nil, c.Upgrade(ctx, plan.KindFormula, nil, false)
			},
			check: func(t *testing.T, _ any, err error, calls []*exec.Cmd) {
				require.NoError(t, err)
				assert.Empty(t, calls)
			},
		},
		"Upgrade non-zero exit": {
			fake: fakeRun{err: exitErr},
			call: func(ctx context.Context, c *Client) (any, error) {
				return nil, c.Upgrade(ctx, plan.KindFormula, []string{"jq"}, false)
			},
			check: func(t *testing.T, _ any, err error, _ []*exec.Cmd) {
				require.ErrorIs(t, err, exitErr)
				require.ErrorContains(t, err, "brew upgrade --formula jq")
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Setenv("BREWUPDATER_TEST_MARKER", "1")
			fake := tc.fake
			c := &Client{run: fake.run}

			got, err := tc.call(t.Context(), c)

			tc.check(t, got, err, fake.calls)
		})
	}
}
