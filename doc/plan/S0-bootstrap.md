# S0 — Bootstrap

**Goal**: an empty but fully tooled module where lint, mockery and tests already run, locally and in CI.

**Depends on**: nothing.

## Steps

1. `git init -b master` (the directory isn't a repo yet). Commit the existing `CONTEXT.md` and `doc/plan/` on `master` as `docs: add context and implementation plan`.
2. Create the GitHub repo and push `master`. Confirm with the user before running this, because it publishes the repo:
   ```bash
   gh repo create bjet007/brewupdater --private --source . --push
   ```
   The repo stays private until it's ready to share. It must become public before the first release (S7).
3. Branch `S0_bootstrap` for the rest of this session.
4. `go mod init github.com/bjet007/brewupdater` and set `go 1.26`.
5. Tool dependencies, added as `tool` directives in `go.mod`:
   ```bash
   go get -tool github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
   go get -tool github.com/vektra/mockery/v3@latest
   ```
6. `.golangci.yml`: copy bromote's file (`tools/bromote/.golangci.yml`), then:
   - add `errorlint`, `exhaustive`, `contextcheck`, `testifylint`, `prealloc`, `makezero`, `asciicheck`, with `testifylint: enable-all: true`
   - replace `gl.nsesi.io` with `github.com/bjet007` in `goimports.local-prefixes` and the `gci` sections
   - drop bromote-specific exclusions (`internal/cli/reporter/` goprintffuncname, `/config.go$` split_words)
7. `.mockery.yml`: bromote's header (`all: false`, `dir: "{{.InterfaceDir}}/mocks"`, `pkgname: mocks`, `filename`, `structname`) with an empty `packages:` map. S4 fills it in.
8. `cmd/brewupdater/main.go`: a stub that prints nothing and exits 0. It becomes a call to `internal/cmd/brewupdater.Execute()` in S6.
9. `CLAUDE.md`, following the `team-docs/guideline/claude-code.md` minimum template, adapted:
   - Session startup: read `CONTEXT.md`, then `CONTEXT.local.md` if present, then `doc/plan/README.md`.
   - Conventions: point to the *Conventions* section of `doc/plan/README.md`.
   - No odteam plugin or team-docs ADR sections.
10. `.gitignore`: `CLAUDE.local.md`, `CONTEXT.local.md`, `dist/`, `/brewupdater`.
11. `.github/workflows/ci.yml` runs on push and pull request:
   - `actions/checkout`, then `actions/setup-go` with `go-version-file: go.mod`
   - `go tool golangci-lint run ./...`
   - `go test -race ./...`
12. Push `S0_bootstrap` and open a PR to `master` (`gh pr create`). The new `ci.yml` runs on this first PR.
13. Make CI a merge gate: protect `master` and require the CI job to pass:
   ```bash
   gh api -X PUT repos/bjet007/brewupdater/branches/master/protection \
     -F 'required_status_checks[strict]=true' -f 'required_status_checks[contexts][]=<ci job name>' \
     -F enforce_admins=false -F required_pull_request_reviews=null -F restrictions=null
   ```
   Use the job name from `ci.yml`. Confirm with the user first.
   Branch protection on a **private** repo needs GitHub Pro. On a free plan, skip this step and treat a green CI run as a manual merge gate until the repo goes public. S7 turns protection on.

## Tests

Nothing to test-drive yet. Add one trivial test only if `golangci-lint` or CI needs a package with a test file to run. Delete it in S1.

## Done when

- `go tool golangci-lint fmt ./...`, `go tool golangci-lint run ./...` and `go test ./...` all pass. CI only runs `run`, which in v2 also reports formatter diffs.
- `go tool mockery` is installed. With an empty `packages:` map it exits with "no packages specified in config", which is expected; the first real run happens in S4.
- `go build ./cmd/brewupdater` succeeds.
- The CI run on the S0 PR is green, `/current-mr-check` has been addressed, and the PR is merged into `master`.
