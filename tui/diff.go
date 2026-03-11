package main

import (
	"fmt"
	"strings"

	"github.com/pmezard/go-difflib/difflib"
	"github.com/rivo/tview"
)

func generateDiff(fromContent, toContent, fromLabel, toLabel string) (string, error) {
	diff := difflib.UnifiedDiff{
		A:        difflib.SplitLines(fromContent),
		B:        difflib.SplitLines(toContent),
		FromFile: fromLabel,
		ToFile:   toLabel,
		Context:  3,
	}
	return difflib.GetUnifiedDiffString(diff)
}

func colorizeDiff(diffText string) string {
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
			buf.WriteString(fmt.Sprintf("[green]%s[-]\n", escaped))
		case strings.HasPrefix(line, "-"):
			buf.WriteString(fmt.Sprintf("[red]%s[-]\n", escaped))
		case strings.HasPrefix(line, "@@"):
			buf.WriteString(fmt.Sprintf("[yellow]%s[-]\n", escaped))
		default:
			buf.WriteString(escaped + "\n")
		}
	}
	return buf.String()
}
