package ui

import (
	"strings"
	"testing"

	"lazyconfigs/internal/config"
	"lazyconfigs/internal/hydra"
)

func testTheme() config.ThemeColors {
	return config.CompileTheme(config.ColorConfig{
		BorderFocused:   "green",
		BorderUnfocused: "default",
		Cursor:          "#6a9fb5",
		DiffFrom:        "#ff69b4",
		ActiveVariant:   "green",
		DiffAdd:         "green",
		DiffRemove:      "red",
		DiffHunk:        "yellow",
		Error:           "red",
		ValueOk:         "green",
		ValueError:      "red",
	})
}

func TestRenderItem(t *testing.T) {
	theme := testTheme()

	tests := []struct {
		name     string
		node     *hydra.TreeNode
		selected bool
		contains []string
		excludes []string
	}{
		{
			name:     "leaf not selected",
			node:     &hydra.TreeNode{Key: "model", Value: "gpt4", IsLeaf: true, Depth: 0},
			selected: false,
			contains: []string{"[::d]", "model", "gpt4"},
		},
		{
			name:     "leaf selected",
			node:     &hydra.TreeNode{Key: "model", Value: "gpt4", IsLeaf: true, Depth: 0},
			selected: true,
			contains: []string{"[::b]", "model", "gpt4", theme.Tags.Cursor},
		},
		{
			name: "expanded parent",
			node: &hydra.TreeNode{
				Key: "monty", Value: "informed", Expanded: true, Depth: 0,
				Children: []*hydra.TreeNode{{Key: "child"}},
			},
			selected: false,
			contains: []string{"▼", "monty", "informed"},
		},
		{
			name: "collapsed parent",
			node: &hydra.TreeNode{
				Key: "monty", Value: "informed", Expanded: false, Depth: 0,
				Children: []*hydra.TreeNode{{Key: "child"}},
			},
			selected: false,
			contains: []string{"▶", "monty"},
		},
		{
			name:     "error node",
			node:     &hydra.TreeNode{Key: "broken", Value: "missing", Error: "file not found", IsLeaf: true},
			selected: false,
			contains: []string{"broken", theme.Tags.ValueError},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RenderItem(tt.node, tt.selected, theme)
			for _, s := range tt.contains {
				if !strings.Contains(result, s) {
					t.Errorf("expected result to contain %q, got %q", s, result)
				}
			}
		})
	}
}

func TestRenderItemAbsolute(t *testing.T) {
	theme := testTheme()
	node := &hydra.TreeNode{Key: "monty", Value: "informed", Absolute: true, IsLeaf: true}
	result := RenderItem(node, false, theme)
	if !strings.Contains(result, "/monty") {
		t.Errorf("absolute key should have / prefix, got %q", result)
	}
}

func TestRenderVariantItem(t *testing.T) {
	theme := testTheme()

	tests := []struct {
		name           string
		varName        string
		cursorSelected bool
		isActive       bool
		isDiffFrom     bool
		contains       []string
	}{
		{
			name:     "plain",
			varName:  "default",
			contains: []string{"[::d]", "default"},
		},
		{
			name:     "active",
			varName:  "active_var",
			isActive: true,
			contains: []string{theme.Tags.ActiveVariant, "*", "active_var"},
		},
		{
			name:           "cursor selected",
			varName:        "selected_var",
			cursorSelected: true,
			contains:       []string{"[::b]", "selected_var"},
		},
		{
			name:           "active and selected",
			varName:        "both",
			cursorSelected: true,
			isActive:       true,
			contains:       []string{theme.Tags.ActiveVariant, "::b", "*", "both"},
		},
		{
			name:           "diff from selected",
			varName:        "diff_var",
			cursorSelected: true,
			isDiffFrom:     true,
			contains:       []string{theme.Tags.DiffFrom, "::b", "diff_var"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RenderVariantItem(tt.varName, tt.cursorSelected, tt.isActive, tt.isDiffFrom, theme)
			for _, s := range tt.contains {
				if !strings.Contains(result, s) {
					t.Errorf("expected result to contain %q, got %q", s, result)
				}
			}
		})
	}
}
