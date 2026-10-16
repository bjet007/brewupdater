# S2 — `internal/config`

**Goal**: resolve the config path and load YAML into a struct with defaults.

**Depends on**: S0.

## Shape

```go
package config

type Config struct {
    Favorites  []string `yaml:"favorites"`
    Greedy     bool     `yaml:"greedy"`
    AutoUpdate bool     `yaml:"auto_update"`
}

func Default() Config // AutoUpdate: true, Greedy: false, no favorites

// Path returns the config file location: XDG_CONFIG_HOME if set, else ~/.config.
func Path() (string, error)

// Load reads path; a missing file returns Default() and no error.
func Load(path string) (Config, error)
```

Use `go.yaml.in/yaml/v3`, as bromote does. Decode with `KnownFields(true)` so a typo like `favourites:` fails loudly rather than being ignored.

`auto_update` defaults to `true`, so an absent key must not decode to `false`. Start from `Default()` and decode on top of it.

## Test cases

`TestPath` (use `t.Setenv`):

| Case | Expectation |
|---|---|
| `XDG_CONFIG_HOME` set | `$XDG_CONFIG_HOME/brewupdater/config.yaml` |
| `XDG_CONFIG_HOME` empty | `$HOME/.config/brewupdater/config.yaml` |

`TestLoad` (use `t.TempDir`):

| Case | Expectation |
|---|---|
| file missing | `Default()`, no error |
| empty file | `Default()` |
| favorites only | favorites set, `AutoUpdate` still true |
| `auto_update: false` | false |
| `greedy: true` | true |
| unknown key | error mentioning the key |
| invalid YAML | wrapped error containing the path |
| path is a directory | error |

## Done when

All cases are green and lint is clean.
