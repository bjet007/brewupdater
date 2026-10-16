package checklist_test

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bjet007/brewupdater/internal/cli/checklist"
	"github.com/bjet007/brewupdater/internal/cli/tuitest"
	"github.com/bjet007/brewupdater/internal/plan"
)

var update = flag.Bool("update", false, "rewrite the golden files")

var (
	node    = plan.Package{Name: "node", Kind: plan.KindFormula, Installed: "20.1.0", Latest: "22.3.0"}
	kubectl = plan.Package{Name: "kubectl", Kind: plan.KindFormula, Installed: "1.29.0", Latest: "1.31.2", Pinned: true}
	firefox = plan.Package{Name: "firefox", Kind: plan.KindCask, Installed: "128.0", Latest: "131.0.2"}
	jq      = plan.Package{Name: "jq", Kind: plan.KindFormula, Installed: "1.6", Latest: "1.7.1"}
	wget    = plan.Package{Name: "wget", Kind: plan.KindFormula, Installed: "1.21", Latest: "1.24.5", Pinned: true}

	// fullPlan's rows, in display order: node, kubectl (pinned), firefox, jq.
	fullPlan = plan.Plan{
		Favorites: []plan.Row{
			{Package: node, Favorite: true, Preselected: true},
			{Package: kubectl, Favorite: true},
		},
		Others: []plan.Row{{Package: firefox}, {Package: jq}},
	}

	pinnedFirstPlan = plan.Plan{
		Favorites: []plan.Row{{Package: kubectl, Favorite: true}},
		Others:    []plan.Row{{Package: jq}},
	}

	noFavoritesPlan = plan.Plan{
		Others: []plan.Row{{Package: firefox}, {Package: jq}, {Package: wget}},
	}
)

func TestModel(t *testing.T) {
	tests := map[string]struct {
		plan        plan.Plan
		keys        []string
		wantCursor  int
		want        []plan.Package
		wantAborted bool
	}{
		"initial state": {
			plan:       fullPlan,
			wantCursor: 0,
			want:       []plan.Package{node},
		},
		"initial cursor skips a pinned first row": {
			plan:       pinnedFirstPlan,
			wantCursor: 1,
		},
		"down moves onto a pinned row": {
			plan:       fullPlan,
			keys:       []string{"down"},
			wantCursor: 1,
			want:       []plan.Package{node},
		},
		"down past last favorite skips the separator": {
			plan:       fullPlan,
			keys:       []string{"down", "down"},
			wantCursor: 2,
			want:       []plan.Package{node},
		},
		"j and k move like down and up": {
			plan:       fullPlan,
			keys:       []string{"j", "j", "j", "k"},
			wantCursor: 2,
			want:       []plan.Package{node},
		},
		"up at the top stays": {
			plan:       fullPlan,
			keys:       []string{"up"},
			wantCursor: 0,
			want:       []plan.Package{node},
		},
		"down at the bottom stays": {
			plan:       fullPlan,
			keys:       []string{"down", "down", "down", "down"},
			wantCursor: 3,
			want:       []plan.Package{node},
		},
		"space on pinned does nothing": {
			plan:       fullPlan,
			keys:       []string{"down", "space"},
			wantCursor: 1,
			want:       []plan.Package{node},
		},
		"space unticks a ticked row": {
			plan:       fullPlan,
			keys:       []string{"space"},
			wantCursor: 0,
		},
		"space ticks an unticked row": {
			plan:       fullPlan,
			keys:       []string{"down", "down", "space"},
			wantCursor: 2,
			want:       []plan.Package{node, firefox},
		},
		"a ticks all selectable rows": {
			plan: fullPlan,
			keys: []string{"a"},
			want: []plan.Package{node, firefox, jq},
		},
		"a twice unticks all": {
			plan: fullPlan,
			keys: []string{"a", "a"},
		},
		"enter returns ticked packages in display order": {
			plan:       fullPlan,
			keys:       []string{"down", "down", "down", "space", "up", "space", "enter"},
			wantCursor: 2,
			want:       []plan.Package{node, firefox, jq},
		},
		"esc aborts": {
			plan:        fullPlan,
			keys:        []string{"esc"},
			want:        []plan.Package{node},
			wantAborted: true,
		},
		"ctrl+c aborts": {
			plan:        fullPlan,
			keys:        []string{"ctrl+c"},
			want:        []plan.Package{node},
			wantAborted: true,
		},
		"keys after enter are ignored": {
			plan: fullPlan,
			keys: []string{"enter", "a"},
			want: []plan.Package{node},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			driven := tuitest.DriveForm(t, checklist.NewModel(tc.plan), tc.keys...)

			got, aborted := checklist.Outcome(driven)
			assert.Equal(t, tc.wantCursor, checklist.Cursor(driven))
			assert.Equal(t, tc.want, got)
			assert.Equal(t, tc.wantAborted, aborted)
		})
	}
}

