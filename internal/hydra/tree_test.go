package hydra

import (
	"testing"
)

func TestFlattenTree(t *testing.T) {
	tests := []struct {
		name      string
		roots     []*TreeNode
		wantCount int
		wantKeys  []string
	}{
		{
			name:      "empty roots",
			roots:     nil,
			wantCount: 0,
		},
		{
			name:      "single leaf",
			roots:     []*TreeNode{{Key: "a", IsLeaf: true}},
			wantCount: 1,
			wantKeys:  []string{"a"},
		},
		{
			name: "all collapsed",
			roots: []*TreeNode{
				{Key: "a", Children: []*TreeNode{{Key: "b"}}},
				{Key: "c"},
			},
			wantCount: 2,
			wantKeys:  []string{"a", "c"},
		},
		{
			name: "mixed expanded and collapsed",
			roots: []*TreeNode{
				{Key: "a", Expanded: true, Children: []*TreeNode{
					{Key: "b", IsLeaf: true},
					{Key: "c", Children: []*TreeNode{{Key: "d"}}}, // collapsed
				}},
				{Key: "e"},
			},
			wantCount: 4,
			wantKeys:  []string{"a", "b", "c", "e"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FlattenTree(tt.roots)
			if len(got) != tt.wantCount {
				t.Fatalf("got %d items, want %d", len(got), tt.wantCount)
			}
			for i, key := range tt.wantKeys {
				if got[i].Key != key {
					t.Errorf("[%d] Key = %q, want %q", i, got[i].Key, key)
				}
			}
		})
	}
}

func TestFlattenTreeDepthFirst(t *testing.T) {
	// Multi-level expanded tree should produce depth-first order
	root := &TreeNode{
		Key: "root", Expanded: true,
		Children: []*TreeNode{
			{Key: "child1", Expanded: true, Children: []*TreeNode{
				{Key: "grandchild1", IsLeaf: true},
			}},
			{Key: "child2", IsLeaf: true},
		},
	}
	got := FlattenTree([]*TreeNode{root})
	wantKeys := []string{"root", "child1", "grandchild1", "child2"}
	if len(got) != len(wantKeys) {
		t.Fatalf("got %d items, want %d", len(got), len(wantKeys))
	}
	for i, key := range wantKeys {
		if got[i].Key != key {
			t.Errorf("[%d] Key = %q, want %q", i, got[i].Key, key)
		}
	}
}

func TestCollectAndRestoreExpanded(t *testing.T) {
	// Build a tree with some expanded nodes
	tree := []*TreeNode{
		{Key: "a", SourceFilePath: "/s.yaml", Expanded: true, Children: []*TreeNode{
			{Key: "b", SourceFilePath: "/a.yaml", Expanded: false},
			{Key: "c", SourceFilePath: "/a.yaml", Expanded: true},
		}},
		{Key: "d", SourceFilePath: "/s.yaml", Expanded: false},
	}

	// Collect
	expanded := CollectExpanded(tree)
	if len(expanded) != 2 {
		t.Fatalf("expected 2 expanded entries, got %d", len(expanded))
	}

	// Build a fresh tree (all collapsed)
	newTree := []*TreeNode{
		{Key: "a", SourceFilePath: "/s.yaml", Children: []*TreeNode{
			{Key: "b", SourceFilePath: "/a.yaml"},
			{Key: "c", SourceFilePath: "/a.yaml"},
		}},
		{Key: "d", SourceFilePath: "/s.yaml"},
	}

	// Restore
	RestoreExpanded(newTree, expanded)

	if !newTree[0].Expanded {
		t.Error("node 'a' should be expanded")
	}
	if newTree[0].Children[0].Expanded {
		t.Error("node 'b' should not be expanded")
	}
	if !newTree[0].Children[1].Expanded {
		t.Error("node 'c' should be expanded")
	}
	if newTree[1].Expanded {
		t.Error("node 'd' should not be expanded")
	}
}
