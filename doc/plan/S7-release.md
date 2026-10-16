# S7 — Release (GoReleaser + Homebrew cask tap)

**Goal**: pushing a `v*` tag publishes GitHub release binaries and updates `bjet007/homebrew-tap`, so you can run `brew install bjet007/tap/brewupdater`.

**Depends on**: S6.

## Prerequisites (manual, done by the user)

1. Make `bjet007/brewupdater` public (`gh repo edit bjet007/brewupdater --visibility public --accept-visibility-change-consequences`). `brew install` downloads release assets anonymously, so a private repo doesn't work.
2. Turn on branch protection for `master`, requiring `lint-test` (the S0 step-13 command).
3. Create the public GitHub repo `bjet007/homebrew-tap` (it can start empty).
4. Create a fine-grained PAT with **Contents: read/write** on `homebrew-tap` only.
5. Add it to the `brewupdater` repo secrets as `HOMEBREW_TAP_GITHUB_TOKEN`.

## `.goreleaser.yaml` (GoReleaser v2)

```yaml
version: 2

builds:
  - main: ./cmd/brewupdater
    env: [CGO_ENABLED=0]
    goos: [darwin, linux]
    goarch: [amd64, arm64]
    ldflags:
      - -s -w -X github.com/bjet007/brewupdater/internal/cmd/brewupdater.version={{.Version}}

archives:
  - formats: [tar.gz]

checksum:
  name_template: checksums.txt

homebrew_casks:
  - repository:
      owner: bjet007
      name: homebrew-tap
      token: "{{ .Env.HOMEBREW_TAP_GITHUB_TOKEN }}"
    homepage: https://github.com/bjet007/brewupdater
    description: Pick which outdated Homebrew packages to upgrade, favorites first
    hooks:
      post:
        install: |
          if OS.mac?
            system_command "/usr/bin/xattr", args: ["-dr", "com.apple.quarantine", "#{staged_path}/brewupdater"]
          end
```

`homebrew_casks` replaces the `brews` (formula) section that GoReleaser deprecated in v2.10. The `xattr` post-install hook avoids Gatekeeper quarantine on the unsigned binary, so no notarization is needed. Check the current GoReleaser docs for the exact keys before writing the file.

## `.github/workflows/release.yml`

- Runs on `push: tags: ['v*']`.
- `actions/checkout` with `fetch-depth: 0` (GoReleaser needs the git history), then `actions/setup-go` with `go-version-file: go.mod`.
- `goreleaser/goreleaser-action` with `args: release --clean`, plus env `GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}` and `HOMEBREW_TAP_GITHUB_TOKEN: ${{ secrets.HOMEBREW_TAP_GITHUB_TOKEN }}`.
- `permissions: contents: write`.

## Versioning

`v0.1.0` was tagged manually. Since then, `release.yml` also runs on every push to `master`: `go tool svu next --v0` computes the next version from the Conventional Commits since the last tag, and when it differs from `svu current` the job tags it and runs GoReleaser in the same job (a tag pushed with `GITHUB_TOKEN` doesn't trigger another workflow). Manual `v*` tags still release.

## Verification (instead of unit tests)

| Check | Expectation |
|---|---|
| `goreleaser check` | valid config |
| `goreleaser release --snapshot --clean` | 4 archives in `dist/`; `dist/*/brewupdater --version` prints a snapshot version |
| `dist/homebrew/Casks/brewupdater.rb` | contains the post-install `xattr` hook |
| first real tag | release published, cask committed to the tap |
| `brew install bjet007/tap/brewupdater && brewupdater --version` | runs without a Gatekeeper prompt |
| `brewupdater --list` after a second tag | lists `brewupdater` itself as outdated |
