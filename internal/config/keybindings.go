package config

import (
	"fmt"
	"regexp"
	"sort"
	"unicode"
)

// Action identifies one of alturd's ten rebindable global keyboard actions.
// The zero value ActionNone means "no action bound to this key" and is
// returned by Lookup when a pressed key has no binding.
type Action string

// The ten rebindable global actions. String values are the exact action
// names a user types under a config file's [keybindings] table (D-04,
// flat-snake-case schema).
const (
	ActionNone             Action = ""
	ActionNextHunk         Action = "next_hunk"
	ActionPrevHunk         Action = "prev_hunk"
	ActionNextFile         Action = "next_file"
	ActionPrevFile         Action = "prev_file"
	ActionToggleFocus      Action = "toggle_focus"
	ActionToggleRenderMode Action = "toggle_render_mode"
	ActionOpenSearch       Action = "open_search"
	ActionToggleAllFiles   Action = "toggle_all_files"
	ActionQuit             Action = "quit"
	ActionAbort            Action = "abort"
)

// canonicalActions holds the ten actions in a fixed order. Every iteration
// over actions in this package uses this slice, never Go map range order, so
// Lookup's first-match and any generated error messages are deterministic
// across runs (D-02).
var canonicalActions = []Action{
	ActionNextHunk,
	ActionPrevHunk,
	ActionNextFile,
	ActionPrevFile,
	ActionToggleFocus,
	ActionToggleRenderMode,
	ActionOpenSearch,
	ActionToggleAllFiles,
	ActionQuit,
	ActionAbort,
}

// isKnownAction reports whether name is one of the ten canonical action
// names.
func isKnownAction(name string) bool {
	for _, a := range canonicalActions {
		if string(a) == name {
			return true
		}
	}
	return false
}

// namedKeys is the explicit allowlist of multi-rune key names that
// tea.KeyPressMsg.String() produces for non-printable keys.
var namedKeys = map[string]bool{
	"tab":       true,
	"esc":       true,
	"enter":     true,
	"space":     true,
	"backspace": true,
	"delete":    true,
	"up":        true,
	"down":      true,
	"left":      true,
	"right":     true,
	"home":      true,
	"end":       true,
	"pgup":      true,
	"pgdown":    true,
}

// modifierKeyPattern matches the modifier+key form tea.KeyPressMsg.String()
// produces for a modified key, e.g. "ctrl+a".
var modifierKeyPattern = regexp.MustCompile(`^(ctrl|alt|shift)\+[a-z0-9]$`)

// validKeyString reports whether s is a form tea.KeyPressMsg.String() can
// actually produce: exactly one non-space rune, a member of namedKeys, or the
// ctrl/alt/shift modifier form. Anything else cannot be matched at dispatch
// time, so D-02 rejects it at config load time instead of letting it become
// dead, unreachable configuration (config: unrecognized key "<key>" for
// action "<action>").
func validKeyString(s string) bool {
	if s == "" {
		return false
	}
	runes := []rune(s)
	if len(runes) == 1 && !unicode.IsSpace(runes[0]) {
		return true
	}
	if namedKeys[s] {
		return true
	}
	return modifierKeyPattern.MatchString(s)
}

// Keymap maps an Action to the key string (as produced by
// tea.KeyPressMsg.String()) that dispatches it.
type Keymap map[Action]string

// DefaultKeymap returns the ten Phase 3 default keybindings, taken verbatim
// from 03-UI-SPEC.md's Key Binding Contract.
func DefaultKeymap() Keymap {
	return Keymap{
		ActionNextHunk:         "n",
		ActionPrevHunk:         "N",
		ActionNextFile:         "]",
		ActionPrevFile:         "[",
		ActionToggleFocus:      "tab",
		ActionToggleRenderMode: "v",
		ActionOpenSearch:       "/",
		ActionToggleAllFiles:   "a",
		ActionQuit:             "q",
		ActionAbort:            "Q",
	}
}

// Lookup returns the Action bound to key (a tea.KeyPressMsg.String() value),
// or ActionNone when no action is bound to that key. Iterates
// canonicalActions rather than ranging the map directly so the result is
// stable when (a config bug notwithstanding) more than one action would
// otherwise match.
//
// Examples:
//
//	DefaultKeymap().Lookup("q")   → ActionQuit
//	DefaultKeymap().Lookup("z")   → ActionNone
func (k Keymap) Lookup(key string) Action {
	for _, a := range canonicalActions {
		if k[a] == key {
			return a
		}
	}
	return ActionNone
}

// Merge applies overrides onto k, implementing D-01: only the actions named
// in overrides change; every other action keeps its existing (default) key.
// Merge builds the result in a local copy first and only writes it back into
// the receiver's underlying map once every override has been validated, so a
// rejected config never leaves k partially mutated.
//
// Validation is strict in both directions per D-02 (no silent last-one-wins):
//  1. every override action name must be one of the ten known actions;
//  2. every override key value must be a form validKeyString accepts;
//  3. the *merged* map (not the override map alone) must not bind two
//     actions to the same key — this also catches a rebind that collides
//     with a different, untouched action's default key, which validating
//     the override map alone would miss.
//
// Override names are inspected in sorted order and the merged map is scanned
// in canonicalActions order, so the reported error is identical on every run
// regardless of Go's randomized map iteration order.
func (k Keymap) Merge(overrides map[string]string) error {
	names := make([]string, 0, len(overrides))
	for name := range overrides {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		if !isKnownAction(name) {
			return fmt.Errorf("config: unknown keybinding action %q", name)
		}
	}
	for _, name := range names {
		key := overrides[name]
		if !validKeyString(key) {
			return fmt.Errorf("config: unrecognized key %q for action %q", key, name)
		}
	}

	merged := make(Keymap, len(k))
	for a, key := range k {
		merged[a] = key
	}
	for _, name := range names {
		merged[Action(name)] = overrides[name]
	}

	seen := make(map[string]Action, len(merged))
	for _, a := range canonicalActions {
		key, ok := merged[a]
		if !ok {
			continue
		}
		if first, dup := seen[key]; dup {
			return fmt.Errorf("config: key %q is bound to both %q and %q", key, first, a)
		}
		seen[key] = a
	}

	for a := range k {
		delete(k, a)
	}
	for a, key := range merged {
		k[a] = key
	}
	return nil
}
