// Package main_test — install-difftool subprocess integration coverage
// (DIFFTOOL-03). Every test isolates git config via GIT_CONFIG_GLOBAL/
// GIT_CONFIG_SYSTEM so none of them ever reads or writes the developer's
// real ~/.gitconfig. Reuses TestMain/alturdBin from main_test.go — no
// second TestMain is declared in this package.
package main_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// isolatedEnv returns an environment slice with GIT_CONFIG_GLOBAL pointed at
// gitConfigPath and GIT_CONFIG_SYSTEM disabled, so subprocesses using it
// never touch the developer's real gitconfig files.
func isolatedEnv(gitConfigPath string) []string {
	return append(os.Environ(),
		"GIT_CONFIG_GLOBAL="+gitConfigPath,
		"GIT_CONFIG_SYSTEM="+os.DevNull,
	)
}

// runAlturd runs the built alturd binary with the given isolated env and
// working directory, returning stdout, stderr and the exit code.
func runAlturd(t *testing.T, env []string, dir string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	cmd := exec.Command(alturdBin, args...)
	cmd.Env = env
	cmd.Dir = dir
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err := cmd.Run()
	code := 0
	if err != nil {
		exitErr, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatalf("running alturd %v: %v", args, err)
		}
		code = exitErr.ExitCode()
	}
	return outBuf.String(), errBuf.String(), code
}

// gitConfigGetRaw reads a single key back from the isolated gitconfig via a
// real `git config --global --get` subprocess, independent of alturd's own
// gitConfigGet implementation, so the test asserts against ground truth.
func gitConfigGetRaw(t *testing.T, env []string, key string) (string, bool) {
	t.Helper()
	cmd := exec.Command("git", "config", "--global", "--get", key)
	cmd.Env = env
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", false
	}
	return strings.TrimRight(out.String(), "\n"), true
}

// countConfigKeys counts the total number of key=value lines in the isolated
// gitconfig via `git config --global --list`.
func countConfigKeys(t *testing.T, gitConfigPath string) int {
	t.Helper()
	cmd := exec.Command("git", "config", "--global", "--list")
	cmd.Env = isolatedEnv(gitConfigPath)
	out, err := cmd.Output()
	if err != nil {
		return 0
	}
	trimmed := strings.TrimSpace(string(out))
	if trimmed == "" {
		return 0
	}
	return len(strings.Split(trimmed, "\n"))
}

func skipIfNoGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH — skipping")
	}
}

func TestInstallDifftoolWritesFourKeys(t *testing.T) {
	skipIfNoGit(t)
	gitConfigPath := filepath.Join(t.TempDir(), "gitconfig")
	env := isolatedEnv(gitConfigPath)

	stdout, _, code := runAlturd(t, env, t.TempDir(), "install-difftool")
	if code != 0 {
		t.Fatalf("install-difftool exited %d, want 0; stdout=%q", code, stdout)
	}
	wantStdout := "Installed alturd as git difftool \"alturd\" (scope: global).\n"
	if stdout != wantStdout {
		t.Errorf("stdout = %q, want %q", stdout, wantStdout)
	}

	if v, ok := gitConfigGetRaw(t, env, "diff.tool"); !ok || v != "alturd" {
		t.Errorf("diff.tool = %q (present=%v), want \"alturd\"", v, ok)
	}
	wantCmd := `alturd --difftool-local "$LOCAL" --difftool-remote "$REMOTE" --difftool-path "$MERGED"`
	if v, ok := gitConfigGetRaw(t, env, "difftool.alturd.cmd"); !ok || v != wantCmd {
		t.Errorf("difftool.alturd.cmd = %q (present=%v), want %q", v, ok, wantCmd)
	}
	if v, ok := gitConfigGetRaw(t, env, "difftool.prompt"); !ok || v != "false" {
		t.Errorf("difftool.prompt = %q (present=%v), want \"false\"", v, ok)
	}
	if v, ok := gitConfigGetRaw(t, env, "difftool.trustExitCode"); !ok || v != "true" {
		t.Errorf("difftool.trustExitCode = %q (present=%v), want \"true\"", v, ok)
	}
}

func TestInstallDifftoolIsIdempotent(t *testing.T) {
	skipIfNoGit(t)
	gitConfigPath := filepath.Join(t.TempDir(), "gitconfig")
	env := isolatedEnv(gitConfigPath)
	dir := t.TempDir()

	if _, _, code := runAlturd(t, env, dir, "install-difftool"); code != 0 {
		t.Fatalf("first run exited %d, want 0", code)
	}
	stdout, _, code := runAlturd(t, env, dir, "install-difftool")
	if code != 0 {
		t.Fatalf("second run exited %d, want 0", code)
	}
	want := "alturd is already configured as git difftool \"alturd\" (scope: global).\n"
	if stdout != want {
		t.Errorf("stdout = %q, want %q", stdout, want)
	}
	if v, _ := gitConfigGetRaw(t, env, "diff.tool"); v != "alturd" {
		t.Errorf("diff.tool = %q, want \"alturd\" unchanged", v)
	}
	if v, ok := gitConfigGetRaw(t, env, "difftool.alturd.cmd"); !ok {
		t.Errorf("difftool.alturd.cmd missing after idempotent re-run")
	} else if want := `alturd --difftool-local "$LOCAL" --difftool-remote "$REMOTE" --difftool-path "$MERGED"`; v != want {
		t.Errorf("difftool.alturd.cmd = %q, want %q", v, want)
	}
}

