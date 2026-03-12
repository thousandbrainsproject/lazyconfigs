package ui

import (
	"fmt"
	"strings"

	"github.com/pmezard/go-difflib/difflib"
	"github.com/rivo/tview"

	"lazyconfigs/internal/config"
)

func GenerateDiff(fromContent, toContent, fromLabel, toLabel string) (string, error) {
	diff := difflib.UnifiedDiff{
		A:        difflib.SplitLines(fromContent),
		B:        difflib.SplitLines(toContent),
		FromFile: fromLabel,
		ToFile:   toLabel,
		Context:  3,
	}
	return difflib.GetUnifiedDiffString(diff)
}

func ColorizeDiff(diffText string, theme config.ThemeColors) string {
	lines := strings.Split(diffText, "\n")
	// strings.Split produces a trailing empty element when input ends with "\n"
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	var buf strings.Builder
	for _, line := range lines {
		escaped := tview.Escape(line)
		switch {
		case strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---"):
			buf.WriteString(fmt.Sprintf("[::b]%s[-:-:-]\n", escaped))
		case strings.HasPrefix(line, "+"):
			buf.WriteString(fmt.Sprintf("[%s]%s[-]\n", theme.Tags.DiffAdd, escaped))
		case strings.HasPrefix(line, "-"):
			buf.WriteString(fmt.Sprintf("[%s]%s[-]\n", theme.Tags.DiffRemove, escaped))
		case strings.HasPrefix(line, "@@"):
			buf.WriteString(fmt.Sprintf("[%s]%s[-]\n", theme.Tags.DiffHunk, escaped))
		default:
			buf.WriteString(escaped + "\n")
		}
	}
	return buf.String()
}
