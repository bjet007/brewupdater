// Package tuitest drives bubbletea models through real keypresses, so an
// interactive surface is exercised the way a user uses it. A test run has
// no TTY, so without this seam only a widget's non-interactive fallback is ever
// reached and its keys go untested.
package tuitest

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/require"
)

// maxCommands bounds the command cascade one keypress may trigger, so a model
// that schedules itself forever fails the test rather than hanging it.
const maxCommands = 100

// namedKeys are the keys with no printable rune, spelled the way key.Binding
// matches them.
var namedKeys = map[string]tea.Key{
	"up":        {Code: tea.KeyUp},
	"down":      {Code: tea.KeyDown},
	"left":      {Code: tea.KeyLeft},
	"right":     {Code: tea.KeyRight},
	"enter":     {Code: tea.KeyEnter},
	"esc":       {Code: tea.KeyEscape},
	"tab":       {Code: tea.KeyTab},
	"home":      {Code: tea.KeyHome},
	"end":       {Code: tea.KeyEnd},
	"pgup":      {Code: tea.KeyPgUp},
	"pgdown":    {Code: tea.KeyPgDown},
	"space":     {Code: tea.KeySpace, Text: " "},
	"backspace": {Code: tea.KeyBackspace},
}

// Key builds the keypress a name denotes: a name from the table above, a
// "ctrl+"-prefixed rune, or any single printable rune.
func Key(tb testing.TB, name string) tea.KeyPressMsg {
	tb.Helper()

	if key, ok := namedKeys[name]; ok {
		return tea.KeyPressMsg(key)
	}

	if rest, ok := strings.CutPrefix(name, "ctrl+"); ok {
		runes := []rune(rest)
		require.Len(tb, runes, 1, "ctrl+ takes a single rune, got %q", name)

		return tea.KeyPressMsg{Code: runes[0], Mod: tea.ModCtrl}
	}

	runes := []rune(name)
	require.Len(tb, runes, 1, "unknown key %q", name)

	return tea.KeyPressMsg{Code: runes[0], Text: name}
}

// DriveForm feeds keys to a model the way the bubbletea runtime would: the
// commands Init and Update return are executed and their messages fed back,
// since a huh form advances its own state through commands rather than inside
// Update. Driving stops at the first quit, so keys after it are not delivered.
func DriveForm(tb testing.TB, model tea.Model, keys ...string) tea.Model {
	tb.Helper()

	model, quit := drain(tb, model, model.Init())
	if quit {
		return model
	}

	for _, name := range keys {
		updated, cmd := model.Update(Key(tb, name))

		model, quit = drain(tb, updated, cmd)
		if quit {
			return model
		}
	}

	return model
}

// Content is a model's rendered view, for asserting on what the user sees.
func Content(model tea.Model) string { return model.View().Content }

// drain runs a command and every command its messages produce, feeding each
// message back into the model. It reports whether the program was told to quit.
func drain(tb testing.TB, model tea.Model, cmd tea.Cmd) (tea.Model, bool) {
	tb.Helper()

	pending := []tea.Cmd{cmd}

	for budget := maxCommands; len(pending) > 0; budget-- {
		require.Positive(tb, budget, "command cascade did not settle")

		next := pending[0]
		pending = pending[1:]

		if next == nil {
			continue
		}

		switch msg := next().(type) {
		case nil:
		case tea.QuitMsg, tea.InterruptMsg:
			return model, true
		case tea.BatchMsg:
			pending = append(pending, msg...)
		default:
			var cmd tea.Cmd

			model, cmd = model.Update(msg)
			pending = append(pending, cmd)
		}
	}

	return model, false
}
