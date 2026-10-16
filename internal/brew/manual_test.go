//go:build manual

package brew_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bjet007/brewupdater/internal/brew"
)

// Run with: go test -tags manual -run TestManual -v ./internal/brew/
func TestManual(t *testing.T) {
	tests := map[string]struct {
		greedy bool
	}{
		"not greedy": {},
		"greedy":     {greedy: true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			c := brew.New()

			pkgs, err := c.Outdated(t.Context(), tc.greedy)
			require.NoError(t, err)
			for _, p := range pkgs {
				t.Logf("%-40s kind=%d %s -> %s pinned=%t", p.Name, p.Kind, p.Installed, p.Latest, p.Pinned)
			}

			installed, err := c.Installed(t.Context())
			require.NoError(t, err)
			t.Logf("%d outdated, %d installed", len(pkgs), len(installed))
		})
	}
}
