# brewupdater — implementation plan

A Go CLI/TUI that lists outdated Homebrew packages, puts the user's **Favorites** first (preselected), and **Upgrades** the chosen **Selection**. Terms are defined in [`CONTEXT.md`](../../CONTEXT.md).

The work is split into sessions. Each session is self-contained: start a fresh Claude Code session, read `CONTEXT.md`, this file, and the session file, then work only on that session.

## Session workflow

1. Start a **new Claude Code session**. Don't continue a previous one.
2. Branch from an up-to-date `master` as `<session>_<slug>`, e.g. `S1_plan-ordering`. S0 creates `master` itself.
3. Implement the session with the TDD loop below. Commit (Conventional Commits) only when the user asks, after showing the diff.
4. Review the branch against `master` with `/current-mr-check`. Fix the findings that matter and re-run the checks.
5. Push the branch and open a GitHub PR to `master` (`gh pr create`). CI (lint and test) must be green. It's a required check where branch protection is available; while the repo is private on a free plan, check it by hand before merging.
6. Merge the PR, pull `master` locally, and only then start the next session.

| # | File | Delivers |
|---|---|---|
| 0 | [S0-bootstrap.md](S0-bootstrap.md) | Module, tooling, lint/mock config, CLAUDE.md, CI lint+test |
| 1 | [S1-plan.md](S1-plan.md) | `internal/plan` — ordering, preselection, pinned, unknown favorites |
| 2 | [S2-config.md](S2-config.md) | `internal/config` — YAML, XDG path, defaults |
| 3 | [S3-brew.md](S3-brew.md) | `internal/brew` — exec wrapper, `--json=v2` parsing, update/upgrade |
| 4 | [S4-upgrade.md](S4-upgrade.md) | `internal/upgrade` — orchestration flow, consumer interfaces, mocks |
| 5 | [S5-cli.md](S5-cli.md) | `internal/cli` — Bubble Tea checklist, reporter, tuitest |
| 6 | [S6-cmd.md](S6-cmd.md) | `internal/cmd` — cobra + fang, flags, `--list`/`--json`/`-y` |
| 7 | [S7-release.md](S7-release.md) | GoReleaser, Homebrew cask tap, release workflow |

Sessions 1–3 only depend on S0 and can be done in any order. S4 needs S1 and S3. S5 needs S1. S6 needs S2, S4 and S5. S7 needs S6.

---

## Agreed behavior

- **Scope**: formulae and casks from `brew outdated --json=v2`. Self-updating casks (`auto_updates`) are hidden unless `--greedy` / `greedy: true`.
- **Upgrade** always goes to the latest version available in the tap. Choosing a target version is out of scope.
- **Ordering**: outdated **Favorites** first in config order, then all other outdated packages alphabetically, separated visually.
- **Favorites that are not outdated** are hidden. **Unknown favorites** (not installed / typo) are ignored silently in the TUI and reported as a warning by `--list`/`--json`.
- **Preselection**: every outdated, non-pinned **Favorite** starts ticked. Everything else starts unticked.
- **Pinned packages** are shown greyed out and cannot be selected.
- **Refresh**: `brew update` runs first (spinner) unless `--no-update` / `auto_update: false`.
- **Upgrade run**: leave the TUI, then run at most two batches with the terminal handed to brew (live output, sudo prompts work):
  `brew upgrade --formula <names…>` then `brew upgrade --cask [--greedy] <names…>`, both with `HOMEBREW_NO_AUTO_UPDATE=1`. The cask batch runs even if the formula batch failed.
- **Summary**: re-run `brew outdated`; a selected package still outdated is a failure. Print `✔ …  ✘ …`; exit non-zero if any failed.
- **Abort** (Esc/Ctrl-C in the checklist) is a decision, not an error: print a short message, exit 0.

## CLI

| Flag | Effect |
|---|---|
| *(none)* | refresh → checklist TUI → upgrade |
| `--config <path>` | override the config location |
| `--no-update` | skip `brew update` |
| `--greedy` | include self-updating casks |
| `--list` | print the ordered outdated list (★ = favorite) and exit |
| `--json` | same as `--list`, as JSON |
| `-y`, `--yes` | skip the TUI; upgrade the preselected set |

