// Package main (internal test file) guards G-04-2: difftoolDiff's internal
// `git diff --no-index` invocation must never dispatch to an external-diff
// program named by GIT_EXTERNAL_DIFF or diff.external, no matter how it got
// there. Without --no-ext-diff on that call, git's own difftool builtin
// (which unconditionally sets GIT_EXTERNAL_DIFF=git-difftool--helper in the
// environment of any difftool.<name>.cmd child) causes difftoolDiff to
// re-dispatch back into git-difftool--helper, which re-invokes alturd,
// which calls difftoolDiff again — unbounded recursive process spawning
// until fork() fails. See .planning/debug/difftool-recursive-diff-loop.md
// for the empirical root-cause diagnosis (experiments A, B and C) that this
// test's four subtests re-encode as a safe, decomposed regression gate.
package main

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestDifftoolDiffIgnoresExternalDiffConfiguration asserts that
// difftoolDiff computes its own diff under both an environment-sourced and
// a configuration-sourced external-diff pointer, that the named external
// program is never actually executed, and that the identical-files exit-0
// contract loadDifftoolFiles depends on is preserved.
func TestDifftoolDiffIgnoresExternalDiffConfiguration(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH — skipping")
	}

	// Isolate gitconfig for every subtest: a developer with a real external
	// differ configured must not be able to turn this test green or red by
	// accident.
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(t.TempDir(), "gitconfig"))
	t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)

	dir := t.TempDir()
	localPath := filepath.Join(dir, "local.go")
	remotePath := filepath.Join(dir, "remote.go")
	if err := os.WriteFile(localPath, []byte("package main\n\nfunc old() {}\n"), 0o644); err != nil {
		t.Fatalf("write local file: %v", err)
	}
	if err := os.WriteFile(remotePath, []byte("package main\n\nfunc new() {}\n"), 0o644); err != nil {
		t.Fatalf("write remote file: %v", err)
	}

	// requireRealDiff reads the reader returned by difftoolDiff fully and
	// asserts it contains the hallmarks of a real, git-computed unified
	// diff. Assert on these substrings rather than the whole blob: the
	// index line's blob hashes are stable but the header format is git's,
	// not ours, and pinning it would make the test brittle across git
	// versions.
	requireRealDiff := func(t *testing.T, r io.Reader) {
		t.Helper()
		out, err := io.ReadAll(r)
		if err != nil {
			t.Fatalf("reading diff output: %v", err)
		}
		text := string(out)
		for _, want := range []string{"@@", "-func old() {}", "+func new() {}"} {
			if !strings.Contains(text, want) {
				t.Errorf("diff output missing %q; got:\n%s", want, text)
			}
		}
	}

	t.Run("external_diff_env_var", func(t *testing.T) {
		t.Setenv("GIT_EXTERNAL_DIFF", filepath.Join(dir, "no-such-external-diff"))

		r, err := difftoolDiff(localPath, remotePath)
		if err != nil {
			t.Fatalf("difftoolDiff() error = %v, want nil", err)
		}
		requireRealDiff(t, r)
	})

	t.Run("diff_external_config", func(t *testing.T) {
		cfgPath := filepath.Join(t.TempDir(), "gitconfig")
		body := "[diff]\n\texternal = " + filepath.Join(dir, "no-such-external-diff-2") + "\n"
		if err := os.WriteFile(cfgPath, []byte(body), 0o644); err != nil {
			t.Fatalf("write gitconfig: %v", err)
		}
		t.Setenv("GIT_CONFIG_GLOBAL", cfgPath)

		r, err := difftoolDiff(localPath, remotePath)
		if err != nil {
			t.Fatalf("difftoolDiff() error = %v, want nil", err)
		}
		requireRealDiff(t, r)
	})

	t.Run("external_diff_program_never_invoked", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("spy is a /bin/sh script — env-var and config subtests above already cover Windows")
		}

		spyDir := t.TempDir()
		spyPath := filepath.Join(spyDir, "spy.sh")
		markerPath := filepath.Join(spyDir, "marker")
		script := "#!/bin/sh\ntouch " + markerPath + "\nexit 0\n"
		if err := os.WriteFile(spyPath, []byte(script), 0o755); err != nil {
			t.Fatalf("write spy script: %v", err)
		}
		t.Setenv("GIT_EXTERNAL_DIFF", spyPath)

		r, err := difftoolDiff(localPath, remotePath)
		if err != nil {
			t.Fatalf("difftoolDiff() error = %v, want nil", err)
		}
		requireRealDiff(t, r)

		if _, statErr := os.Stat(markerPath); !os.IsNotExist(statErr) {
			t.Errorf("marker file exists — spy.sh was invoked, want it never invoked (stat err = %v)", statErr)
		}
	})

	t.Run("identical_files_still_exit_zero", func(t *testing.T) {
		t.Setenv("GIT_EXTERNAL_DIFF", filepath.Join(dir, "no-such-external-diff"))

		r, err := difftoolDiff(localPath, localPath)
		if err != nil {
			t.Fatalf("difftoolDiff() error = %v, want nil", err)
		}
		out, err := io.ReadAll(r)
		if err != nil {
			t.Fatalf("reading diff output: %v", err)
		}
		if len(out) != 0 {
			t.Errorf("difftoolDiff(local, local) output = %q, want empty", out)
		}
	})
}
