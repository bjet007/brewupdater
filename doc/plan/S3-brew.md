# S3 — `internal/brew`

**Goal**: a concrete `*Client` that talks to the `brew` binary: refresh, list outdated, upgrade. No interface here, since consumers declare their own in S4.

**Depends on**: S0. S1 is needed for the `plan.Package` type. If S1 isn't done yet, create `plan.Package` first.

## Shape

Mirror bromote's `pkg/<tool>/{args,exec,parse}.go` split:

```go
package brew

type Client struct {
    run runFunc // swappable in tests
}

func New() *Client

func (c *Client) Update(ctx context.Context) error
func (c *Client) Outdated(ctx context.Context, greedy bool) ([]plan.Package, error)
func (c *Client) Installed(ctx context.Context) ([]string, error) // only if S1 kept the installed parameter
func (c *Client) Upgrade(ctx context.Context, kind plan.Kind, names []string, greedy bool) error
```

- **args.go**: pure builders, the bulk of the TDD.
- **parse.go**: pure `--json=v2` decoding into `[]plan.Package`.
- **exec.go**: `os/exec` glue.
  - `Update` and `Outdated` capture output. On failure, include stderr in the wrapped error.
  - `Upgrade` connects `Stdin`, `Stdout` and `Stderr` to the process's own, so brew output streams live and sudo prompts work.
  - `Upgrade` sets `HOMEBREW_NO_AUTO_UPDATE=1` on top of `os.Environ()`.
  - Upgrading an empty name list is a no-op and does not run brew.

## Test cases

`TestOutdatedArgs`:

| Case | Expectation |
|---|---|
| not greedy | `outdated --json=v2` |
| greedy | `outdated --json=v2 --greedy` |

`TestUpgradeArgs`:

| Case | Expectation |
|---|---|
| formulae | `upgrade --formula a b` |
| casks, not greedy | `upgrade --cask x` |
| casks, greedy | `upgrade --cask --greedy x` |
| formulae, greedy | `--greedy` not added (casks only) |

`TestParseOutdated`: fixtures in `testdata/`. Capture real output once with `brew outdated --json=v2 > testdata/real.json` and trim it.

| Case | Expectation |
|---|---|
| formulae only | Kind, Installed (last of `installed_versions`), Latest, Pinned mapped |
| casks only | `KindCask` |
| cask `installed_versions` as a string (older brew) | parsed |
| cask `installed_versions` as an array | parsed |
| multiple installed versions | Installed is the last one |
| empty `{"formulae":[],"casks":[]}` | empty slice, no error |
| malformed JSON | wrapped error |

`TestClient` uses a fake `runFunc` that records argv, env and attached streams:

| Case | Expectation |
|---|---|
| Update ok | ran `brew update` |
| Update fails | error contains stderr |
| Outdated ok | parsed result |
| Upgrade env | `HOMEBREW_NO_AUTO_UPDATE=1` present |
| Upgrade streams | stdin, stdout and stderr attached |
| Upgrade empty names | runFunc not called |
| Upgrade non-zero exit | wrapped error |

## Done when

- All cases are green and lint is clean.
- A manual check passes: run a scratch `go run` or a test with `//go:build manual` that calls `New().Outdated(ctx, false)` against your real brew, and confirm the output looks right.
