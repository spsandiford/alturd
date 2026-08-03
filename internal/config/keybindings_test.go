package config_test

import (
	"testing"

	"github.com/alturd/alturd/internal/config"
)

// TestKeybindings_PartialOverride verifies D-01: overriding one action leaves
// every other action's default key untouched, and the rebound key resolves
// through Lookup to the right action while the old default key no longer
// resolves to anything.
func TestKeybindings_PartialOverride(t *testing.T) {
	k := config.DefaultKeymap()
	if err := k.Merge(map[string]string{"quit": "x"}); err != nil {
		t.Fatalf("Merge(quit=x) = %v, want nil error", err)
	}

	if got := k.Lookup("x"); got != config.ActionQuit {
		t.Errorf("Lookup(%q) = %v, want %v", "x", got, config.ActionQuit)
	}
	if got := k.Lookup("q"); got != config.ActionNone {
		t.Errorf("Lookup(%q) = %v, want %v (old default no longer bound)", "q", got, config.ActionNone)
	}
	if got := k.Lookup("n"); got != config.ActionNextHunk {
		t.Errorf("Lookup(%q) = %v, want %v (untouched default)", "n", got, config.ActionNextHunk)
	}
}

// TestKeybindings_UnknownAction verifies D-02's first validation direction:
// an override naming an action that is not one of the ten known actions is
// rejected with the exact single-line 04-UI-SPEC.md template.
func TestKeybindings_UnknownAction(t *testing.T) {
	k := config.DefaultKeymap()
	err := k.Merge(map[string]string{"nex_hunk": "n"})

	wantErr := `config: unknown keybinding action "nex_hunk"`
	if err == nil {
		t.Fatalf("Merge(nex_hunk=n) = nil error, want %q", wantErr)
	}
	if got := err.Error(); got != wantErr {
		t.Errorf("Merge(nex_hunk=n) error = %q, want %q", got, wantErr)
	}
}

// TestKeybindings_UnrecognizedKey verifies D-02's second validation
// direction: an override whose key value is not a form
// tea.KeyPressMsg.String() can produce is rejected with the exact
// single-line 04-UI-SPEC.md template.
func TestKeybindings_UnrecognizedKey(t *testing.T) {
	k := config.DefaultKeymap()
	err := k.Merge(map[string]string{"next_hunk": "ctrl+shift+meta+banana"})

	wantErr := `config: unrecognized key "ctrl+shift+meta+banana" for action "next_hunk"`
	if err == nil {
		t.Fatalf("Merge(next_hunk=ctrl+shift+meta+banana) = nil error, want %q", wantErr)
	}
	if got := err.Error(); got != wantErr {
		t.Errorf("Merge(next_hunk=ctrl+shift+meta+banana) error = %q, want %q", got, wantErr)
	}
}

// TestKeybindings_DuplicateRejected verifies D-02's merge-then-validate
// duplicate check: two actions bound to the same key are rejected, whether
// the collision is between two overridden actions or between an overridden
// action and a different, untouched action's default key. The reported
// error names the two actions in canonicalActions order so it is identical
// on every run regardless of map iteration order.
func TestKeybindings_DuplicateRejected(t *testing.T) {
	tests := []struct {
		name      string
		overrides map[string]string
		wantErr   string
	}{
		{
			name:      "duplicate_rejected",
			overrides: map[string]string{"next_hunk": "x", "prev_hunk": "x"},
			wantErr:   `config: key "x" is bound to both "next_hunk" and "prev_hunk"`,
		},
		{
			name:      "duplicate_against_untouched_default",
			overrides: map[string]string{"next_hunk": "q"},
			wantErr:   `config: key "q" is bound to both "next_hunk" and "quit"`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			k := config.DefaultKeymap()
			err := k.Merge(tc.overrides)
			if err == nil {
				t.Fatalf("Merge(%v) = nil error, want %q", tc.overrides, tc.wantErr)
			}
			if got := err.Error(); got != tc.wantErr {
				t.Errorf("Merge(%v) error = %q, want %q", tc.overrides, got, tc.wantErr)
			}
		})
	}
}

// TestKeybindings_DuplicateErrorDeterministic verifies that running the
// same duplicate-producing merge twice in a row produces byte-identical
// error messages (canonical action ordering, not map ordering).
func TestKeybindings_DuplicateErrorDeterministic(t *testing.T) {
	overrides := map[string]string{"next_hunk": "x", "prev_hunk": "x"}

	k1 := config.DefaultKeymap()
	err1 := k1.Merge(overrides)
	k2 := config.DefaultKeymap()
	err2 := k2.Merge(overrides)

	if err1 == nil || err2 == nil {
		t.Fatalf("Merge(%v) = %v, %v, want both non-nil", overrides, err1, err2)
	}
	if err1.Error() != err2.Error() {
		t.Errorf("Merge(%v) error not deterministic: %q vs %q", overrides, err1.Error(), err2.Error())
	}
}
