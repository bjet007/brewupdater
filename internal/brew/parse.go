package brew

import (
	"encoding/json"
	"fmt"

	"github.com/bjet007/brewupdater/internal/plan"
)

type outdatedDocument struct {
	Formulae []outdatedEntry `json:"formulae"`
	Casks    []outdatedEntry `json:"casks"`
}

type outdatedEntry struct {
	Name              string            `json:"name"`
	InstalledVersions installedVersions `json:"installed_versions"`
	CurrentVersion    string            `json:"current_version"`
	Pinned            bool              `json:"pinned"`
}

// installedVersions accepts both shapes brew has used for casks: a single string (older brew) or an array.
type installedVersions []string

func (v *installedVersions) UnmarshalJSON(data []byte) error {
	var one string
	if err := json.Unmarshal(data, &one); err == nil {
		*v = installedVersions{one}
		return nil
	}

	var many []string
	if err := json.Unmarshal(data, &many); err != nil {
		return fmt.Errorf("decoding installed_versions: %w", err)
	}
	*v = many
	return nil
}

// ParseOutdated decodes the output of `brew outdated --json=v2`.
// Installed is the newest installed version, which brew lists last.
func ParseOutdated(data []byte) ([]plan.Package, error) {
	var doc outdatedDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parsing brew outdated json: %w", err)
	}

	pkgs := make([]plan.Package, 0, len(doc.Formulae)+len(doc.Casks))
	for _, e := range doc.Formulae {
		pkgs = append(pkgs, e.toPackage(plan.KindFormula))
	}
	for _, e := range doc.Casks {
		pkgs = append(pkgs, e.toPackage(plan.KindCask))
	}
	return pkgs, nil
}

func (e outdatedEntry) toPackage(kind plan.Kind) plan.Package {
	var installed string
	if n := len(e.InstalledVersions); n > 0 {
		installed = e.InstalledVersions[n-1]
	}
	return plan.Package{
		Name:      e.Name,
		Kind:      kind,
		Installed: installed,
		Latest:    e.CurrentVersion,
		Pinned:    e.Pinned,
	}
}
