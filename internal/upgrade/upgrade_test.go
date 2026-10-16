package upgrade_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/bjet007/brewupdater/internal/brew"
	"github.com/bjet007/brewupdater/internal/plan"
	"github.com/bjet007/brewupdater/internal/upgrade"
	"github.com/bjet007/brewupdater/internal/upgrade/mocks"
)

var (
	_ upgrade.Refresher = (*brew.Client)(nil)
	_ upgrade.Lister    = (*brew.Client)(nil)
	_ upgrade.Upgrader  = (*brew.Client)(nil)
)

var (
	errBrew    = errors.New("brew failed")
	errUpgrade = errors.New("upgrade failed")
	errAborted = errors.New("aborted")
)

var (
	node    = plan.Package{Name: "node", Kind: plan.KindFormula, Installed: "1.0", Latest: "2.0"}
	jq      = plan.Package{Name: "jq", Kind: plan.KindFormula, Installed: "1.0", Latest: "2.0"}
	firefox = plan.Package{Name: "firefox", Kind: plan.KindCask, Installed: "1.0", Latest: "2.0"}
	slack   = plan.Package{Name: "slack", Kind: plan.KindCask, Installed: "1.0", Latest: "2.0"}

	allOutdated = []plan.Package{slack, jq, node, firefox}
	installed   = []string{"firefox", "jq", "node", "slack"}

	// fullPlan is what plan.Build makes of allOutdated with node as the only favorite.
	fullPlan = plan.Plan{
		Favorites: []plan.Row{{Package: node, Favorite: true, Preselected: true}},
		Others:    []plan.Row{{Package: firefox}, {Package: jq}, {Package: slack}},
	}
)

type deps struct {
	refresher *mocks.Refresher
	lister    *mocks.Lister
	upgrader  *mocks.Upgrader
	selector  *mocks.Selector
	reporter  *mocks.Reporter
}

func newDeps(t *testing.T) deps {
	t.Helper()

	return deps{
		refresher: mocks.NewRefresher(t),
		lister:    mocks.NewLister(t),
		upgrader:  mocks.NewUpgrader(t),
		selector:  mocks.NewSelector(t),
		reporter:  mocks.NewReporter(t),
	}
}

func (d deps) service() upgrade.Service {
	return upgrade.Service{
		Refresher: d.refresher,
		Lister:    d.lister,
		Upgrader:  d.upgrader,
		Selector:  d.selector,
		Reporter:  d.reporter,
	}
}

func (d deps) refreshes(err error) {
	d.reporter.EXPECT().Refreshing(mock.Anything, mock.Anything).
		RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) }).Once()
	d.refresher.EXPECT().Update(mock.Anything).Return(err).Once()
}

func (d deps) lists(greedy bool, outdated []plan.Package) {
	d.lister.EXPECT().Outdated(mock.Anything, greedy).Return(outdated, nil).Once()
	d.lister.EXPECT().Installed(mock.Anything).Return(installed, nil).Once()
}

func (d deps) rechecks(greedy bool, outdated []plan.Package) {
	d.lister.EXPECT().Outdated(mock.Anything, greedy).Return(outdated, nil).Once()
}

func (d deps) upgrades(kind plan.Kind, names []string, greedy bool, err error) *mock.Call {
	return d.upgrader.EXPECT().Upgrade(mock.Anything, kind, names, greedy).Return(err).Once()
}

func (d deps) summarizes(r upgrade.Result) {
	d.reporter.EXPECT().Summary(r).Once()
}

