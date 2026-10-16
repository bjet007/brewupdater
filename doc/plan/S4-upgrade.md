# S4 — `internal/upgrade`

**Goal**: the end-to-end flow as one testable unit, depending only on interfaces it declares itself.

**Depends on**: S1, S3 (types only; tests use mocks).

## Interfaces (declared here, 1–3 methods each)

```go
package upgrade

type Refresher interface { Update(ctx context.Context) error }
type Lister interface    { Outdated(ctx context.Context, greedy bool) ([]plan.Package, error) }
type Upgrader interface  { Upgrade(ctx context.Context, kind plan.Kind, names []string, greedy bool) error }
type Selector interface  { Select(ctx context.Context, p plan.Plan) ([]plan.Package, error) }
type Reporter interface {
    Refreshing(ctx context.Context, fn func(context.Context) error) error // wraps fn in a spinner
    Summary(r Result)
}
```

Add `Installed()` to `Lister` only if S1 kept the `installed` parameter. `*brew.Client` must satisfy `Refresher`, `Lister` and `Upgrader` without any adapter.

Register every interface in `.mockery.yml` under `github.com/bjet007/brewupdater/internal/upgrade`, then run `go tool mockery`.

## Shape

```go
type Options struct {
    Favorites  []string
    Greedy     bool
    AutoUpdate bool
    Yes        bool // skip Selector, use plan.Preselected()
}

type Result struct {
    Upgraded []plan.Package
    Failed   []plan.Package
}

func (r Result) OK() bool

type Service struct { /* the interfaces */ }

func (s Service) Plan(ctx context.Context, o Options) (plan.Plan, error) // refresh + list + plan.Build; reused by --list/--json
func (s Service) Run(ctx context.Context, o Options) (Result, error)     // Plan + select + upgrade + re-check + Summary
```

The flow inside `Run`:

1. `Plan`: refresh if `AutoUpdate`, then `Outdated`, then `plan.Build`.
2. Select: `Selector.Select`, or `Preselected()` when `Yes`.
3. Upgrade the formulae, then the casks. Keep both errors; don't stop after the first.
4. Re-check with `Outdated`. Selected packages still outdated go in `Failed`, the rest in `Upgraded`. The re-check is the authority: batch errors are dropped when it succeeds (brew's output was already shown live). If it fails, `Run` returns it joined with the batch errors and skips `Summary`.
5. Call `Reporter.Summary`.

## Test cases (`TestServiceRun`, mock-driven)

| Case | Expectation |
|---|---|
| auto_update false | `Update` never called |
| refresh fails | error returned, nothing else called |
| list fails | wrapped error |
| nothing outdated | Selector not called, empty Result, Summary called |
| user selects nothing | no Upgrade calls |
| selector returns `checklist.ErrAborted` | passed through unchanged (`errors.Is`), no Upgrade |
| mixed selection | one formula call, then one cask call, names in plan order |
| only casks selected | no formula call |
| formula batch fails | cask batch still runs |
| re-check shows one still outdated | that one in Failed, `OK()` false |
| `Yes` | Selector not called, preselected favorites upgraded |
| `Yes`, no favorites outdated | no Upgrade calls |
| greedy | passed to `Outdated` (both calls) and to the cask `Upgrade` |

Also `TestServicePlan`: refresh plus list plus build, with no Selector or Upgrader interaction.

The `ErrAborted` sentinel belongs to `checklist` (S5). If S5 isn't done yet, declare `upgrade.ErrAborted` and have S5 return it, or wrap it. Choose one approach and note it in the code.

## Done when

All cases are green, the mocks are generated (not hand-written), and lint is clean.
