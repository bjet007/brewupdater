package brew_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bjet007/brewupdater/internal/brew"
	"github.com/bjet007/brewupdater/internal/plan"
)

func TestOutdatedArgs(t *testing.T) {
	tests := map[string]struct {
		greedy bool
		want   []string
	}{
		"not greedy": {
			want: []string{"outdated", "--json=v2"},
		},
		"greedy": {
			greedy: true,
			want:   []string{"outdated", "--json=v2", "--greedy"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, brew.OutdatedArgs(tc.greedy))
		})
	}
}

func TestUpgradeArgs(t *testing.T) {
	tests := map[string]struct {
		kind   plan.Kind
		names  []string
		greedy bool
		want   []string
	}{
		"formulae": {
			kind:  plan.KindFormula,
			names: []string{"a", "b"},
			want:  []string{"upgrade", "--formula", "a", "b"},
		},
		"casks, not greedy": {
			kind:  plan.KindCask,
			names: []string{"x"},
			want:  []string{"upgrade", "--cask", "x"},
		},
		"casks, greedy": {
			kind:   plan.KindCask,
			names:  []string{"x"},
			greedy: true,
			want:   []string{"upgrade", "--cask", "--greedy", "x"},
		},
		"formulae, greedy": {
			kind:   plan.KindFormula,
			names:  []string{"a"},
			greedy: true,
			want:   []string{"upgrade", "--formula", "a"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, brew.UpgradeArgs(tc.kind, tc.names, tc.greedy))
		})
	}
}