func TestView(t *testing.T) {
	tests := map[string]struct {
		plan plan.Plan
		keys []string
	}{
		"full":         {plan: fullPlan},
		"no favorites": {plan: noFavoritesPlan, keys: []string{"down", "space"}},
		"only favorites": {plan: plan.Plan{
			Favorites: []plan.Row{{Package: node, Favorite: true, Preselected: true}},
		}},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			driven := tuitest.DriveForm(t, checklist.NewModel(tc.plan), tc.keys...)
			got := ansi.Strip(tuitest.Content(driven))

			path := filepath.Join("testdata", strings.ReplaceAll(name, " ", "_")+".golden")
			if *update {
				require.NoError(t, os.WriteFile(path, []byte(got), 0o600))
			}
			want, err := os.ReadFile(path)
			require.NoError(t, err)
			assert.Equal(t, string(want), got)
		})
	}
}

// longPlan's rows, in display order: node, kubectl (pinned), then pkg01 to pkg10.
var longPlan = func() plan.Plan {
	p := plan.Plan{Favorites: fullPlan.Favorites}
	for i := 1; i <= 10; i++ {
		name := fmt.Sprintf("pkg%02d", i)
		p.Others = append(p.Others, plan.Row{Package: plan.Package{Name: name, Kind: plan.KindFormula, Installed: "1", Latest: "2"}})
	}
	return p
}()

var rowName = regexp.MustCompile(`(\S+)\s+\[(?:formula|cask)\]`)

func TestViewport(t *testing.T) {
	tests := map[string]struct {
		height     int
		keys       []string
		resizeTo   int
		wantRows   []string
		wantCursor string
		wantHelp   string
		wantTicked []string
	}{
		"unknown height draws every row": {
			wantRows: []string{
				"node", "kubectl", "pkg01", "pkg02", "pkg03", "pkg04", "pkg05", "pkg06", "pkg07", "pkg08", "pkg09", "pkg10",
			},
			wantCursor: "node",
			wantTicked: []string{"node"},
		},
		"window starts at the top": {
			height:     9,
			wantRows:   []string{"node", "kubectl", "pkg01", "pkg02"},
			wantCursor: "node",
			wantHelp:   "1-4 of 12",
			wantTicked: []string{"node"},
		},
		"down past the window scrolls one row": {
			height:     9,
			keys:       []string{"down", "down", "down", "down"},
			wantRows:   []string{"kubectl", "pkg01", "pkg02", "pkg03"},
			wantCursor: "pkg03",
			wantHelp:   "2-5 of 12",
			wantTicked: []string{"node"},
		},
		"up past the window scrolls back": {
			height:     9,
			keys:       []string{"down", "down", "down", "down", "down", "up", "up", "up", "up", "up"},
			wantRows:   []string{"node", "kubectl", "pkg01", "pkg02"},
			wantCursor: "node",
			wantHelp:   "1-4 of 12",
			wantTicked: []string{"node"},
		},
		"space ticks the row under the cursor once scrolled": {
			height:     9,
			keys:       []string{"down", "down", "down", "down", "down", "down", "space"},
			wantRows:   []string{"pkg02", "pkg03", "pkg04", "pkg05"},
			wantCursor: "pkg05",
			wantHelp:   "4-7 of 12",
			wantTicked: []string{"node", "pkg05"},
		},
		"shrinking the terminal keeps the cursor on screen": {
			height:     40,
			keys:       []string{"down", "down", "down", "down", "down", "down", "down", "down", "down", "down", "down"},
			resizeTo:   9,
			wantRows:   []string{"pkg07", "pkg08", "pkg09", "pkg10"},
			wantCursor: "pkg10",
			wantHelp:   "9-12 of 12",
			wantTicked: []string{"node"},
		},
		"growing the terminal shows the rows above": {
			height:   9,
			keys:     []string{"down", "down", "down", "down", "down", "down", "down", "down", "down", "down", "down"},
			resizeTo: 40,
			wantRows: []string{
				"node", "kubectl", "pkg01", "pkg02", "pkg03", "pkg04", "pkg05", "pkg06", "pkg07", "pkg08", "pkg09", "pkg10",
			},
			wantCursor: "pkg10",
			wantTicked: []string{"node"},
		},
		"a ticks rows outside the window": {
			height:     9,
			keys:       []string{"down", "down", "down", "down", "down", "down", "a"},
			wantRows:   []string{"pkg02", "pkg03", "pkg04", "pkg05"},
			wantCursor: "pkg05",
			wantHelp:   "4-7 of 12",
			wantTicked: []string{
				"node", "pkg01", "pkg02", "pkg03", "pkg04", "pkg05", "pkg06", "pkg07", "pkg08", "pkg09", "pkg10",
			},
		},
		"tiny terminal still draws the cursor row": {
			height:     6,
			keys:       []string{"down", "down"},
			wantRows:   []string{"pkg01"},
			wantCursor: "pkg01",
			wantHelp:   "3-3 of 12",
			wantTicked: []string{"node"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			model := checklist.NewModel(longPlan)
			if tc.height > 0 {
				model, _ = model.Update(tea.WindowSizeMsg{Width: 80, Height: tc.height})
			}
			model = tuitest.DriveForm(t, model, tc.keys...)
			if tc.resizeTo > 0 {
				model, _ = model.Update(tea.WindowSizeMsg{Width: 80, Height: tc.resizeTo})
			}
			lines := strings.Split(ansi.Strip(tuitest.Content(model)), "\n")

			var rows []string
			var cursor string
			for _, l := range lines {
				match := rowName.FindStringSubmatch(l)
				if match == nil {
					continue
				}
				rows = append(rows, match[1])
				if strings.HasPrefix(l, ">") {
					cursor = match[1]
				}
			}
			got, _ := checklist.Outcome(model)
			ticked := make([]string, 0, len(got))
			for _, pkg := range got {
				ticked = append(ticked, pkg.Name)
			}

			assert.Equal(t, tc.wantRows, rows)
			assert.Equal(t, tc.wantCursor, cursor)
			assert.Equal(t, tc.wantTicked, ticked)
			height := tc.height
			if tc.resizeTo > 0 {
				height = tc.resizeTo
			}
			if height > 0 {
				assert.LessOrEqual(t, len(lines), height)
			}
			if tc.wantHelp == "" {
				assert.NotContains(t, lines[len(lines)-1], " of ")
			} else {
				assert.True(t, strings.HasPrefix(lines[len(lines)-1], tc.wantHelp+" · "), lines[len(lines)-1])
			}
		})
	}
}

