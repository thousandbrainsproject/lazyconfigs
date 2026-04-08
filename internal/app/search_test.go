package app

import (
	"testing"
)

func TestFuzzyMatch(t *testing.T) {
	tests := []struct {
		name   string
		query  string
		target string
		want   bool
	}{
		{"exact match", "model", "model", true},
		{"subsequence", "mdl", "model", true},
		{"case insensitive", "MDL", "model", true},
		{"no match", "xyz", "model", false},
		{"empty query matches anything", "", "anything", true},
		{"empty target no match", "a", "", false},
		{"both empty", "", "", true},
		{"query longer than target", "models", "mdl", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fuzzyMatch(tt.query, tt.target)
			if got != tt.want {
				t.Errorf("fuzzyMatch(%q, %q) = %v, want %v", tt.query, tt.target, got, tt.want)
			}
		})
	}
}
