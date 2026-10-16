package brew_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bjet007/brewupdater/internal/brew"
	"github.com/bjet007/brewupdater/internal/plan"
)

func TestParseOutdated(t *testing.T) {
	tests := map[string]struct {
		fixture string
		want    []plan.Package
		wantErr bool
	}{
		"real output": {
			fixture: "real.json",
			want: []plan.Package{
				{Name: "abseil", Kind: plan.KindFormula, Installed: "20250814.0_1", Latest: "20260817.0"},
				{Name: "atlassian/acli/acli", Kind: plan.KindFormula, Installed: "1.3.36-stable", Latest: "1.3.39-stable"},
				{Name: "sourcetree", Kind: plan.KindCask, Installed: "4.2.7,263", Latest: "4.2.19,317"},
			},
		},
		"formulae only": {
			fixture: "formulae.json",
			want: []plan.Package{
				{Name: "node", Kind: plan.KindFormula, Installed: "24.1.0", Latest: "24.3.0"},
				{Name: "postgresql@16", Kind: plan.KindFormula, Installed: "16.4", Latest: "16.6", Pinned: true},
			},
		},
		"casks only": {
			fixture: "casks-array.json",
			want: []plan.Package{
				{Name: "sourcetree", Kind: plan.KindCask, Installed: "4.2.7,263", Latest: "4.2.19,317"},
			},
		},
		"cask installed_versions as a string": {
			fixture: "casks-string.json",
			want: []plan.Package{
				{Name: "firefox", Kind: plan.KindCask, Installed: "130.0", Latest: "131.0"},
			},
		},
		"multiple installed versions": {
			fixture: "multiple-installed.json",
			want: []plan.Package{
				{Name: "python@3.13", Kind: plan.KindFormula, Installed: "3.13.1", Latest: "3.13.2"},
			},
		},
		"empty": {
			fixture: "empty.json",
			want:    []plan.Package{},
		},
		"malformed JSON": {
			fixture: "malformed.json",
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join("testdata", tc.fixture))
			require.NoError(t, err)

			got, err := brew.ParseOutdated(data)

			if tc.wantErr {
				require.ErrorContains(t, err, "parsing brew outdated json")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}