func TestServiceRun(t *testing.T) {
	tests := map[string]struct {
		opts       upgrade.Options
		setup      func(d deps)
		want       upgrade.Result
		wantErrIs  []error
		wantErrMsg string
	}{
		"auto_update false": {
			opts: upgrade.Options{Favorites: []string{"node"}},
			setup: func(d deps) {
				d.lists(false, nil)
				d.summarizes(upgrade.Result{})
			},
		},
		"refresh fails": {
			opts: upgrade.Options{AutoUpdate: true},
			setup: func(d deps) {
				d.refreshes(errBrew)
			},
			wantErrIs:  []error{errBrew},
			wantErrMsg: "refreshing",
		},
		"list fails": {
			opts: upgrade.Options{AutoUpdate: true},
			setup: func(d deps) {
				d.refreshes(nil)
				d.lister.EXPECT().Outdated(mock.Anything, false).Return(nil, errBrew).Once()
			},
			wantErrIs:  []error{errBrew},
			wantErrMsg: "listing outdated packages",
		},
		"installed fails": {
			setup: func(d deps) {
				d.lister.EXPECT().Outdated(mock.Anything, false).Return(allOutdated, nil).Once()
				d.lister.EXPECT().Installed(mock.Anything).Return(nil, errBrew).Once()
			},
			wantErrIs:  []error{errBrew},
			wantErrMsg: "listing installed packages",
		},
		"nothing outdated": {
			opts: upgrade.Options{Favorites: []string{"node"}, AutoUpdate: true},
			setup: func(d deps) {
				d.refreshes(nil)
				d.lists(false, nil)
				d.summarizes(upgrade.Result{})
			},
		},
		"user selects nothing": {
			opts: upgrade.Options{Favorites: []string{"node"}},
			setup: func(d deps) {
				d.lists(false, allOutdated)
				d.selector.EXPECT().Select(mock.Anything, fullPlan).Return(nil, nil).Once()
				d.summarizes(upgrade.Result{})
			},
		},
		"selector aborts": {
			opts: upgrade.Options{Favorites: []string{"node"}},
			setup: func(d deps) {
				d.lists(false, allOutdated)
				d.selector.EXPECT().Select(mock.Anything, fullPlan).Return(nil, errAborted).Once()
			},
			wantErrIs: []error{errAborted},
		},
		"mixed selection": {
			opts: upgrade.Options{Favorites: []string{"node"}},
			setup: func(d deps) {
				d.lists(false, allOutdated)
				d.selector.EXPECT().Select(mock.Anything, fullPlan).Return([]plan.Package{slack, jq, node, firefox}, nil).Once()
				mock.InOrder(
					d.upgrades(plan.KindFormula, []string{"node", "jq"}, false, nil),
					d.upgrades(plan.KindCask, []string{"firefox", "slack"}, false, nil),
				)
				d.rechecks(false, nil)
				d.summarizes(upgrade.Result{Upgraded: []plan.Package{node, firefox, jq, slack}})
			},
			want: upgrade.Result{Upgraded: []plan.Package{node, firefox, jq, slack}},
		},
		"only casks selected": {
			setup: func(d deps) {
				d.lists(false, allOutdated)
				d.selector.EXPECT().Select(mock.Anything, mock.Anything).Return([]plan.Package{firefox}, nil).Once()
				d.upgrades(plan.KindCask, []string{"firefox"}, false, nil)
				d.rechecks(false, []plan.Package{node, jq, slack})
				d.summarizes(upgrade.Result{Upgraded: []plan.Package{firefox}})
			},
			want: upgrade.Result{Upgraded: []plan.Package{firefox}},
		},
		"formula batch fails": {
			opts: upgrade.Options{Favorites: []string{"node"}},
			setup: func(d deps) {
				d.lists(false, allOutdated)
				d.selector.EXPECT().Select(mock.Anything, fullPlan).Return([]plan.Package{node, firefox}, nil).Once()
				d.upgrades(plan.KindFormula, []string{"node"}, false, errUpgrade)
				d.upgrades(plan.KindCask, []string{"firefox"}, false, nil)
				d.rechecks(false, []plan.Package{node, jq, slack})
				d.summarizes(upgrade.Result{Upgraded: []plan.Package{firefox}, Failed: []plan.Package{node}})
			},
			want: upgrade.Result{Upgraded: []plan.Package{firefox}, Failed: []plan.Package{node}},
		},
		"cask batch fails": {
			opts: upgrade.Options{Favorites: []string{"node"}},
			setup: func(d deps) {
				d.lists(false, allOutdated)
				d.selector.EXPECT().Select(mock.Anything, fullPlan).Return([]plan.Package{node, firefox}, nil).Once()
				d.upgrades(plan.KindFormula, []string{"node"}, false, nil)
				d.upgrades(plan.KindCask, []string{"firefox"}, false, errUpgrade)
				d.rechecks(false, []plan.Package{firefox, jq, slack})
				d.summarizes(upgrade.Result{Upgraded: []plan.Package{node}, Failed: []plan.Package{firefox}})
			},
			want: upgrade.Result{Upgraded: []plan.Package{node}, Failed: []plan.Package{firefox}},
		},
		"re-check shows one still outdated": {
			opts: upgrade.Options{Favorites: []string{"node"}},
			setup: func(d deps) {
				d.lists(false, allOutdated)
				d.selector.EXPECT().Select(mock.Anything, fullPlan).Return([]plan.Package{node, jq}, nil).Once()
				d.upgrades(plan.KindFormula, []string{"node", "jq"}, false, nil)
				d.rechecks(false, []plan.Package{jq})
				d.summarizes(upgrade.Result{Upgraded: []plan.Package{node}, Failed: []plan.Package{jq}})
			},
			want: upgrade.Result{Upgraded: []plan.Package{node}, Failed: []plan.Package{jq}},
		},
		"re-check matches kind as well as name": {
			setup: func(d deps) {
				d.lists(false, []plan.Package{node})
				d.selector.EXPECT().Select(mock.Anything, mock.Anything).Return([]plan.Package{node}, nil).Once()
				d.upgrades(plan.KindFormula, []string{"node"}, false, nil)
				d.rechecks(false, []plan.Package{{Name: "node", Kind: plan.KindCask}})
				d.summarizes(upgrade.Result{Upgraded: []plan.Package{node}})
			},
			want: upgrade.Result{Upgraded: []plan.Package{node}},
		},
		"re-check fails": {
			setup: func(d deps) {
				d.lists(false, allOutdated)
				d.selector.EXPECT().Select(mock.Anything, mock.Anything).Return([]plan.Package{node}, nil).Once()
				d.upgrades(plan.KindFormula, []string{"node"}, false, errUpgrade)
				d.lister.EXPECT().Outdated(mock.Anything, false).Return(nil, errBrew).Once()
			},
			wantErrIs:  []error{errUpgrade, errBrew},
			wantErrMsg: "re-checking outdated packages",
		},
		"yes": {
			opts: upgrade.Options{Favorites: []string{"node"}, Yes: true},
			setup: func(d deps) {
				d.lists(false, allOutdated)
				d.upgrades(plan.KindFormula, []string{"node"}, false, nil)
				d.rechecks(false, []plan.Package{firefox, jq, slack})
				d.summarizes(upgrade.Result{Upgraded: []plan.Package{node}})
			},
			want: upgrade.Result{Upgraded: []plan.Package{node}},
		},
		"yes, no favorites outdated": {
			opts: upgrade.Options{Favorites: []string{"kubectl"}, Yes: true},
			setup: func(d deps) {
				d.lists(false, allOutdated)
				d.summarizes(upgrade.Result{})
			},
		},
		"greedy": {
			opts: upgrade.Options{Greedy: true},
			setup: func(d deps) {
				d.lists(true, allOutdated)
				d.selector.EXPECT().Select(mock.Anything, mock.Anything).Return([]plan.Package{jq, slack}, nil).Once()
				d.upgrades(plan.KindFormula, []string{"jq"}, true, nil)
				d.upgrades(plan.KindCask, []string{"slack"}, true, nil)
				d.rechecks(true, nil)
				d.summarizes(upgrade.Result{Upgraded: []plan.Package{jq, slack}})
			},
			want: upgrade.Result{Upgraded: []plan.Package{jq, slack}},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			d := newDeps(t)
			tc.setup(d)

			got, err := d.service().Run(t.Context(), tc.opts)

			if len(tc.wantErrIs) == 0 {
				require.NoError(t, err)
			}
			for _, want := range tc.wantErrIs {
				require.ErrorIs(t, err, want)
			}
			if tc.wantErrMsg != "" {
				require.ErrorContains(t, err, tc.wantErrMsg)
			}
			assert.Equal(t, tc.want, got)
			assert.Equal(t, tc.want.OK(), got.OK())
		})
	}
}

