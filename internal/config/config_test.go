package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bjet007/brewupdater/internal/config"
)

func TestPath(t *testing.T) {
	tests := map[string]struct {
		xdgConfigHome string
		home          string
		want          string
		wantErr       bool
	}{
		"XDG_CONFIG_HOME set": {
			xdgConfigHome: "/xdg",
			home:          "/home/me",
			want:          "/xdg/brewupdater/config.yaml",
		},
		"XDG_CONFIG_HOME empty": {
			home: "/home/me",
			want: "/home/me/.config/brewupdater/config.yaml",
		},
		"XDG_CONFIG_HOME relative is ignored": {
			xdgConfigHome: "rel/dir",
			home:          "/home/me",
			want:          "/home/me/.config/brewupdater/config.yaml",
		},
		"HOME unset": {
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Setenv("XDG_CONFIG_HOME", tc.xdgConfigHome)
			t.Setenv("HOME", tc.home)

			got, err := config.Path()

			if tc.wantErr {
				require.ErrorContains(t, err, "resolving home directory")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestLoad(t *testing.T) {
	tests := map[string]struct {
		setup       func(t *testing.T, path string)
		want        config.Config
		errContains []string
	}{
		"file missing": {
			setup: func(*testing.T, string) {},
			want:  config.Default(),
		},
		"empty file": {
			setup: writeFile(""),
			want:  config.Default(),
		},
		"comments only": {
			setup: writeFile("# favorites:\n#   - node\n"),
			want:  config.Default(),
		},
		"favorites only": {
			setup: writeFile("favorites:\n  - node\n  - kubectl\n"),
			want:  config.Config{Favorites: []string{"node", "kubectl"}, AutoUpdate: true},
		},
		"auto_update false": {
			setup: writeFile("auto_update: false\n"),
			want: func() config.Config {
				cfg := config.Default()
				cfg.AutoUpdate = false
				return cfg
			}(),
		},
		"greedy true": {
			setup: writeFile("greedy: true\n"),
			want:  config.Config{Greedy: true, AutoUpdate: true},
		},
		"unknown key": {
			setup:       writeFile("favourites:\n  - node\n"),
			errContains: []string{"favourites"},
		},
		"invalid YAML": {
			setup:       writeFile("favorites: [node\n"),
			errContains: []string{"parsing config", "config.yaml"},
		},
		"multiple documents": {
			setup:       writeFile("greedy: true\n---\nauto_update: false\n"),
			errContains: []string{"parsing config", "config.yaml", "multiple documents"},
		},
		"broken second document": {
			setup:       writeFile("greedy: true\n---\n: [\n"),
			errContains: []string{"parsing config", "config.yaml"},
		},
		"path is a directory": {
			setup: func(t *testing.T, path string) {
				t.Helper()
				require.NoError(t, os.Mkdir(path, 0o750))
			},
			errContains: []string{"reading config", "config.yaml"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.yaml")
			tc.setup(t, path)

			got, err := config.Load(path)

			if tc.errContains != nil {
				require.Error(t, err)
				for _, s := range tc.errContains {
					require.ErrorContains(t, err, s)
				}
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func writeFile(content string) func(t *testing.T, path string) {
	return func(t *testing.T, path string) {
		t.Helper()
		require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	}
}
