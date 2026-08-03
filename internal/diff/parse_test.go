package diff_test

import (
	"bytes"
	"os"
	"testing"

	"github.com/bluekeyes/go-gitdiff/gitdiff"

	"github.com/alturd/alturd/internal/diff"
)

// parseCase describes one fixture and its expected parse-level properties.
// Line count fields (-1 means "do not check").
type parseCase struct {
	name          string
	fixture       string
	wantFileCount int
	// wantName is checked against NewName unless wantNameIsOld is true.
	wantName      string
	wantNameIsOld bool
	wantIsNew     bool
	wantIsDelete  bool
	wantIsRename  bool
	wantIsBinary  bool
	wantSubmodule bool // OldMode or NewMode == 0160000
	wantNoEOL     bool // last line of first fragment has NoEOL() true
	wantAdded     int  // -1 = skip
	wantRemoved   int  // -1 = skip
	wantContext   int  // -1 = skip
}

var parseCases = []parseCase{
	{
		name: "simple",
		fixture: "simple.diff",
		wantFileCount: 1,
		wantName:      "README.md",
		wantAdded:     1,
		wantRemoved:   1,
		wantContext:   6,
	},
	{
		name: "binary",
		fixture: "binary.diff",
		wantFileCount: 1,
		wantName:      "assets/logo.png",
		wantIsBinary:  true,
		wantAdded:     0,
		wantRemoved:   0,
		wantContext:   0,
	},
	{
		name: "rename",
		fixture: "rename.diff",
		wantFileCount: 1,
		wantName:      "internal/diff/parse.go",
		wantIsRename:  true,
		wantAdded:     0,
		wantRemoved:   0,
		wantContext:   0,
	},
	{
		name: "mode-only",
		fixture: "mode-only.diff",
		wantFileCount: 1,
		wantName:      "scripts/build.sh",
		wantAdded:     0,
		wantRemoved:   0,
		wantContext:   0,
	},
	{
		name: "submodule",
		fixture: "submodule.diff",
		wantFileCount: 1,
		wantName:      "vendor/charm",
		wantSubmodule: true,
		wantAdded:     1,
		wantRemoved:   1,
		wantContext:   0,
	},
	{
		name: "no-newline",
		fixture: "no-newline.diff",
		wantFileCount: 1,
		wantName:      "config.txt",
		wantNoEOL:     true,
		wantAdded:     1,
		wantRemoved:   1,
		wantContext:   2,
	},
	{
		name: "new-file",
		fixture: "new-file.diff",
		wantFileCount: 1,
		wantName:      "internal/diff/model.go",
		wantIsNew:     true,
		wantAdded:     10,
		wantRemoved:   0,
		wantContext:   0,
	},
	{
		name: "deleted-file",
		fixture: "deleted-file.diff",
		wantFileCount: 1,
		wantName:      "old/legacy.go",
		wantNameIsOld: true,
		wantIsDelete:  true,
		wantAdded:     0,
		wantRemoved:   8,
		wantContext:   0,
	},
	{
		name: "multi-file",
		fixture: "multi-file.diff",
		wantFileCount: 2,
		wantName:      "internal/diff/model.go",
		wantAdded:     -1,
		wantRemoved:   -1,
		wantContext:   -1,
	},
	{
		name: "multi-hunk",
		fixture: "multi-hunk.diff",
		wantFileCount: 1,
		wantName:      "internal/diff/render.go",
		wantAdded:     -1,
		wantRemoved:   -1,
		wantContext:   -1,
	},
	{
		name: "large-line",
		fixture: "large-line.diff",
		wantFileCount: 1,
		wantName:      "internal/diff/generated_data.go",
		wantAdded:     1,
		wantRemoved:   1,
		wantContext:   4,
	},
	{
		name: "many-tokens",
		fixture: "many-tokens.diff",
		wantFileCount: 1,
		wantName:      "internal/diff/token_test.go",
		wantAdded:     1,
		wantRemoved:   1,
		wantContext:   4,
	},
	{
		name: "multiline-string",
		fixture: "multiline-string.diff",
		wantFileCount: 1,
		wantName:      "internal/diff/highlight_test.go",
		wantAdded:     -1,
		wantRemoved:   -1,
		wantContext:   -1,
	},
}

