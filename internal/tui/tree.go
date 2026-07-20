// Package tui implements the bubbletea v2 terminal UI for alturd.
// It owns all interactive state: pane layout, file selection, search mode,
// and hunk navigation. Data is pre-loaded by cmd/alturd/main.go before
// tea.NewProgram is called (D-06).
package tui

import (
	"sort"
	"strings"

	"github.com/bluekeyes/go-gitdiff/gitdiff"

	"github.com/alturd/alturd/internal/diff"
)

// TreeNode is one node in the file tree displayed in the left pane.
// A node may represent a single directory, a GitHub-style collapsed chain
// (e.g. "src/internal/diff"), or a file leaf.
type TreeNode struct {
	Name     string     // display name; may contain "/" for collapsed chains (D-09)
	Children []*TreeNode
	IsDir    bool
	Path     string // full path; non-empty only for file leaves
	Status   string // "[A]", "[M]", etc.; empty for unchanged files (D-11)
	expanded bool   // for expand/collapse interaction
}

// flatRow is one displayable row in the tree viewport after flattening.
type flatRow struct {
	node  *TreeNode
	depth int
}

// buildTree builds a collapsed TreeNode hierarchy from a flat list of file paths.
// statusMap maps path → status marker (from diff.FileStatus); paths absent from
// statusMap are unchanged files (status = "").
func buildTree(paths []string, statusMap map[string]string) *TreeNode {
	root := &TreeNode{IsDir: true}
	for _, p := range paths {
		status := ""
		if statusMap != nil {
			status = statusMap[p]
		}
		insertPath(root, strings.Split(p, "/"), p, status)
	}
	collapseChain(root)
	sortNode(root)
	return root
}

func insertPath(node *TreeNode, parts []string, fullPath, status string) {
	if len(parts) == 0 {
		return
	}
	if len(parts) == 1 {
		// File leaf.
		node.Children = append(node.Children, &TreeNode{
			Name:   parts[0],
			Path:   fullPath,
			Status: status,
		})
		return
	}
	// Directory node — find or create.
	for _, c := range node.Children {
		if c.IsDir && c.Name == parts[0] {
			insertPath(c, parts[1:], fullPath, status)
			return
		}
	}
	child := &TreeNode{Name: parts[0], IsDir: true}
	node.Children = append(node.Children, child)
	insertPath(child, parts[1:], fullPath, status)
}

// collapseChain merges single-child directory chains into one node (D-09).
func collapseChain(node *TreeNode) {
	for _, c := range node.Children {
		collapseChain(c)
	}
	if node.IsDir && node.Name != "" && len(node.Children) == 1 && node.Children[0].IsDir {
		child := node.Children[0]
		node.Name = node.Name + "/" + child.Name
		node.Children = child.Children
		collapseChain(node) // may qualify again after merge
	}
}

// sortNode sorts children dirs-first, then files, each group alphabetical (TREE-01).
func sortNode(node *TreeNode) {
	sort.SliceStable(node.Children, func(i, j int) bool {
		a, b := node.Children[i], node.Children[j]
		if a.IsDir != b.IsDir {
			return a.IsDir // dirs first
		}
		return a.Name < b.Name
	})
	for _, c := range node.Children {
		sortNode(c)
	}
}

// flattenTree produces an ordered []flatRow for viewport rendering.
// Only expanded directories have their children included.
func flattenTree(node *TreeNode, depth int) []flatRow {
	var rows []flatRow
	for _, c := range node.Children {
		rows = append(rows, flatRow{node: c, depth: depth})
		if c.IsDir && c.expanded {
			rows = append(rows, flattenTree(c, depth+1)...)
		}
	}
	return rows
}

// buildStatusMap builds a path→status map from the pre-loaded diff files.
// Used by NewModel and by the 'a' toggle to mark changed files in full-repo tree.
func buildStatusMap(files []*gitdiff.File) map[string]string {
	m := make(map[string]string, len(files))
	for _, f := range files {
		path := f.NewName
		if path == "" || path == "/dev/null" {
			path = f.OldName
		}
		m[path] = diff.FileStatus(f)
	}
	return m
}

// filePaths returns the display paths of all diff files, in order.
func filePaths(files []*gitdiff.File) []string {
	paths := make([]string, 0, len(files))
	for _, f := range files {
		p := f.NewName
		if p == "" || p == "/dev/null" {
			p = f.OldName
		}
		paths = append(paths, p)
	}
	return paths
}
