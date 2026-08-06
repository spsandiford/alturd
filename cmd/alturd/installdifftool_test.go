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
	"runtime"
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
	if v, ok := gitConfigGetRaw(t, env, "difftool.trustExitCode"); !ok || v != "false" {
		t.Errorf("difftool.trustExitCode = %q (present=%v), want \"false\"", v, ok)
	}
}

// TestInstallDifftoolRewritesStaleTrustExitCode is the G-04-1 regression: a
// machine that already ran the previous version of install-difftool (which
// wrote difftool.trustExitCode=true) must be converged to the corrected
// value by simply re-running install-difftool — not just a fresh machine.
func TestInstallDifftoolRewritesStaleTrustExitCode(t *testing.T) {
	skipIfNoGit(t)
	gitConfigPath := filepath.Join(t.TempDir(), "gitconfig")
	env := isolatedEnv(gitConfigPath)
	dir := t.TempDir()

	seed := exec.Command("git", "config", "--global", "difftool.trustExitCode", "true")
	seed.Env = env
	if out, err := seed.CombinedOutput(); err != nil {
		t.Fatalf("seeding stale difftool.trustExitCode: %v: %s", err, out)
	}

	_, _, code := runAlturd(t, env, dir, "install-difftool")
	if code != 0 {
		t.Fatalf("install-difftool exited %d, want 0", code)
	}

	if v, ok := gitConfigGetRaw(t, env, "difftool.trustExitCode"); !ok || v != "false" {
		t.Errorf("difftool.trustExitCode = %q (present=%v), want \"false\" (G-04-1 convergence of a stale machine)", v, ok)
	}
}

// TestGitDifftoolExitsCleanlyWhenBackendAborts is the end-to-end reproduction
// of G-04-1: a difftool backend that exits non-zero (alturd's own abort
// convention) must not make `git difftool` fatally die. A stub script stands
// in for alturd on the abort path, matching exactly what the original bug
// reporter's logging wrapper confirmed alturd itself does on abort (exit 1,
// no output).
func TestGitDifftoolExitsCleanlyWhenBackendAborts(t *testing.T) {
	skipIfNoGit(t)
	if runtime.GOOS == "windows" {
		t.Skip("writes a POSIX #!/bin/sh stub script — not applicable on windows")
	}

	gitConfigPath := filepath.Join(t.TempDir(), "gitconfig")
	env := isolatedEnv(gitConfigPath)
	repoDir := t.TempDir()

	runGit := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Env = env
		cmd.Dir = repoDir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}

	runGit("init")
	runGit("config", "user.email", "test@example.com")
	runGit("config", "user.name", "Test User")

	filePath := filepath.Join(repoDir, "file.txt")
	if err := os.WriteFile(filePath, []byte("line one\nline two\n"), 0o644); err != nil {
		t.Fatalf("writing file.txt: %v", err)
	}
	runGit("add", "file.txt")
	runGit("commit", "-m", "initial")

	// Uncommitted change so `git difftool` (index vs working tree, its
	// default comparison) has something to show.
	if err := os.WriteFile(filePath, []byte("line one\nline two changed\n"), 0o644); err != nil {
		t.Fatalf("modifying file.txt: %v", err)
	}

	// Stub backend that exits 1 — stands in for alturd on the abort path.
	stubPath := filepath.Join(t.TempDir(), "abort-stub.sh")
	if err := os.WriteFile(stubPath, []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatalf("writing stub script: %v", err)
	}

	if _, _, code := runAlturd(t, env, repoDir, "install-difftool", "--scope", "local"); code != 0 {
		t.Fatalf("install-difftool --scope local exited %d, want 0", code)
	}

	// Override the local difftool.alturd.cmd to the stub's path with a real
	// `git config` subprocess so the local scope's later write wins.
	override := exec.Command("git", "config", "--local", "difftool.alturd.cmd", stubPath)
	override.Env = env
	override.Dir = repoDir
	if out, err := override.CombinedOutput(); err != nil {
		t.Fatalf("overriding difftool.alturd.cmd: %v: %s", err, out)
	}

	runDifftool := func() (combined string, exitCode int) {
		t.Helper()
		cmd := exec.Command("git", "difftool", "-t", "alturd", "--", "file.txt")
		cmd.Env = env
		cmd.Dir = repoDir
		out, err := cmd.CombinedOutput()
		if err != nil {
			exitErr, ok := err.(*exec.ExitError)
			if !ok {
				t.Fatalf("running git difftool: %v", err)
			}
			return string(out), exitErr.ExitCode()
		}
		return string(out), 0
	}

	out, code := runDifftool()
	if code != 0 {
		t.Errorf("git difftool exit = %d, want 0; output=%q", code, out)
	}
	if strings.Contains(out, "external diff died") {
		t.Errorf("git difftool output contains a fatal external-diff line: %q", out)
	}

	// Sensitivity control: seed the local scope's trust key back to the
	// superseded value and confirm the failure mode reappears. This pins the
	// causal link between this one gitconfig value and G-04-1's symptom, so
	// the assertions above cannot pass vacuously. Only non-zero is asserted
	// (never the literal 128) so the control stays robust across the git
	// versions the three-OS CI matrix provides.
	seedStale := exec.Command("git", "config", "--local", "difftool.trustExitCode", "true")
	seedStale.Env = env
	seedStale.Dir = repoDir
	if out, err := seedStale.CombinedOutput(); err != nil {
		t.Fatalf("seeding stale local difftool.trustExitCode: %v: %s", err, out)
	}

	_, controlCode := runDifftool()
	if controlCode == 0 {
		t.Errorf("sensitivity control: git difftool exit = 0 with stale trustExitCode=true, want non-zero")
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
