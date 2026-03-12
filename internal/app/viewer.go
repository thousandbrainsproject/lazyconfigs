// ABOUTME: Viewer panel rendering for file preview and diff display.
// ABOUTME: Handles syntax highlighting, resolved mode, and diff generation.
package app

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rivo/tview"

	"lazyconfigs/internal/hydra"
	"lazyconfigs/internal/ui"
)

// updateViewer reads the file for the given node and displays syntax-highlighted
// content in the viewer panel.
func (a *App) updateViewer(node *hydra.TreeNode) {
	if node == nil || node.FilePath == "" {
		a.viewerPanel.SetText("[darkgray]No file to display[-]")
		return
	}

	if a.resolvedMode {
		resolved, err := hydra.ResolveFile(node.FilePath, a.confDir)
		if err != nil {
			// Fall back to raw with error banner
			data, readErr := os.ReadFile(node.FilePath)
			if readErr != nil {
				a.viewerPanel.SetText(fmt.Sprintf("[red]Resolution error: %s[-]\n[red]Also failed to read raw file: %s[-]", tview.Escape(err.Error()), tview.Escape(readErr.Error())))
			} else {
				rawHighlighted := ui.HighlightCode(string(data), "yaml", a.cfg.SyntaxStyle)
				a.viewerPanel.SetText(fmt.Sprintf("[red]Resolution error: %s[-]\n\n%s", tview.Escape(err.Error()), rawHighlighted))
			}
		} else {
			highlighted := ui.HighlightCode(resolved, "yaml", a.cfg.SyntaxStyle)
			a.viewerPanel.SetText(highlighted)
		}
	} else {
		data, err := os.ReadFile(node.FilePath)
		if err != nil {
			a.viewerPanel.SetText(fmt.Sprintf("[red]Error: %s[-]", tview.Escape(err.Error())))
			a.viewerPanel.SetTitle(" Viewer ")
			return
		}
		content := string(data)
		highlighted := ui.HighlightCode(content, "yaml", a.cfg.SyntaxStyle)
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
	readContent := func(path string) (string, error) {
		if a.resolvedMode {
			return hydra.ResolveFile(path, a.confDir)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("reading %s: %w", filepath.Base(path), err)
		}
		return string(data), nil
	}

	fromContent, err := readContent(fromPath)
	if err != nil {
		a.viewerPanel.SetText(fmt.Sprintf("[red]Error loading from-file: %s[-]", tview.Escape(err.Error())))
		return
	}
	toContent, err := readContent(toPath)
	if err != nil {
		a.viewerPanel.SetText(fmt.Sprintf("[red]Error loading to-file: %s[-]", tview.Escape(err.Error())))
		return
	}

	if fromContent == toContent {
		a.viewerPanel.SetText("[darkgray]Files are identical[-]")
	} else {
		diff, err := ui.GenerateDiff(fromContent, toContent, filepath.Base(fromPath), filepath.Base(toPath))
		if err != nil {
			a.viewerPanel.SetText(fmt.Sprintf("[red]Diff error: %s[-]", tview.Escape(err.Error())))
			return
		}
		a.viewerPanel.SetText(ui.ColorizeDiff(diff, a.theme))
	}
	a.viewerPanel.ScrollToBeginning()

	title := " diff "
	if a.resolvedMode {
		title = " diff [yellow::b]resolved[-:-:-] "
	}
	a.viewerPanel.SetTitle(title)
}
