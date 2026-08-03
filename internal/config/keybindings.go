package config

import "fmt"

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
// For this task (Task 2 of 04-01-PLAN.md), Merge only rejects an override
// action name that is not one of the ten known actions; unrecognized key
// strings and duplicate-key detection are added by Task 3 (D-02).
func (k Keymap) Merge(overrides map[string]string) error {
	merged := make(Keymap, len(k))
	for a, key := range k {
		merged[a] = key
	}
	for name, key := range overrides {
		if !isKnownAction(name) {
			return fmt.Errorf("config: unknown keybinding action %q", name)
		}
		merged[Action(name)] = key
	}

	for a := range k {
		delete(k, a)
	}
	for a, key := range merged {
		k[a] = key
	}
	return nil
}
