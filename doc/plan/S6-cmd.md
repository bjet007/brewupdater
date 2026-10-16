# S6 — `internal/cmd/brewupdater` (cobra + fang)

**Goal**: wire everything behind one root command with the agreed flags.

**Depends on**: S2, S4, S5.

**Reference**: bromote `internal/cmd/bromote/start.go` (`fang.Execute(ctx, root, fang.WithVersion(version))`).

## Shape

```go
package brewupdater

var version = "" // set by GoReleaser ldflags; falls back to debug.ReadBuildInfo()

func Execute() int // exit code; cmd/brewupdater/main.go does os.Exit(Execute())
```

- Flags: `--config`, `--no-update`, `--greedy`, `--list`, `--json`, `-y/--yes`. Mark `--list`, `--json` and `--yes` with `MarkFlagsMutuallyExclusive`.
- Merge rules: start from the config file values, then apply flags on top. `--no-update` forces AutoUpdate off and `--greedy` forces Greedy on. Flags only ever override in one direction.
- Before starting the TUI, check that stdin and stdout are a TTY. Without one and without `--list`/`--json`/`-y`, return an error suggesting `--list` or `-y`.
- `--list` and `--json` call `Service.Plan`, write to stdout, and report `UnknownFavorites` as warnings on stderr. The JSON array contains `{name, kind, installed, latest, pinned, favorite}`.
- Exit codes: 0 on success or abort (print "Cancelled"), 1 if any upgrade failed or any other error occurred.
- Version: if `version` is empty, use `debug.ReadBuildInfo().Main.Version`, then fall back to `"dev"`.

## Test cases

Keep the logic in small pure functions so they can be table-tested without cobra:

| Function | Cases |
|---|---|
| `options(cfg, flags)` | config only; `--no-update` overrides `auto_update: true`; `--greedy` overrides `greedy: false`; favorites come only from config |
| `writeList(w, plan)` | ★ on favorites, aligned columns, `(pinned)` shown, empty plan prints "Everything is up to date" |
| `writeJSON(w, plan)` | the field set above; empty plan writes `[]` |
| `exitCode(result, err)` | nil → 0; `ErrAborted` → 0; failures → 1; other error → 1 |
| `resolveVersion(ldflag, buildInfo)` | ldflag wins; build info is next; then `"dev"` |
| root command | `--list --yes` is rejected as mutually exclusive (run `cmd.SetArgs` and `Execute` on the cobra command, without fang) |

## Done when

- All cases are green and lint is clean.
- Manual checks against real brew all work: `go run ./cmd/brewupdater --list`, `--json`, `--no-update`, the interactive run, and `-y`.
- `cmd/brewupdater/main.go` is only `os.Exit(brewupdater.Execute())`.
