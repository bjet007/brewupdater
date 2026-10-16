# S5 — `internal/cli` (checklist, reporter, tuitest)

**Goal**: the terminal widgets. They know nothing about brew. They receive a `plan.Plan` or a `upgrade.Result` and render it.

**Depends on**: S1 (and S4 for `upgrade.Result` in the reporter).

**Reference**: bromote `internal/cli/{checklist,reporter,tuitest,prompt}`. Copy and trim it rather than redesigning. brewupdater needs no changelog pane, no per-row target and no `Loader`/`Opener`.

## `tuitest`

Copy bromote's `internal/cli/tuitest` and adjust the import paths. Keep its own tests.

## `checklist`

```go
var ErrAborted = errors.New("checklist cancelled")

type Prompt struct{}

func (Prompt) Select(ctx context.Context, p plan.Plan) ([]plan.Package, error) // satisfies upgrade.Selector
```

- Bubble Tea v2 model (`charm.land/bubbletea/v2`) with lipgloss v2 styles.
- Layout: `★` marks favorites, followed by the separator line, then the others. Columns: name, `[formula]`/`[cask]`, `installed → latest`. Pinned rows are dimmed and show `(pinned)`.
- Keys: ↑/↓ or k/j move, the cursor skips the separator, space toggles (a no-op on pinned rows), `a` toggles all selectable rows, enter submits, esc/ctrl-c aborts.
- An empty plan returns immediately with no selection and no program run.

Test cases, run on the model directly or through `tuitest`:

| Case | Expectation |
|---|---|
| initial state | preselected rows ticked, cursor on first selectable row |
| down past last favorite | cursor skips the separator |
| space on pinned | no change |
| space toggles | ticked ↔ unticked |
| `a` | ticks all selectable; pressed again, unticks all |
| enter | returns ticked packages in display order |
| esc | `ErrAborted` |
| render golden | aligned columns, ★, separator, dimmed pinned (strip ANSI before comparing) |
| no favorites | no separator rendered |

## `reporter`

```go
type Reporter struct{ out io.Writer }

func (r Reporter) Refreshing(ctx context.Context, fn func(context.Context) error) error // bubbles v2 spinner; plain text when out isn't a TTY
func (r Reporter) Summary(res upgrade.Result)
```

Test cases:

| Case | Expectation |
|---|---|
| summary all ok | `✔ node, kubectl` |
| summary with failures | `✔ node  ✘ jq` |
| summary nothing selected | "Nothing upgraded" |
| non-TTY refresh | prints a plain "Refreshing taps…" line, runs fn, returns its error |

## Done when

- All cases are green and lint is clean.
- `*checklist.Prompt` and `Reporter` satisfy the `upgrade` interfaces. Add `var _ upgrade.Selector = checklist.Prompt{}` in the wiring code (S6), not here, so `cli` doesn't import `upgrade` just for the assertion.
