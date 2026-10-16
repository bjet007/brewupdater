package brewupdater

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime/debug"
	"testing"

	"charm.land/fang/v2"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/bjet007/brewupdater/internal/cli/checklist"
	"github.com/bjet007/brewupdater/internal/cmd/brewupdater/mocks"
	"github.com/bjet007/brewupdater/internal/config"
	"github.com/bjet007/brewupdater/internal/plan"
	"github.com/bjet007/brewupdater/internal/upgrade"
)

var (
	node    = plan.Package{Name: "node", Kind: plan.KindFormula, Installed: "20.1.0", Latest: "22.0.0"}
	kubectl = plan.Package{Name: "kubectl", Kind: plan.KindFormula, Installed: "1.30", Latest: "1.31"}
	git     = plan.Package{Name: "git", Kind: plan.KindFormula, Installed: "2.44", Latest: "2.45", Pinned: true}
	iterm   = plan.Package{Name: "iterm2", Kind: plan.KindCask, Installed: "3.5", Latest: "3.6"}
)

func TestOptions(t *testing.T) {
	tests := map[string]struct {
		cfg  config.Config
		f    flags
		want upgrade.Options
	}{
		"config only": {
			cfg:  config.Config{Favorites: []string{"node"}, Greedy: true, AutoUpdate: true},
			want: upgrade.Options{Favorites: []string{"node"}, Greedy: true, AutoUpdate: true},
		},
		"--no-update overrides auto_update true": {
			cfg:  config.Config{AutoUpdate: true},
			f:    flags{noUpdate: true},
			want: upgrade.Options{},
		},
		"--greedy overrides greedy false": {
			cfg:  config.Config{AutoUpdate: true},
			f:    flags{greedy: true},
			want: upgrade.Options{Greedy: true, AutoUpdate: true},
		},
		"flags never turn auto_update on or greedy off": {
			cfg:  config.Config{Greedy: true},
			want: upgrade.Options{Greedy: true},
		},
		"favorites come only from config": {
			cfg:  config.Config{Favorites: []string{"node", "kubectl"}},
			f:    flags{yes: true, configPath: "/x.yaml"},
			want: upgrade.Options{Favorites: []string{"node", "kubectl"}, Yes: true},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, options(tc.cfg, tc.f))
		})
	}
}

func TestWriteList(t *testing.T) {
	tests := map[string]struct {
		plan plan.Plan
		want string
	}{
		"empty plan": {
			want: "Everything is up to date\n",
		},
		"favorites starred, columns aligned, pinned shown": {
			plan: plan.Build([]plan.Package{node, kubectl, git, iterm}, nil, []string{"node", "kubectl"}),
			want: "" +
				"★  node     formula  20.1.0  → 22.0.0\n" +
				"★  kubectl  formula  1.30    → 1.31\n" +
				"   git      formula  2.44    → 2.45 (pinned)\n" +
				"   iterm2   cask     3.5     → 3.6\n",
		},
		"no favorites drops the star column": {
			plan: plan.Build([]plan.Package{node, iterm}, nil, nil),
			want: "" +
				"iterm2  cask     3.5     → 3.6\n" +
				"node    formula  20.1.0  → 22.0.0\n",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			require.NoError(t, writeList(&buf, tc.plan))
			assert.Equal(t, tc.want, buf.String())
		})
	}
}

func TestWriteJSON(t *testing.T) {
	tests := map[string]struct {
		plan plan.Plan
		want string
	}{
		"empty plan": {
			want: `[]`,
		},
		"field set": {
			plan: plan.Build([]plan.Package{node, git, iterm}, nil, []string{"node"}),
			want: `[
				{"name":"node","kind":"formula","installed":"20.1.0","latest":"22.0.0","pinned":false,"favorite":true},
				{"name":"git","kind":"formula","installed":"2.44","latest":"2.45","pinned":true,"favorite":false},
				{"name":"iterm2","kind":"cask","installed":"3.5","latest":"3.6","pinned":false,"favorite":false}
			]`,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			require.NoError(t, writeJSON(&buf, tc.plan))
			assert.JSONEq(t, tc.want, buf.String())
		})
	}
}

