package tui

import (
	"testing"

	"github.com/bluekeyes/go-gitdiff/gitdiff"
)

func TestBuildTree(t *testing.T) {
	t.Run("dirs_before_files", func(t *testing.T) {
		paths := []string{"src/main.go", "src/internal/foo.go", "README.md"}
		root := buildTree(paths, nil)
		if len(root.Children) == 0 {
			t.Fatal("buildTree: root has no children")
		}
		// First child must be the 'src' directory (dirs before files).
		if !root.Children[0].IsDir {
			t.Errorf("buildTree(dirs_before_files): first root child IsDir=false, want true (dirs sorted first)")
		}
	})

	t.Run("collapse_chain", func(t *testing.T) {
		paths := []string{"a/b/c/file.go"}
		root := buildTree(paths, nil)
		if len(root.Children) == 0 {
			t.Fatal("buildTree: root has no children")
		}
		// The single-child chain a→b→c should collapse to a node named "a/b/c".
		got := root.Children[0].Name
		if got != "a/b/c" {
			t.Errorf("buildTree(collapse_chain): root child Name = %q, want \"a/b/c\"", got)
		}
	})

	t.Run("status_marker", func(t *testing.T) {
		paths := []string{"foo.go"}
		sm := map[string]string{"foo.go": "[M]"}
		root := buildTree(paths, sm)
		if len(root.Children) == 0 {
			t.Fatal("buildTree: root has no children")
		}
		if root.Children[0].Status != "[M]" {
			t.Errorf("buildTree(status_marker): Status = %q, want \"[M]\"", root.Children[0].Status)
		}
	})

	t.Run("missing_status_is_empty", func(t *testing.T) {
		paths := []string{"bar.go"}
		root := buildTree(paths, nil)
		if len(root.Children) == 0 {
			t.Fatal("buildTree: root has no children")
		}
		if root.Children[0].Status != "" {
			t.Errorf("buildTree(missing_status): Status = %q, want \"\"", root.Children[0].Status)
		}
	})
}

func TestSortNode_DirsFirst(t *testing.T) {
	root := &TreeNode{IsDir: true, Children: []*TreeNode{
		{Name: "zebra.go", IsDir: false},
		{Name: "aaa", IsDir: true},
		{Name: "alpha.go", IsDir: false},
		{Name: "bbb", IsDir: true},
	}}
	sortNode(root)
	// Dirs must precede files.
	for i, c := range root.Children {
		if i < 2 && !c.IsDir {
			t.Errorf("sortNode: Children[%d] is not a dir — expected dirs first", i)
		}
		if i >= 2 && c.IsDir {
			t.Errorf("sortNode: Children[%d] is a dir — expected files after dirs", i)
		}
	}
	// Within the dir group, alphabetical.
	if root.Children[0].Name > root.Children[1].Name {
		t.Errorf("sortNode: dir group not sorted: %q > %q", root.Children[0].Name, root.Children[1].Name)
	}
}

func TestFlattenTree_RespectsExpanded(t *testing.T) {
	child := &TreeNode{Name: "child.go", IsDir: false}
	dir := &TreeNode{Name: "src", IsDir: true, Children: []*TreeNode{child}, expanded: false}
	root := &TreeNode{IsDir: true, Children: []*TreeNode{dir}}

	// Collapsed: dir should appear but child should not.
	rows := flattenTree(root, 0)
	if len(rows) != 1 {
		t.Fatalf("flattenTree(collapsed): got %d rows, want 1", len(rows))
	}
	if rows[0].node != dir {
		t.Error("flattenTree(collapsed): first row is not the directory node")
	}

	// Expanded: both dir and child should appear.
	dir.expanded = true
	rows = flattenTree(root, 0)
	if len(rows) != 2 {
		t.Fatalf("flattenTree(expanded): got %d rows, want 2", len(rows))
	}
	if rows[1].node != child {
		t.Error("flattenTree(expanded): second row is not the child node")
	}
}

func TestBuildStatusMap_EffectiveName(t *testing.T) {
	t.Run("new_name_preferred", func(t *testing.T) {
		f := &gitdiff.File{OldName: "old.go", NewName: "new.go", IsRename: true}
		m := buildStatusMap([]*gitdiff.File{f})
		if _, ok := m["new.go"]; !ok {
			t.Error("buildStatusMap: expected key 'new.go' (NewName preferred)")
		}
	})

	t.Run("old_name_fallback_for_deleted", func(t *testing.T) {
		f := &gitdiff.File{OldName: "removed.go", NewName: "/dev/null", IsDelete: true}
		m := buildStatusMap([]*gitdiff.File{f})
		if _, ok := m["removed.go"]; !ok {
			t.Error("buildStatusMap: expected key 'removed.go' (OldName fallback for deleted)")
		}
		if _, bad := m["/dev/null"]; bad {
			t.Error("buildStatusMap: should not have '/dev/null' as key")
		}
	})
}
