package plan_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bjet007/brewupdater/internal/plan"
)

func formula(name string) plan.Package {
	return plan.Package{Name: name, Kind: plan.KindFormula, Installed: "1.0", Latest: "2.0"}
}

func cask(name string) plan.Package {
	return plan.Package{Name: name, Kind: plan.KindCask, Installed: "1.0", Latest: "2.0"}
}

func pinned(p plan.Package) plan.Package {
	p.Pinned = true
	return p
}

func favorite(p plan.Package) plan.Row {
	return plan.Row{Package: p, Favorite: true, Preselected: !p.Pinned}
}

func other(p plan.Package) plan.Row {
	return plan.Row{Package: p}
}

func TestBuild(t *testing.T) {
	tests := map[string]struct {
		outdated  []plan.Package
		installed []string
		favorites []string
		want      plan.Plan
	}{
		"no outdated, no favorites": {
			want: plan.Plan{},
		},
		"no favorites": {
			outdated:  []plan.Package{formula("node"), formula("jq"), cask("firefox")},
			installed: []string{"node", "jq", "firefox"},
			want: plan.Plan{
				Others: []plan.Row{other(cask("firefox")), other(formula("jq")), other(formula("node"))},
			},
		},
		"favorites in config order, not alphabetical": {
			outdated:  []plan.Package{formula("node"), formula("jq"), formula("kubectl")},
			installed: []string{"node", "jq", "kubectl"},
			favorites: []string{"node", "kubectl"},
			want: plan.Plan{
				Favorites: []plan.Row{favorite(formula("node")), favorite(formula("kubectl"))},
				Others:    []plan.Row{other(formula("jq"))},
			},
		},
		"favorite not outdated but installed": {
			outdated:  []plan.Package{formula("jq")},
			installed: []string{"jq", "node"},
			favorites: []string{"node"},
			want: plan.Plan{
				Others: []plan.Row{other(formula("jq"))},
			},
		},
		"favorite not installed": {
			outdated:  []plan.Package{formula("jq")},
			installed: []string{"jq"},
			favorites: []string{"nodee"},
			want: plan.Plan{
				Others:           []plan.Row{other(formula("jq"))},
				UnknownFavorites: []string{"nodee"},
			},
		},
		"outdated favorite is pinned": {
			outdated:  []plan.Package{pinned(formula("node"))},
			installed: []string{"node"},
			favorites: []string{"node"},
			want: plan.Plan{
				Favorites: []plan.Row{favorite(pinned(formula("node")))},
			},
		},
		"non-favorite pinned": {
			outdated:  []plan.Package{pinned(formula("node"))},
			installed: []string{"node"},
			want: plan.Plan{
				Others: []plan.Row{other(pinned(formula("node")))},
			},
		},
		"duplicate favorite in config": {
			outdated:  []plan.Package{formula("node"), formula("kubectl")},
			installed: []string{"node", "kubectl"},
			favorites: []string{"node", "kubectl", "node"},
			want: plan.Plan{
				Favorites: []plan.Row{favorite(formula("node")), favorite(formula("kubectl"))},
			},
		},
		"duplicate unknown favorite in config": {
			favorites: []string{"nodee", "nodee"},
			want: plan.Plan{
				UnknownFavorites: []string{"nodee"},
			},
		},
		"formula and cask with the same name, one favorite": {
			outdated:  []plan.Package{cask("docker"), formula("docker")},
			installed: []string{"docker"},
			favorites: []string{"docker"},
			want: plan.Plan{
				Favorites: []plan.Row{favorite(formula("docker")), favorite(cask("docker"))},
			},
		},
		"alphabetical sort is case-insensitive": {
			outdated:  []plan.Package{formula("jq"), formula("AwsCli"), formula("ack")},
			installed: []string{"jq", "AwsCli", "ack"},
			want: plan.Plan{
				Others: []plan.Row{other(formula("ack")), other(formula("AwsCli")), other(formula("jq"))},
			},
		},
		"same name in others sorts formula before cask": {
			outdated:  []plan.Package{cask("docker"), formula("docker")},
			installed: []string{"docker"},
			want: plan.Plan{
				Others: []plan.Row{other(formula("docker")), other(cask("docker"))},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := plan.Build(tc.outdated, tc.installed, tc.favorites)

			assert.Equal(t, tc.want, got)
		})
	}
}

func TestRowSelectable(t *testing.T) {
	tests := map[string]struct {
		row  plan.Row
		want bool
	}{
		"not pinned": {
			row:  other(formula("node")),
			want: true,
		},
		"pinned": {
			row:  other(pinned(formula("node"))),
			want: false,
		},
		"pinned favorite": {
			row:  favorite(pinned(formula("node"))),
			want: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.row.Selectable())
		})
	}
}

func TestPlanRows(t *testing.T) {
	tests := map[string]struct {
		plan plan.Plan
		want []plan.Row
	}{
		"empty": {
			plan: plan.Plan{},
			want: []plan.Row{},
		},
		"favorites then others": {
			plan: plan.Plan{
				Favorites: []plan.Row{favorite(formula("node")), favorite(formula("kubectl"))},
				Others:    []plan.Row{other(formula("ack")), other(formula("jq"))},
			},
			want: []plan.Row{
				favorite(formula("node")),
				favorite(formula("kubectl")),
				other(formula("ack")),
				other(formula("jq")),
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.plan.Rows())
		})
	}
}

func TestPlanPreselected(t *testing.T) {
	tests := map[string]struct {
		plan plan.Plan
		want []plan.Package
	}{
		"empty": {
			plan: plan.Plan{},
			want: []plan.Package{},
		},
		"only non-pinned favorites": {
			plan: plan.Plan{
				Favorites: []plan.Row{
					favorite(formula("node")),
					favorite(pinned(formula("kubectl"))),
					favorite(cask("firefox")),
				},
				Others: []plan.Row{other(formula("jq"))},
			},
			want: []plan.Package{formula("node"), cask("firefox")},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.plan.Preselected())
		})
	}
}
