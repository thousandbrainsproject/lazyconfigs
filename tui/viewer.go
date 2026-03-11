package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/rivo/tview"
)

// highlightCode tokenizes code with chroma and converts to tview color tags.
func highlightCode(code, language string) string {
	lexer := lexers.Get(language)
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)

	style := styles.Get("gruvbox")

	iterator, err := lexer.Tokenise(nil, code)
	if err != nil {
		return tview.Escape(code)
	}

	var buf strings.Builder
	for _, token := range iterator.Tokens() {
		text := tview.Escape(token.Value)
		entry := style.Get(token.Type)
		if entry.Colour.IsSet() {
			hex := fmt.Sprintf("#%02x%02x%02x", entry.Colour.Red(), entry.Colour.Green(), entry.Colour.Blue())
			buf.WriteString("[" + hex + "]")
			buf.WriteString(text)
			buf.WriteString("[-]")
		} else {
			buf.WriteString(text)
		}
	}
	return buf.String()
}

// updateViewer reads the file for the given node and displays syntax-highlighted
// content in the viewer panel.
func (a *App) updateViewer(node *TreeNode) {
	if node == nil || node.FilePath == "" {
		a.viewerPanel.SetText("[darkgray]No file to display[-]")
		return
	}

	var content string
	var highlighted string

	if a.resolvedMode {
		resolved, err := resolveFile(node.FilePath, a.confDir)
		if err != nil {
			// Fall back to raw with error banner
			data, _ := os.ReadFile(node.FilePath)
			rawHighlighted := highlightCode(string(data), "yaml")
			a.viewerPanel.SetText(fmt.Sprintf("[red]Resolution error: %s[-]\n\n%s", tview.Escape(err.Error()), rawHighlighted))
		} else {
			highlighted = highlightCode(resolved, "yaml")
			a.viewerPanel.SetText(highlighted)
		}
	} else {
		data, err := os.ReadFile(node.FilePath)
		if err != nil {
			a.viewerPanel.SetText(fmt.Sprintf("[red]Error: %s[-]", tview.Escape(err.Error())))
			a.viewerPanel.SetTitle(" Viewer ")
			return
		}
		content = string(data)
		highlighted = highlightCode(content, "yaml")
		a.viewerPanel.SetText(highlighted)
	}

	a.viewerPanel.ScrollToBeginning()

	// Show relative path in title
	rel := filepath.Base(node.FilePath)
	if r, err := filepath.Rel(filepath.Join(a.confDir, ".."), node.FilePath); err == nil {
		rel = r
	}

	title := fmt.Sprintf(" %s ", rel)
	if a.resolvedMode {
		title = fmt.Sprintf(" %s [yellow::b]resolved[-:-:-] ", rel)
	}
	a.viewerPanel.SetTitle(title)
}

// updateViewerDiff generates and displays a unified diff between two files.
func (a *App) updateViewerDiff(fromPath, toPath string) {
	var fromContent, toContent string
	if a.resolvedMode {
		fromContent, _ = resolveFile(fromPath, a.confDir)
		toContent, _ = resolveFile(toPath, a.confDir)
	} else {
		fromData, _ := os.ReadFile(fromPath)
		toData, _ := os.ReadFile(toPath)
		fromContent = string(fromData)
		toContent = string(toData)
	}

	if fromContent == toContent {
		a.viewerPanel.SetText("[darkgray]Files are identical[-]")
	} else {
		diff := generateDiff(fromContent, toContent, filepath.Base(fromPath), filepath.Base(toPath))
		a.viewerPanel.SetText(colorizeDiff(diff))
	}
	a.viewerPanel.ScrollToBeginning()

	title := " diff "
	if a.resolvedMode {
		title = " diff [yellow::b]resolved[-:-:-] "
	}
	a.viewerPanel.SetTitle(title)
}
