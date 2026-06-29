// Package applog provides XDG-based log file initialization for alturd.
// It exposes a single Init entrypoint that resolves the cross-platform state
// path, caps the log file at 1 MB (tail-retaining truncation), and redirects
// charmbracelet/log output to the file (never to stderr). The caller defers
// Close on the returned *os.File.
//
// Init must be called from cmd/alturd RunE only — never at package init or
// PersistentPreRunE — so --help/--version never create a log file (D-10).
package applog

import (
	"fmt"
	"os"

	charmlog "github.com/charmbracelet/log"

	"github.com/adrg/xdg"
)

// maxLogSize is the maximum allowed log file size before tail truncation.
const maxLogSize = 1 << 20 // 1 MB

// Init opens the log file at $XDG_STATE_HOME/alturd/alturd.log, truncates it
// to the trailing maxLogSize bytes if it exceeds the cap, redirects
// charmbracelet/log output to the file, and returns the open *os.File so the
// caller can defer f.Close().
func Init() (*os.File, error) {
	path, err := xdg.StateFile("alturd/alturd.log")
	if err != nil {
		return nil, fmt.Errorf("resolving log path: %w", err)
	}

	// Truncate before opening for append so the cap is enforced on startup.
	if fi, statErr := os.Stat(path); statErr == nil && fi.Size() > maxLogSize {
		if err := truncateLog(path); err != nil {
			return nil, fmt.Errorf("truncating log: %w", err)
		}
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("opening log file: %w", err)
	}

	charmlog.SetOutput(f)
	return f, nil
}

// truncateLog keeps the LAST maxLogSize bytes of the file, discarding older
// entries. This retains the most recent log content (tail), unlike
// os.Truncate which would retain the head (oldest entries).
func truncateLog(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading log for truncation: %w", err)
	}
	if int64(len(data)) > maxLogSize {
		data = data[int64(len(data))-maxLogSize:]
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("writing truncated log: %w", err)
	}
	return nil
}