func TestServicePlan(t *testing.T) {
	tests := map[string]struct {
		opts      upgrade.Options
		setup     func(d deps)
		want      plan.Plan
		wantErrIs error
	}{
		"refresh, list and build": {
			opts: upgrade.Options{Favorites: []string{"node", "nodee"}, AutoUpdate: true},
			setup: func(d deps) {
				d.refreshes(nil)
				d.lists(false, allOutdated)
			},
			want: plan.Plan{
				Favorites:        fullPlan.Favorites,
				Others:           fullPlan.Others,
				UnknownFavorites: []string{"nodee"},
			},
		},
		"no refresh, greedy": {
			opts: upgrade.Options{Greedy: true},
			setup: func(d deps) {
				d.lists(true, []plan.Package{jq})
			},
			want: plan.Plan{Others: []plan.Row{{Package: jq}}},
		},
		"refresh fails": {
			opts: upgrade.Options{AutoUpdate: true},
			setup: func(d deps) {
				d.refreshes(errBrew)
			},
			wantErrIs: errBrew,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			d := newDeps(t)
			tc.setup(d)

			got, err := d.service().Plan(t.Context(), tc.opts)

			require.ErrorIs(t, err, tc.wantErrIs)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestResultOK(t *testing.T) {
	tests := map[string]struct {
		result upgrade.Result
		want   bool
	}{
		"empty":           {result: upgrade.Result{}, want: true},
		"all upgraded":    {result: upgrade.Result{Upgraded: []plan.Package{node}}, want: true},
		"one still there": {result: upgrade.Result{Upgraded: []plan.Package{node}, Failed: []plan.Package{jq}}, want: false},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.result.OK())
		})
	}
}