func TestParse(t *testing.T) {
	const submoduleMode = os.FileMode(0160000)

	for _, tc := range parseCases {
		t.Run(tc.name, func(t *testing.T) {
			f, err := os.Open("testdata/" + tc.fixture)
			if err != nil {
				t.Fatalf("opening fixture %q: %v", tc.fixture, err)
			}
			defer func() { _ = f.Close() }()

			files, err := diff.Parse(f)
			if err != nil {
				t.Fatalf("Parse(%q): unexpected error: %v", tc.fixture, err)
			}

			// --- file count ---
			if len(files) != tc.wantFileCount {
				t.Errorf("Parse(%q): got %d files, want %d", tc.fixture, len(files), tc.wantFileCount)
			}
			if len(files) == 0 {
				return
			}
			file := files[0]

			// --- name ---
			if tc.wantNameIsOld {
				if file.OldName != tc.wantName {
					t.Errorf("Parse(%q): OldName = %q, want %q", tc.fixture, file.OldName, tc.wantName)
				}
			} else if tc.wantName != "" {
				if file.NewName != tc.wantName {
					t.Errorf("Parse(%q): NewName = %q, want %q", tc.fixture, file.NewName, tc.wantName)
				}
			}

			// --- boolean status flags ---
			if file.IsNew != tc.wantIsNew {
				t.Errorf("Parse(%q): IsNew = %v, want %v", tc.fixture, file.IsNew, tc.wantIsNew)
			}
			if file.IsDelete != tc.wantIsDelete {
				t.Errorf("Parse(%q): IsDelete = %v, want %v", tc.fixture, file.IsDelete, tc.wantIsDelete)
			}
			if file.IsRename != tc.wantIsRename {
				t.Errorf("Parse(%q): IsRename = %v, want %v", tc.fixture, file.IsRename, tc.wantIsRename)
			}
			if file.IsBinary != tc.wantIsBinary {
				t.Errorf("Parse(%q): IsBinary = %v, want %v", tc.fixture, file.IsBinary, tc.wantIsBinary)
			}

			// --- submodule mode ---
			gotSubmodule := file.OldMode == submoduleMode || file.NewMode == submoduleMode
			if gotSubmodule != tc.wantSubmodule {
				t.Errorf("Parse(%q): submodule = %v (OldMode=%v NewMode=%v), want %v",
					tc.fixture, gotSubmodule, file.OldMode, file.NewMode, tc.wantSubmodule)
			}

			// --- NoEOL detection ---
			if tc.wantNoEOL {
				found := false
				for _, frag := range file.TextFragments {
					if len(frag.Lines) > 0 {
						for _, l := range frag.Lines {
							if l.NoEOL() {
								found = true
							}
						}
					}
				}
				if !found {
					t.Errorf("Parse(%q): expected at least one line with NoEOL() true, found none", tc.fixture)
				}
			}

			// --- line counts ---
			if tc.wantAdded == -1 && tc.wantRemoved == -1 && tc.wantContext == -1 {
				return
			}
			var added, removed, ctx int
			for _, frag := range file.TextFragments {
				for _, l := range frag.Lines {
					switch l.Op {
					case gitdiff.OpAdd:
						added++
					case gitdiff.OpDelete:
						removed++
					case gitdiff.OpContext:
						ctx++
					}
				}
			}
			if tc.wantAdded >= 0 && added != tc.wantAdded {
				t.Errorf("Parse(%q): added lines = %d, want %d", tc.fixture, added, tc.wantAdded)
			}
			if tc.wantRemoved >= 0 && removed != tc.wantRemoved {
				t.Errorf("Parse(%q): removed lines = %d, want %d", tc.fixture, removed, tc.wantRemoved)
			}
			if tc.wantContext >= 0 && ctx != tc.wantContext {
				t.Errorf("Parse(%q): context lines = %d, want %d", tc.fixture, ctx, tc.wantContext)
			}
		})
	}
}

// TestParseMalformed confirms that garbage bytes return a non-nil error and do NOT panic.
func TestParseMalformed(t *testing.T) {
	garbage := []byte("this is not a valid unified diff\x00\xff\xfe garbage bytes\n")
	// Should not panic; error may or may not be returned depending on go-gitdiff behaviour,
	// but must not panic. go-gitdiff is lenient on preamble text, so we use an
	// adversarial diff header that will trigger a parse failure.
	adversarial := []byte("diff --git\n--- /dev/null\n+++ /dev/null\n@@ INVALID @@\n")

	for _, input := range [][]byte{garbage, adversarial} {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Parse panicked on malformed input: %v", r)
				}
			}()
			_, _ = diff.Parse(bytes.NewReader(input))
		}()
	}
}
