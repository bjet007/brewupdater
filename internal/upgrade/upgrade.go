// Package upgrade runs the whole flow: refresh, build the plan, let the user pick, upgrade, then check what is still outdated.
package upgrade

import (
	"context"
	"errors"
	"fmt"

	"github.com/bjet007/brewupdater/internal/plan"
)

// Refresher refreshes tap metadata.
type Refresher interface {
	Update(ctx context.Context) error
}

// Lister reports what is outdated and what is installed.
type Lister interface {
	Outdated(ctx context.Context, greedy bool) ([]plan.Package, error)
	Installed(ctx context.Context) ([]string, error)
}

// Upgrader upgrades a batch of packages of one kind.
type Upgrader interface {
	Upgrade(ctx context.Context, kind plan.Kind, names []string, greedy bool) error
}

// Selector lets the user pick the packages to upgrade from the plan. Packages that are not in the plan are ignored.
type Selector interface {
	Select(ctx context.Context, p plan.Plan) ([]plan.Package, error)
}

// Reporter shows progress and the final summary.
type Reporter interface {
	// Refreshing runs fn behind a spinner.
	Refreshing(ctx context.Context, fn func(context.Context) error) error
	Summary(r Result)
}

// Options drive one run.
type Options struct {
	Favorites  []string
	Greedy     bool
	AutoUpdate bool
	// Yes skips the Selector and upgrades the preselected packages.
	Yes bool
}

// Result tells which selected packages were upgraded and which are still outdated.
type Result struct {
	Upgraded []plan.Package
	Failed   []plan.Package
}

// OK is true when no selected package is still outdated.
func (r Result) OK() bool {
	return len(r.Failed) == 0
}

// Service runs the flow. Selector and Upgrader are only needed by Run.
type Service struct {
	Refresher Refresher
	Lister    Lister
	Upgrader  Upgrader
	Selector  Selector
	Reporter  Reporter
}

// Plan refreshes tap metadata when asked to, then builds the plan from what is outdated.
func (s Service) Plan(ctx context.Context, o Options) (plan.Plan, error) {
	if o.AutoUpdate {
		if err := s.Reporter.Refreshing(ctx, s.Refresher.Update); err != nil {
			return plan.Plan{}, fmt.Errorf("refreshing tap metadata: %w", err)
		}
	}

	outdated, err := s.Lister.Outdated(ctx, o.Greedy)
	if err != nil {
		return plan.Plan{}, fmt.Errorf("listing outdated packages: %w", err)
	}
	installed, err := s.Lister.Installed(ctx)
	if err != nil {
		return plan.Plan{}, fmt.Errorf("listing installed packages: %w", err)
	}
	return plan.Build(outdated, installed, o.Favorites), nil
}

// Run builds the plan, gets a selection, upgrades the formulae then the casks, and checks what is still outdated.
// The Selector's error is returned unwrapped, so the caller can tell an abort (checklist.ErrAborted) with errors.Is.
// A failed upgrade batch is not an error by itself: the packages it left outdated end up in Result.Failed.
// The re-check ignores cancellation, so an interrupted upgrade still ends with a summary of what is left.
// If the re-check itself fails, Run returns it joined with the batch errors and does not call Summary.
func (s Service) Run(ctx context.Context, o Options) (Result, error) {
	p, err := s.Plan(ctx, o)
	if err != nil {
		return Result{}, err
	}

	var picked []plan.Package
	switch {
	case o.Yes:
		picked = p.Preselected()
	case len(p.Rows()) > 0:
		picked, err = s.Selector.Select(ctx, p)
		if err != nil {
			return Result{}, err
		}
	}

	selection := inPlanOrder(p, picked)
	if len(selection) == 0 {
		s.Reporter.Summary(Result{})
		return Result{}, nil
	}

	upgradeErr := errors.Join(
		s.upgradeBatch(ctx, plan.KindFormula, selection, o.Greedy),
		s.upgradeBatch(ctx, plan.KindCask, selection, o.Greedy),
	)

	still, err := s.Lister.Outdated(context.WithoutCancel(ctx), o.Greedy)
	if err != nil {
		return Result{}, errors.Join(upgradeErr, fmt.Errorf("re-checking outdated packages: %w", err))
	}

	r := classify(selection, still)
	s.Reporter.Summary(r)
	return r, nil
}

func (s Service) upgradeBatch(ctx context.Context, kind plan.Kind, selection []plan.Package, greedy bool) error {
	var names []string
	for _, pkg := range selection {
		if pkg.Kind == kind {
			names = append(names, pkg.Name)
		}
	}
	if len(names) == 0 {
		return nil
	}
	return s.Upgrader.Upgrade(ctx, kind, names, greedy)
}

type key struct {
	name string
	kind plan.Kind
}

func keyOf(p plan.Package) key {
	return key{name: p.Name, kind: p.Kind}
}

func inPlanOrder(p plan.Plan, picked []plan.Package) []plan.Package {
	want := make(map[key]bool, len(picked))
	for _, pkg := range picked {
		want[keyOf(pkg)] = true
	}

	var selection []plan.Package
	for _, r := range p.Rows() {
		if want[keyOf(r.Package)] {
			selection = append(selection, r.Package)
		}
	}
	return selection
}

func classify(selection, still []plan.Package) Result {
	outdated := make(map[key]bool, len(still))
	for _, pkg := range still {
		outdated[keyOf(pkg)] = true
	}

	var r Result
	for _, pkg := range selection {
		if outdated[keyOf(pkg)] {
			r.Failed = append(r.Failed, pkg)
		} else {
			r.Upgraded = append(r.Upgraded, pkg)
		}
	}
	return r
}
