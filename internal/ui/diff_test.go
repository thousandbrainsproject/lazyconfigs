package ui

import (
	"strings"
	"testing"
)

func TestGenerateDiff(t *testing.T) {
	tests := []struct {
		name        string
		from        string
		to          string
		wantEmpty   bool
		wantContain []string
	}{
		{
			name:      "identical content",
			from:      "foo: bar\n",
			to:        "foo: bar\n",
			wantEmpty: true,
		},
		{
			name:        "single line change",
			from:        "foo: bar\n",
			to:          "foo: baz\n",
			wantContain: []string{"-foo: bar", "+foo: baz"},
		},
		{
			name:        "added lines",
			from:        "a: 1\n",
			to:          "a: 1\nb: 2\n",
			wantContain: []string{"+b: 2"},
		},
		{
			name:        "removed lines",
			from:        "a: 1\nb: 2\n",
			to:          "a: 1\n",
			wantContain: []string{"-b: 2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GenerateDiff(tt.from, tt.to, "from", "to")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantEmpty {
				if got != "" {
					t.Errorf("expected empty diff, got %q", got)
				}
				return
			}
			for _, s := range tt.wantContain {
				if !strings.Contains(got, s) {
					t.Errorf("expected diff to contain %q, got:\n%s", s, got)
				}
			}
		})
	}
}

func TestColorizeDiff(t *testing.T) {
	theme := testTheme()

	tests := []struct {
		name     string
		diff     string
		contains []string
	}{
		{
			name: "empty diff",
			diff: "",
		},
		{
			name:     "added line",
			diff:     "+added line\n",
			contains: []string{theme.Tags.DiffAdd, "added line"},
		},
		{
			name:     "removed line",
			diff:     "-removed line\n",
			contains: []string{theme.Tags.DiffRemove, "removed line"},
		},
		{
			name:     "hunk header",
			diff:     "@@ -1,3 +1,3 @@\n",
			contains: []string{theme.Tags.DiffHunk, "@@"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ColorizeDiff(tt.diff, theme)
			for _, s := range tt.contains {
				if !strings.Contains(result, s) {
					t.Errorf("expected result to contain %q, got %q", s, result)
				}
			}
		})
	}
}

func TestColorizeDiffHeaderLines(t *testing.T) {
	theme := testTheme()
	diff := "--- from\n+++ to\n"
	result := ColorizeDiff(diff, theme)
	if !strings.Contains(result, "[::b]") {
		t.Errorf("header lines should be bold, got %q", result)
	}
}
