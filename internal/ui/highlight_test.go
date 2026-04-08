package ui

import (
	"strings"
	"testing"
)

func TestHighlightCode(t *testing.T) {
	tests := []struct {
		name      string
		code      string
		language  string
		style     string
		wantEmpty bool
		contains  string
	}{
		{
			name:     "simple yaml has color tags",
			code:     "key: value\n",
			language: "yaml",
			style:    "gruvbox",
			contains: "#",
		},
		{
			name:      "empty string",
			code:      "",
			language:  "yaml",
			style:     "gruvbox",
			wantEmpty: true,
		},
		{
			name:     "unknown language falls back",
			code:     "some text here",
			language: "nonexistent_language_xyz",
			style:    "gruvbox",
			contains: "some text here",
		},
		{
			name:     "tview brackets escaped",
			code:     "key: [value]\n",
			language: "yaml",
			style:    "gruvbox",
			contains: "[[",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HighlightCode(tt.code, tt.language, tt.style)
			if tt.wantEmpty {
				if result != "" {
					t.Errorf("expected empty, got %q", result)
				}
				return
			}
			if tt.contains != "" && !strings.Contains(result, tt.contains) {
				t.Errorf("expected result to contain %q, got %q", tt.contains, result)
			}
		})
	}
}