`--list`, `--json` and `--yes` are mutually exclusive.

## Config

`$XDG_CONFIG_HOME/brewupdater/config.yaml`, falling back to `~/.config/brewupdater/config.yaml`. A missing file is not an error.

```yaml
favorites:
  - node
  - kubectl
greedy: false       # default
auto_update: true   # default
```

## Layout

```
brewupdater/
├── cmd/brewupdater/main.go       → internal/cmd/brewupdater.Execute()
├── internal/
│   ├── cmd/brewupdater/          cobra root + fang, flag wiring, version
│   ├── brew/                     concrete *Client: exec.go / args.go / parse.go, testdata/
│   ├── config/                   YAML load, XDG path resolution
│   ├── plan/                     pure domain: Package, Row, Plan, Build()
│   ├── upgrade/                  the flow; declares consumer interfaces
│   │   └── mocks/                mockery-generated
│   └── cli/
│       ├── checklist/            Bubble Tea v2 model
│       ├── reporter/             spinner + summary
│       └── tuitest/              copied from bromote
├── doc/plan/                     this plan
├── .github/workflows/            ci.yml, release.yml
├── .golangci.yml
├── .mockery.yml
├── .goreleaser.yaml
├── CLAUDE.md
├── CONTEXT.md
└── go.mod                        module github.com/bjet007/brewupdater, go 1.26
```

**Reference implementation**: `/Users/bjean/Documents/workspace/ssa/orbit-determination/tools/bromote` — same stack (Charm v2 under `charm.land/...`, cobra + fang, mockery v3, testify, `go.yaml.in/yaml/v3`). Copy patterns from it; don't import it.

## Conventions

These follow `team-docs/guideline/go.md`, `code-comments.md` and `commit-and-mr.md` (at `/Users/bjean/Documents/workspace/ssa/orbit-determination/team-docs/guideline/`), adapted for a personal GitHub project.

### TDD loop (every session)

1. **Red** — add one case to the table-driven test; run `go test ./internal/<pkg>/...` and see it fail for the right reason.
2. **Green** — the smallest code change that makes it pass.
3. **Refactor** — clean up with all tests green.
4. Repeat until the session's case list is done.

The session ends when all three are green:

```bash
go tool golangci-lint fmt ./...
go tool golangci-lint run ./...
go test ./...
```

Never commit unless the user asks. Show the diff first.

### Go

- **Tests**: table-driven with a `map[string]struct{…}` keyed by test name, iterated with `t.Run`. Assertions with testify `assert`/`require` (`testifylint` enforced).
- **Interfaces** are declared in the package that **uses** them, 1–3 methods each. Producers (`brew`, `checklist`, `reporter`) expose concrete types. Mocks are generated with `go tool mockery`, never hand-written.
- **Errors**: wrap with `fmt.Errorf("doing x: %w", err)`. Sentinel errors only when a caller checks them with `errors.Is` (e.g. `checklist.ErrAborted`). No custom error types unless a caller needs structured data.
- **Logging**: `log/slog` only when logging is needed (user-facing output goes through `reporter`). Messages start with a capital letter; attribute keys use `snake_case`.
- **Comments**: none by default. Doc comments only when the name can't carry the meaning, written in plain language. No drive-by refactors outside the session's scope.

### Lint and mocks

- `.golangci.yml` uses the golangci-lint **v2** schema: bromote's file as the base, plus `go.md`'s linters (`errorlint`, `exhaustive`, `contextcheck`, `testifylint`, `prealloc`, `makezero`, `asciicheck`). Import prefix is `github.com/bjet007`.
- `.mockery.yml` uses the mockery **v3** schema like bromote: `dir: "{{.InterfaceDir}}/mocks"`, `pkgname: mocks`.

### Commits

Conventional Commits, `<type>(<scope>): <summary>`, with the package as scope (no Jira here):

```
feat(plan): order favorites first and preselect them
test(brew): add cask fixture with string installed_versions
ci: add lint and test workflow
```
