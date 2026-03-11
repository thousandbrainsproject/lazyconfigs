package main

import (
	"fmt"
	"strings"

	"github.com/pmezard/go-difflib/difflib"
	"github.com/rivo/tview"
)

func generateDiff(fromContent, toContent, fromLabel, toLabel string) string {
	diff := difflib.UnifiedDiff{
		A:        difflib.SplitLines(fromContent),
		B:        difflib.SplitLines(toContent),
		FromFile: fromLabel,
		ToFile:   toLabel,
		Context:  3,
	}
	text, _ := difflib.GetUnifiedDiffString(diff)
	return text
}

func colorizeDiff(diffText string) string {
	var buf strings.Builder
	for _, line := range strings.Split(diffText, "\n") {
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