func TestWriteWarnings(t *testing.T) {
	tests := map[string]struct {
		plan plan.Plan
		want string
	}{
		"no unknown favorites": {},
		"one line per unknown favorite": {
			plan: plan.Plan{UnknownFavorites: []string{"nodee", "kubctl"}},
			want: "Warning: favorite \"nodee\" is not installed\nWarning: favorite \"kubctl\" is not installed\n",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			writeWarnings(&buf, tc.plan)
			assert.Equal(t, tc.want, buf.String())
		})
	}
}

func TestExitCode(t *testing.T) {
	tests := map[string]struct {
		res  upgrade.Result
		err  error
		want int
	}{
		"success": {
			res: upgrade.Result{Upgraded: []plan.Package{node}},
		},
		"nothing selected": {},
		"aborted checklist": {
			err: checklist.ErrAborted,
		},
		"Ctrl-C during the refresh spinner": {
			err: fmt.Errorf("refreshing tap metadata: refresh interrupted: %w", context.Canceled),
		},
		"failed upgrade": {
			res:  upgrade.Result{Upgraded: []plan.Package{node}, Failed: []plan.Package{kubectl}},
			want: 1,
		},
		"other error": {
			err:  errors.New("listing outdated packages: boom"),
			want: 1,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, exitCode(tc.res, tc.err))
		})
	}
}

func TestResolveVersion(t *testing.T) {
	tests := map[string]struct {
		ldflag    string
		buildInfo *debug.BuildInfo
		want      string
	}{
		"ldflag wins": {
			ldflag:    "v1.2.3",
			buildInfo: &debug.BuildInfo{Main: debug.Module{Version: "v0.9.0"}},
			want:      "v1.2.3",
		},
		"build info is next": {
			buildInfo: &debug.BuildInfo{Main: debug.Module{Version: "v0.9.0"}},
			want:      "v0.9.0",
		},
		"devel build info falls back to dev": {
			buildInfo: &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}},
			want:      "dev",
		},
		"no build info falls back to dev": {
			want: "dev",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, resolveVersion(tc.ldflag, tc.buildInfo))
		})
	}
}

func TestRootCmd(t *testing.T) {
	tests := map[string]struct {
		args      []string
		want      flags
		wantErr   string
		wantNoRun bool
	}{
		"no flags": {},
		"every flag": {
			args: []string{"--config", "/c.yaml", "--no-update", "--greedy", "-y"},
			want: flags{configPath: "/c.yaml", noUpdate: true, greedy: true, yes: true},
		},
		"--list": {
			args: []string{"--list"},
			want: flags{list: true},
		},
		"--json": {
			args: []string{"--json"},
			want: flags{json: true},
		},
		"--list --yes is rejected": {
			args:      []string{"--list", "--yes"},
			wantErr:   "none of the others can be",
			wantNoRun: true,
		},
		"--list --json is rejected": {
			args:      []string{"--list", "--json"},
			wantErr:   "none of the others can be",
			wantNoRun: true,
		},
		"positional args are rejected": {
			args:      []string{"node"},
			wantErr:   "unknown command",
			wantNoRun: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var got *flags
			cmd := newRootCmd(func(_ *cobra.Command, f flags) error {
				got = &f
				return nil
			})
			cmd.SetArgs(tc.args)
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)

			err := cmd.Execute()
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
			} else {
				require.NoError(t, err)
			}
			if tc.wantNoRun {
				assert.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			assert.Equal(t, tc.want, *got)
		})
	}
}

