// Package plan turns the outdated packages and the user's favorites into the ordered,
// preselected list the user picks from.
package plan

import (
	"cmp"
	"slices"
	"strings"
)

// Kind tells a formula from a cask.
type Kind int

// Kinds of package.
const (
	KindFormula Kind = iota
	KindCask
)

// Package is an outdated formula or cask, as brew reports it.
type Package struct {
	Name      string
	Kind      Kind
	Installed string
	Latest    string
	Pinned    bool
}

// Row is a Package as the user sees it in the list.
type Row struct {
	Package
	Favorite    bool
	Preselected bool
}

// Selectable is false for a pinned package, which brew refuses to upgrade.
func (r Row) Selectable() bool {
	return !r.Pinned
}

// Plan is the list the user picks from.
type Plan struct {
	Favorites []Row
	Others    []Row
	// UnknownFavorites are favorites that are neither outdated nor installed, usually a typo in the config.
	UnknownFavorites []string
}

// Build orders favorites first in config order, then other outdated packages alphabetically.
// A favorite matches every outdated package with its name, formula or cask.
func Build(outdated []Package, installed []string, favorites []string) Plan {
	sorted := slices.Clone(outdated)
	slices.SortStableFunc(sorted, comparePackages)

	var p Plan
	seen := make(map[string]bool, len(favorites))
	for _, name := range favorites {
		if seen[name] {
			continue
		}
		seen[name] = true

		matched := false
		for _, pkg := range sorted {
			if pkg.Name == name {
				p.Favorites = append(p.Favorites, Row{Package: pkg, Favorite: true, Preselected: !pkg.Pinned})
				matched = true
			}
		}
		if !matched && !slices.Contains(installed, name) {
			p.UnknownFavorites = append(p.UnknownFavorites, name)
		}
	}

	for _, pkg := range sorted {
		if !seen[pkg.Name] {
			p.Others = append(p.Others, Row{Package: pkg})
		}
	}

	return p
}

func comparePackages(a, b Package) int {
	return cmp.Or(
		cmp.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)),
		cmp.Compare(a.Name, b.Name),
		cmp.Compare(a.Kind, b.Kind),
	)
}

// Rows returns the favorites followed by the others.
func (p Plan) Rows() []Row {
	rows := make([]Row, 0, len(p.Favorites)+len(p.Others))
	rows = append(rows, p.Favorites...)
	return append(rows, p.Others...)
}

// Preselected returns the packages that start ticked.
func (p Plan) Preselected() []Package {
	pkgs := make([]Package, 0, len(p.Favorites))
	for _, r := range p.Favorites {
		if r.Preselected {
			pkgs = append(pkgs, r.Package)
		}
	}
	return pkgs
}
