package applog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/adrg/xdg"
)

// setXDGStateHome sets XDG_STATE_HOME to dir and reloads xdg package globals.
// It registers two cleanup callbacks (in the correct LIFO order) so that
// after the test the env var is restored and xdg globals are refreshed to
// reflect the restored value — preventing package-global state leaks between
// subtests.
func setXDGStateHome(t *testing.T, dir string) {
	t.Helper()
	// Register xdg.Reload cleanup FIRST so it runs LAST (LIFO).
	// This means: t.Setenv's cleanup (restore env var) runs before our Reload,
	// so xdg sees the restored env on its final Reload.
	t.Cleanup(func() { xdg.Reload() })
	t.Setenv("XDG_STATE_HOME", dir)
	xdg.Reload()
}

func TestInit(t *testing.T) {
	t.Run("path_under_xdg_state_home", func(t *testing.T) {
		tmpDir := t.TempDir()
		setXDGStateHome(t, tmpDir)

		f, err := Init()
		if err != nil {
			t.Fatalf("Init() error = %v", err)
		}
		defer func() { _ = f.Close() }()

		gotPath := f.Name()
		wantPath := filepath.Join(tmpDir, "alturd", "alturd.log")
		if gotPath != wantPath {
			t.Errorf("Init() file path = %q, want %q", gotPath, wantPath)
		}

		if _, err := os.Stat(gotPath); err != nil {
			t.Errorf("log file does not exist after Init: %v", err)
		}
	})

	t.Run("truncates_over_cap", func(t *testing.T) {
		tmpDir := t.TempDir()
		setXDGStateHome(t, tmpDir)

		logPath := filepath.Join(tmpDir, "alturd", "alturd.log")
		if err := os.MkdirAll(filepath.Dir(logPath), 0700); err != nil {
			t.Fatalf("creating log dir: %v", err)
		}

		// Build a file larger than maxLogSize with a distinctive tail marker.
		// Total size = maxLogSize + 10000 bytes.
		tailMarker := "TAIL_MARKER_END"
		bodySize := maxLogSize + 10000 - len(tailMarker)
		body := make([]byte, bodySize)
		for i := range body {
			body[i] = 'x'
		}
		content := append(body, []byte(tailMarker)...)
		if err := os.WriteFile(logPath, content, 0600); err != nil {
			t.Fatalf("writing oversized log: %v", err)
		}

		f, err := Init()
		if err != nil {
			t.Fatalf("Init() error = %v", err)
		}
		defer func() { _ = f.Close() }()

		diskData, err := os.ReadFile(logPath)
		if err != nil {
			t.Fatalf("reading log after Init: %v", err)
		}

		if int64(len(diskData)) > maxLogSize {
			t.Errorf("file size after truncation = %d, want <= %d (maxLogSize = 1 MB)", len(diskData), maxLogSize)
		}

		// Tail must be retained (not head).
		if !strings.HasSuffix(string(diskData), tailMarker) {
			tail := diskData
			if len(tail) > 50 {
				tail = tail[len(tail)-50:]
			}
			t.Errorf("tail marker not found; file ends with: %q", string(tail))
		}
	})

	t.Run("small_file_untouched", func(t *testing.T) {
		tmpDir := t.TempDir()
		setXDGStateHome(t, tmpDir)

		logPath := filepath.Join(tmpDir, "alturd", "alturd.log")
		if err := os.MkdirAll(filepath.Dir(logPath), 0700); err != nil {
			t.Fatalf("creating log dir: %v", err)
		}

		knownContent := "SMALL_LOG_CONTENT\n"
		if err := os.WriteFile(logPath, []byte(knownContent), 0600); err != nil {
			t.Fatalf("writing small log: %v", err)
		}

		f, err := Init()
		if err != nil {
			t.Fatalf("Init() error = %v", err)
		}
		defer func() { _ = f.Close() }()

		diskData, err := os.ReadFile(logPath)
		if err != nil {
			t.Fatalf("reading log after Init: %v", err)
		}

		// Opened in O_APPEND mode; original content must be preserved at head.
		if !strings.HasPrefix(string(diskData), knownContent) {
			prefix := string(diskData)
			if len(prefix) > 60 {
				prefix = prefix[:60]
			}
			t.Errorf("small file content not preserved; file starts with: %q, want: %q", prefix, knownContent)
		}
	})
}