func TestRun(t *testing.T) {
	errBrew := errors.New("brew failed")
	favoritePlan := plan.Build([]plan.Package{node}, nil, []string{"node", "nodee"})
	upgraded := upgrade.Result{Upgraded: []plan.Package{node}}

	type progress int
	const (
		toStdout progress = iota + 1
		toStderr
	)

	tests := map[string]struct {
		f            flags
		terminal     bool
		setup        func(r *mocks.Runner)
		wantProgress progress
		want         upgrade.Result
		wantErrIs    error
		wantStdout   string
		wantStderr   string
	}{
		"interactive without a terminal": {
			wantErrIs: errNoTerminal,
		},
		"interactive runs the flow": {
			terminal: true,
			setup: func(r *mocks.Runner) {
				r.EXPECT().Run(mock.Anything, upgrade.Options{AutoUpdate: true}).Return(upgraded, nil).Once()
			},
			wantProgress: toStdout,
			want:         upgraded,
		},
		"--yes runs without a terminal": {
			f: flags{yes: true},
			setup: func(r *mocks.Runner) {
				r.EXPECT().Run(mock.Anything, upgrade.Options{AutoUpdate: true, Yes: true}).Return(upgraded, nil).Once()
			},
			wantProgress: toStdout,
			want:         upgraded,
		},
		"--list writes the plan to stdout and warnings to stderr": {
			f: flags{list: true},
			setup: func(r *mocks.Runner) {
				r.EXPECT().Plan(mock.Anything, upgrade.Options{AutoUpdate: true}).Return(favoritePlan, nil).Once()
			},
			wantProgress: toStderr,
			wantStdout:   "★  node  formula  20.1.0  → 22.0.0\n",
			wantStderr:   "Warning: favorite \"nodee\" is not installed\n",
		},
		"--json writes JSON to stdout": {
			f: flags{json: true},
			setup: func(r *mocks.Runner) {
				r.EXPECT().Plan(mock.Anything, upgrade.Options{AutoUpdate: true}).Return(plan.Plan{}, nil).Once()
			},
			wantProgress: toStderr,
			wantStdout:   "[]\n",
		},
		"--list when the plan fails": {
			f: flags{list: true},
			setup: func(r *mocks.Runner) {
				r.EXPECT().Plan(mock.Anything, mock.Anything).Return(plan.Plan{}, errBrew).Once()
			},
			wantProgress: toStderr,
			wantErrIs:    errBrew,
		},
		"missing --config file": {
			f:         flags{list: true, configPath: filepath.Join(t.TempDir(), "missing.yaml")},
			wantErrIs: os.ErrNotExist,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Setenv("XDG_CONFIG_HOME", t.TempDir())
			var stdout, stderr bytes.Buffer
			r := mocks.NewRunner(t)
			if tc.setup != nil {
				tc.setup(r)
			}
			var gotProgress progress
			a := app{
				stdout:   &stdout,
				stderr:   &stderr,
				terminal: tc.terminal,
				newRunner: func(w io.Writer) Runner {
					gotProgress = map[io.Writer]progress{&stdout: toStdout, &stderr: toStderr}[w]
					return r
				},
			}

			got, err := a.run(t.Context(), tc.f)

			if tc.wantErrIs != nil {
				require.ErrorIs(t, err, tc.wantErrIs)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tc.want, got)
			assert.Equal(t, tc.wantProgress, gotProgress)
			assert.Equal(t, tc.wantStdout, stdout.String())
			assert.Equal(t, tc.wantStderr, stderr.String())
		})
	}
}

func TestLoadConfig(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(file, []byte("favorites: [node]\n"), 0o600))

	tests := map[string]struct {
		path      string
		want      config.Config
		wantErrIs error
	}{
		"default path missing gives defaults": {
			want: config.Default(),
		},
		"explicit path is read": {
			path: file,
			want: config.Config{Favorites: []string{"node"}, AutoUpdate: true},
		},
		"explicit path missing is an error": {
			path:      filepath.Join(dir, "missing.yaml"),
			wantErrIs: os.ErrNotExist,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Setenv("XDG_CONFIG_HOME", t.TempDir())

			got, err := loadConfig(tc.path)

			if tc.wantErrIs != nil {
				require.ErrorIs(t, err, tc.wantErrIs)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestHandleError(t *testing.T) {
	tests := map[string]struct {
		err  error
		want string
	}{
		"aborted checklist": {
			err:  checklist.ErrAborted,
			want: "Cancelled\n",
		},
		"wrapped context.Canceled": {
			err:  fmt.Errorf("refreshing tap metadata: %w", context.Canceled),
			want: "Cancelled\n",
		},
		"other error": {
			err:  errors.New("listing outdated packages: boom"),
			want: "listing outdated packages: boom",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			handleError(&buf, fang.Styles{}, tc.err)
			assert.Contains(t, buf.String(), tc.want)
			if tc.want != cancelledText+"\n" {
				assert.NotContains(t, buf.String(), cancelledText)
			}
		})
	}
}
