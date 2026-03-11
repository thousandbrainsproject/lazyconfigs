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

	data, err := os.ReadFile(node.FilePath)
	if err != nil {
		a.viewerPanel.SetText(fmt.Sprintf("[red]Error: %s[-]", tview.Escape(err.Error())))
		a.viewerPanel.SetTitle(" Viewer ")
		return
	}

	highlighted := highlightCode(string(data), "yaml")
	a.viewerPanel.SetText(highlighted)
	a.viewerPanel.ScrollToBeginning()

	// Show relative path in title
	if rel, err := filepath.Rel(filepath.Join(a.confDir, ".."), node.FilePath); err == nil {
		a.viewerPanel.SetTitle(fmt.Sprintf(" %s ", rel))
	} else {
		a.viewerPanel.SetTitle(fmt.Sprintf(" %s ", filepath.Base(node.FilePath)))
	}
}
