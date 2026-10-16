# S1 — `internal/plan`

**Goal**: the pure function that turns "what brew says is outdated" plus "my favorites" into the ordered, preselected list the UI shows.

**Depends on**: S0.

## Shape (a starting point; let the tests drive the details)

```go
package plan

type Kind int

const (
    KindFormula Kind = iota
    KindCask
)

type Package struct {
    Name      string
    Kind      Kind
    Installed string // version currently installed
    Latest    string // version available in the tap
    Pinned    bool
}

type Row struct {
    Package
    Favorite    bool
    Preselected bool
}

func (r Row) Selectable() bool // false when Pinned

type Plan struct {
    Favorites        []Row    // outdated favorites, in config order
    Others           []Row    // remaining outdated packages, alphabetical
    UnknownFavorites []string // favorites not found among outdated or installed
}

func Build(outdated []Package, installed []string, favorites []string) Plan
func (p Plan) Rows() []Row      // Favorites then Others
func (p Plan) Preselected() []Package
```

`plan` owns these domain types. `brew` (S3) converts its JSON into `[]plan.Package`.

`installed` is the set of installed package names. It separates "favorite that's up to date" (hidden, no warning) from "favorite that isn't installed" (warning). If S3 shows that getting the installed list is expensive, drop the parameter and treat every non-outdated favorite as hidden with no warning. Record that decision in `CONTEXT.md`.

## Test cases (`TestBuild`, map-keyed)

| Case | Expectation |
|---|---|
| no outdated, no favorites | empty Favorites and Others |
| no favorites | all outdated in Others, alphabetical, none preselected |
| favorites in config order, not alphabetical | `kubectl, node` in config keeps that order |
| favorite not outdated but installed | not shown, not in UnknownFavorites |
| favorite not installed | in UnknownFavorites |
| outdated favorite is pinned | in Favorites, `Preselected=false`, `Selectable()=false` |
| non-favorite pinned | in Others, not selectable |
| duplicate favorite in config | appears once, at its first position |
| formula and cask with the same name, one favorite | both rows marked Favorite (match by name only) |
| alphabetical sort is case-insensitive | `AwsCli` sorts between `ack` and `jq` |

Plus `TestPlanRows` (concatenation order) and `TestPlanPreselected` (only favorites, never pinned).

## Done when

- All cases are green and lint is clean.
- Coverage of `internal/plan` is about 100% (pure code, so there's no excuse).