func TestInstallDifftoolBlocksExistingTool(t *testing.T) {
	skipIfNoGit(t)
	gitConfigPath := filepath.Join(t.TempDir(), "gitconfig")
	env := isolatedEnv(gitConfigPath)
	dir := t.TempDir()

	seed := exec.Command("git", "config", "--global", "diff.tool", "vimdiff")
	seed.Env = env
	if out, err := seed.CombinedOutput(); err != nil {
		t.Fatalf("seeding diff.tool: %v: %s", err, out)
	}

	_, stderr, code := runAlturd(t, env, dir, "install-difftool")
	if code != 1 {
		t.Fatalf("exit = %d, want 1", code)
	}
	want := "diff.tool is already set to \"vimdiff\"; pass --force to overwrite.\n"
	if stderr != want {
		t.Errorf("stderr = %q, want %q", stderr, want)
	}
	if v, _ := gitConfigGetRaw(t, env, "diff.tool"); v != "vimdiff" {
		t.Errorf("diff.tool = %q, want \"vimdiff\" (unchanged)", v)
	}
}

func TestInstallDifftoolForceOverwrites(t *testing.T) {
	skipIfNoGit(t)
	gitConfigPath := filepath.Join(t.TempDir(), "gitconfig")
	env := isolatedEnv(gitConfigPath)
	dir := t.TempDir()

	seed := exec.Command("git", "config", "--global", "diff.tool", "vimdiff")
	seed.Env = env
	if out, err := seed.CombinedOutput(); err != nil {
		t.Fatalf("seeding diff.tool: %v: %s", err, out)
	}

	stdout, _, code := runAlturd(t, env, dir, "install-difftool", "--force")
	if code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	want := "Overwrote existing diff.tool \"vimdiff\" with \"alturd\" (scope: global, --force).\n"
	if stdout != want {
		t.Errorf("stdout = %q, want %q", stdout, want)
	}
	if v, _ := gitConfigGetRaw(t, env, "diff.tool"); v != "alturd" {
		t.Errorf("diff.tool = %q, want \"alturd\"", v)
	}
}

func TestInstallDifftoolOnlyTouchesFourKeys(t *testing.T) {
	skipIfNoGit(t)
	gitConfigPath := filepath.Join(t.TempDir(), "gitconfig")
	env := isolatedEnv(gitConfigPath)
	dir := t.TempDir()

	for _, kv := range [][2]string{{"user.name", "Test User"}, {"core.editor", "vim"}} {
		seed := exec.Command("git", "config", "--global", kv[0], kv[1])
		seed.Env = env
		if out, err := seed.CombinedOutput(); err != nil {
			t.Fatalf("seeding %s: %v: %s", kv[0], err, out)
		}
	}
	preCount := countConfigKeys(t, gitConfigPath)

	if _, _, code := runAlturd(t, env, dir, "install-difftool"); code != 0 {
		t.Fatalf("install-difftool exited %d, want 0", code)
	}

	if v, _ := gitConfigGetRaw(t, env, "user.name"); v != "Test User" {
		t.Errorf("user.name = %q, want unchanged \"Test User\"", v)
	}
	if v, _ := gitConfigGetRaw(t, env, "core.editor"); v != "vim" {
		t.Errorf("core.editor = %q, want unchanged \"vim\"", v)
	}

	postCount := countConfigKeys(t, gitConfigPath)
	if postCount != preCount+4 {
		t.Errorf("key count = %d, want %d (pre %d + 4)", postCount, preCount+4, preCount)
	}
}

func TestInstallDifftoolLocalScopeOutsideRepo(t *testing.T) {
	skipIfNoGit(t)
	gitConfigPath := filepath.Join(t.TempDir(), "gitconfig")
	env := isolatedEnv(gitConfigPath)
	nonRepoDir := t.TempDir()

	_, stderr, code := runAlturd(t, env, nonRepoDir, "install-difftool", "--scope", "local")
	if code != 1 {
		t.Fatalf("exit = %d, want 1; stderr=%q", code, stderr)
	}
	want := "--scope local requires running inside a git repository.\n"
	if stderr != want {
		t.Errorf("stderr = %q, want %q", stderr, want)
	}
}

func TestInstallDifftoolRejectsEmptyArgs(t *testing.T) {
	skipIfNoGit(t)
	gitConfigPath := filepath.Join(t.TempDir(), "gitconfig")
	env := isolatedEnv(gitConfigPath)
	dir := t.TempDir()

	if _, err := os.Stat(gitConfigPath); !os.IsNotExist(err) {
		t.Fatalf("gitconfig unexpectedly exists before test")
	}

	_, stderr1, code1 := runAlturd(t, env, dir, "install-difftool", "--name", "")
	if code1 != 1 {
		t.Errorf("--name '' exit = %d, want 1", code1)
	}
	if stderr1 != "--name must not be empty\n" {
		t.Errorf("--name '' stderr = %q", stderr1)
	}

	_, stderr2, code2 := runAlturd(t, env, dir, "install-difftool", "--scope", "")
	if code2 != 1 {
		t.Errorf("--scope '' exit = %d, want 1", code2)
	}
	if stderr2 != "--scope must be \"global\" or \"local\"\n" {
		t.Errorf("--scope '' stderr = %q", stderr2)
	}

	if _, err := os.Stat(gitConfigPath); !os.IsNotExist(err) {
		t.Errorf("gitconfig was created despite rejected empty args")
	}
}
