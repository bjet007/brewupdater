package tuitest_test

import (
	"errors"
	"fmt"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bjet007/brewupdater/internal/cli/tuitest"
)

// The keys the recorder model below answers, one per command shape DriveForm has
// to handle.
const (
	keyEnter     = "enter"
	keyInert     = "n"
	keyPlain     = "x"
	keyQuit      = "q"
	keyInterrupt = "i"
	keyBatch     = "b"
	keyEmptyCmd  = "e"

	noteInit = "init"
)

// errStubFailed is what a stubbed TB panics with in place of failing the real
// test, so the helper's own guard rails can be asserted on.
var errStubFailed = errors.New("stub failed")

// stubTB stands in for the test's own TB wherever the code under test is
// expected to fail it.
type stubTB struct {
	testing.TB

	messages []string
}

func (s *stubTB) Helper() {}

func (s *stubTB) Name() string { return "stub" }

func (s *stubTB) Errorf(format string, args ...any) {
	s.messages = append(s.messages, fmt.Sprintf(format, args...))
}

func (s *stubTB) FailNow() { panic(errStubFailed) }

// expectFailure runs fn against a stubbed TB and asserts that it failed it.
func expectFailure(t *testing.T, fn func(tb testing.TB)) {
	t.Helper()

	stub := &stubTB{}

	defer func() {
		recovered := recover()
		require.Equal(t, errStubFailed, recovered, "the helper should have failed the test")
		assert.NotEmpty(t, stub.messages, "a failure should say why")
	}()

	fn(stub)
}

func TestKey(t *testing.T) {
	tests := map[string]struct {
		key  string
		want string
	}{
		"an arrow":                         {key: "up", want: "up"},
		"a named key":                      {key: keyEnter, want: keyEnter},
		"escape":                           {key: "esc", want: "esc"},
		"space, which has no visible rune": {key: "space", want: "space"},
		"a printable rune":                 {key: keyPlain, want: keyPlain},
		"a modified rune":                  {key: "ctrl+c", want: "ctrl+c"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.want, tuitest.Key(t, tt.key).String())
		})
	}
}

func TestKeyRejectsUnknownNames(t *testing.T) {
	tests := map[string]struct {
		key string
	}{
		"an unnamed multi-rune key": {key: "wiggle"},
		"a multi-rune modified key": {key: "ctrl+shift"},
		"nothing at all":            {key: ""},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			expectFailure(t, func(tb testing.TB) {
				tuitest.Key(tb, tt.key)
			})
		})
	}
}

type noteMsg string

// recorder reports back what it was driven with. Its commands cover the cascade
// DriveForm has to run on a model's behalf.
type recorder struct {
	keys  []string
	notes []string
	// loop makes every message schedule another, without end.
	loop bool
	// quitOnInit ends the program before the first key is ever sent.
	quitOnInit bool
}

func (r *recorder) Init() tea.Cmd {
	if r.quitOnInit {
		return tea.Quit
	}

	return note(noteInit)
}

func (r *recorder) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case noteMsg:
		r.notes = append(r.notes, string(msg))

		if r.loop {
			return r, note("again")
		}
	case tea.KeyPressMsg:
		r.keys = append(r.keys, msg.String())

		switch msg.String() {
		case keyQuit:
			return r, tea.Quit
		case keyInterrupt:
			return r, tea.Interrupt
		case keyBatch:
			return r, tea.Batch(note("first"), note("second"))
		case keyEmptyCmd:
			return r, func() tea.Msg { return nil }
		}
	}

	return r, nil
}

func (r *recorder) View() tea.View {
	return tea.NewView("keys: " + join(r.keys) + "\nnotes: " + join(r.notes))
}

func join(values []string) string {
	out := ""

	for i, value := range values {
		if i > 0 {
			out += ","
		}

		out += value
	}

	return out
}

func TestDriveForm(t *testing.T) {
	tests := map[string]struct {
		keys      []string
		wantKeys  []string
		wantNotes []string
	}{
		"the init command runs before any key": {
			wantNotes: []string{noteInit},
		},
		"keys arrive in order": {
			keys:      []string{keyInert, keyPlain},
			wantKeys:  []string{keyInert, keyPlain},
			wantNotes: []string{noteInit},
		},
		"a batch is unwrapped and every message fed back": {
			keys:      []string{keyBatch},
			wantKeys:  []string{keyBatch},
			wantNotes: []string{noteInit, "first", "second"},
		},
		"a command producing no message is passed over": {
			keys:      []string{keyEmptyCmd, keyInert},
			wantKeys:  []string{keyEmptyCmd, keyInert},
			wantNotes: []string{noteInit},
		},
		"quitting stops the keys after it": {
			keys:      []string{keyInert, keyQuit, keyPlain},
			wantKeys:  []string{keyInert, keyQuit},
			wantNotes: []string{noteInit},
		},
		"an interrupt stops them too": {
			keys:      []string{keyInterrupt, keyPlain},
			wantKeys:  []string{keyInterrupt},
			wantNotes: []string{noteInit},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			driven := tuitest.DriveForm(t, &recorder{}, tt.keys...)

			model, ok := driven.(*recorder)
			require.True(t, ok)

			assert.Equal(t, tt.wantKeys, model.keys)
			assert.Equal(t, tt.wantNotes, model.notes)
		})
	}
}

func TestDriveFormStopsWhenInitQuits(t *testing.T) {
	driven := tuitest.DriveForm(t, &recorder{quitOnInit: true}, keyPlain)

	model, ok := driven.(*recorder)
	require.True(t, ok)
	assert.Empty(t, model.keys, "no key should reach a program that quit on init")
}

func TestDriveFormBoundsAnEndlessCascade(t *testing.T) {
	expectFailure(t, func(tb testing.TB) {
		tuitest.DriveForm(tb, &recorder{loop: true})
	})
}

func TestContent(t *testing.T) {
	driven := tuitest.DriveForm(t, &recorder{}, keyBatch)
	assert.Equal(t, "keys: "+keyBatch+"\nnotes: "+noteInit+",first,second", tuitest.Content(driven))
}

func note(text string) tea.Cmd {
	return func() tea.Msg { return noteMsg(text) }
}
