package brew

import "github.com/bjet007/brewupdater/internal/plan"

// UpdateArgs builds the argv that refreshes tap metadata.
func UpdateArgs() []string {
	return []string{"update"}
}

// OutdatedArgs builds the argv that lists outdated formulae and casks as JSON.
func OutdatedArgs(greedy bool) []string {
	args := []string{"outdated", "--json=v2"}
	if greedy {
		args = append(args, "--greedy")
	}
	return args
}

// InstalledArgs builds the argv that lists every installed formula and cask, one name per line.
// Tap packages get their full name, as in the outdated output.
func InstalledArgs() []string {
	return []string{"list", "--full-name", "-1"}
}

// UpgradeArgs builds the argv that upgrades names of one kind. greedy only applies to casks.
func UpgradeArgs(kind plan.Kind, names []string, greedy bool) []string {
	args := []string{"upgrade"}
	switch kind {
	case plan.KindFormula:
		args = append(args, "--formula")
	case plan.KindCask:
		args = append(args, "--cask")
		if greedy {
			args = append(args, "--greedy")
		}
	}
	return append(args, names...)
}