// faint is the SGR parameter for dimmed text, which the golden files strip.
const faint = "\x1b[2;"

func TestViewDimsPinnedRows(t *testing.T) {
	tests := map[string]struct {
		pkg       string
		wantFaint bool
	}{
		"pinned favorite":     {pkg: kubectl.Name, wantFaint: true},
		"selectable favorite": {pkg: node.Name},
		"selectable other":    {pkg: jq.Name},
	}

	lines := strings.Split(tuitest.Content(checklist.NewModel(fullPlan)), "\n")
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var line string
			for _, l := range lines {
				if strings.Contains(l, tc.pkg) {
					line = l
				}
			}
			require.NotEmpty(t, line)
			assert.Equal(t, tc.wantFaint, strings.Contains(line, faint))
		})
	}
}

func TestViewIsEmptyOnceSettled(t *testing.T) {
	tests := map[string]struct {
		key string
	}{
		"submitted": {key: "enter"},
		"aborted":   {key: "esc"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			driven := tuitest.DriveForm(t, checklist.NewModel(fullPlan), tc.key)

			assert.Empty(t, tuitest.Content(driven))
		})
	}
}

// The bytes a terminal sends, since a headless program parses its own input.
const (
	inputSpace = " "
	inputEnter = "\r"
	inputDown  = "\x1b[B"
	inputEsc   = "\x1b"
	inputCtrlC = "\x03"
)

func TestRun(t *testing.T) {
	tests := map[string]struct {
		plan    plan.Plan
		input   string
		want    []plan.Package
		wantErr error
	}{
		"empty plan returns without running": {
			plan: plan.Plan{},
		},
		"enter submits": {
			plan:  fullPlan,
			input: inputDown + inputDown + inputSpace + inputEnter,
			want:  []plan.Package{node, firefox},
		},
		"esc aborts": {
			plan:    fullPlan,
			input:   inputSpace + inputEsc,
			wantErr: checklist.ErrAborted,
		},
		"ctrl+c aborts": {
			plan:    fullPlan,
			input:   inputCtrlC,
			wantErr: checklist.ErrAborted,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := checklist.RunWith(context.Background(), tc.plan,
				tea.WithInput(strings.NewReader(tc.input)),
				tea.WithoutRenderer(),
				tea.WithoutSignals(),
			)

			require.ErrorIs(t, err, tc.wantErr)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestRunCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := checklist.RunWith(ctx, fullPlan,
		tea.WithInput(strings.NewReader("")),
		tea.WithoutRenderer(),
		tea.WithoutSignals(),
	)

	require.Error(t, err)
	assert.NotErrorIs(t, err, checklist.ErrAborted)
}
