# brewupdater

[![ci](https://github.com/bjet007/brewupdater/actions/workflows/ci.yml/badge.svg)](https://github.com/bjet007/brewupdater/actions/workflows/ci.yml)

Pick which outdated Homebrew packages to upgrade, favorites first.

`brewupdater` runs `brew update`, lists your outdated formulae and casks in a checklist with your favorites at the top and already ticked, then upgrades what you picked and tells you what failed.

## Install

```sh
brew install bjet007/tap/brewupdater
```

Or with Go 1.26+:

```sh
go install github.com/bjet007/brewupdater/cmd/brewupdater@latest
```

## Usage

```sh
brewupdater
```

1. Refreshes tap metadata with `brew update` (skip with `--no-update`).
2. Shows the outdated packages: favorites first in config order, then everything else alphabetically. Outdated favorites start ticked. Pinned packages (`brew pin`) are shown but can't be selected.
3. Upgrades the selection with `brew upgrade --formula …` then `brew upgrade --cask …`, with brew's own output and prompts (e.g. `sudo`).
4. Re-checks `brew outdated` and prints `✔`/`✘` per package. Exits non-zero if any upgrade failed.

| Key | Action |
|---|---|
| `↑`/`k`, `↓`/`j` | move |
| `space` | tick / untick |
| `a` | tick / untick all |
| `enter` | upgrade the ticked packages |
| `esc`, `ctrl+c` | quit without upgrading |

### Flags

| Flag | Effect |
|---|---|
| `--config <path>` | use another config file |
| `--no-update` | skip `brew update` |
| `--greedy` | include casks that update themselves (`auto_updates`) |
| `--list` | print the ordered outdated list (★ = favorite) and exit |
| `--json` | same as `--list`, as JSON |
| `-y`, `--yes` | skip the checklist and upgrade the outdated favorites |

`--list`, `--json` and `--yes` are mutually exclusive. `--list` and `--json` also warn about favorites that aren't installed.

## Config

`$XDG_CONFIG_HOME/brewupdater/config.yaml`, or `~/.config/brewupdater/config.yaml`. The file is optional.

```yaml
favorites:      # shown first, in this order, and preselected when outdated
  - node
  - kubectl
greedy: false       # default; same as --greedy
auto_update: true   # default; false is the same as --no-update
```

## Development

```sh
go tool golangci-lint fmt ./...
go tool golangci-lint run ./...
go test ./...
```

Releases are cut by pushing a `v*` tag; GoReleaser publishes the binaries and updates the cask in [bjet007/homebrew-tap](https://github.com/bjet007/homebrew-tap).

## License

[MIT](LICENSE)
