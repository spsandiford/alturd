// Package config resolves alturd's user-facing TOML configuration file:
// keybinding overrides ([keybindings]) and the theme key. Load is the single
// entrypoint and must be called from cmd/alturd's RunE only — never at
// package init or PersistentPreRunE — so --help and --version never touch
// the filesystem, mirroring internal/log's Init discipline and satisfying
// D-03 (nothing is written to disk on first run).
package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/adrg/xdg"
	toml "github.com/pelletier/go-toml/v2"
)

// rawConfig is the strict TOML decode target. Keybindings is a
// map[string]string (not a fixed struct) because the ten action names are
// application-level knowledge, not TOML-schema knowledge — DisallowUnknownFields
// cannot catch a typo'd action name at decode time for this reason, so that
// check happens explicitly in Keymap.Merge instead. Theme is decoded and
// carried now so Plan 04-03 only has to add validation, not re-shape this
// struct.
type rawConfig struct {
	Theme       string            `toml:"theme"`
	Keybindings map[string]string `toml:"keybindings"`
}

// Config is alturd's fully-resolved configuration.
type Config struct {
	Theme string
	Keys  Keymap
}

// DefaultConfig returns the built-in defaults: auto theme detection and the
// ten Phase 3 default keybindings.
func DefaultConfig() *Config {
	return &Config{
		Theme: "auto",
		Keys:  DefaultKeymap(),
	}
}

// Load resolves and decodes alturd's TOML configuration.
//
// When explicitPath is non-empty, Load opens exactly that file; a missing or
// unreadable file is a startup error, never a silent fallback to defaults.
//
// When explicitPath is empty, Load performs a read-only search for
// $XDG_CONFIG_HOME/alturd/config.toml via xdg.SearchConfigFile — the
// non-creating variant of adrg/xdg's config-path helpers. The directory-
// creating write-path helper in that same package must never be used on this
// read path: it creates $XDG_CONFIG_HOME/alturd/ on disk merely by resolving
// the path, which would violate D-03. When the search finds nothing, that is
// not an error — Load returns DefaultConfig(), nil.
func Load(explicitPath string) (*Config, error) {
	path := explicitPath
	if path == "" {
		found, err := xdg.SearchConfigFile("alturd/config.toml")
		if err != nil {
			// Not found — silently use defaults (D-03).
			return DefaultConfig(), nil
		}
		path = found
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	defer f.Close()

	var raw rawConfig
	dec := toml.NewDecoder(f)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&raw); err != nil {
		var strictErr *toml.StrictMissingError
		if errors.As(err, &strictErr) && len(strictErr.Errors) > 0 {
			key := strictErr.Errors[0].Key()
			field := ""
			if len(key) > 0 {
				field = key[len(key)-1]
			}
			return nil, fmt.Errorf("config: unknown field %q in %s", field, path)
		}
		return nil, fmt.Errorf("config: %w", err)
	}

	cfg := DefaultConfig()
	if raw.Theme != "" {
		cfg.Theme = raw.Theme
	}
	if err := cfg.Keys.Merge(raw.Keybindings); err != nil {
		return nil, err
	}
	return cfg, nil
}
