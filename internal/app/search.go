// ABOUTME: Fuzzy search functionality for builder and variant panels.
// ABOUTME: Filters items in real-time as the user types a search query.
package app

import (
	"fmt"
	"strings"

	"github.com/rivo/tview"

	"lazyconfigs/internal/hydra"
	"lazyconfigs/internal/ui"
	"lazyconfigs/internal/version"
)

// fuzzyMatch returns true if all characters in query appear in target
// in order (case-insensitive, not necessarily contiguous).
func fuzzyMatch(query, target string) bool {
	qRunes := []rune(strings.ToLower(query))
	qi := 0
	for _, r := range strings.ToLower(target) {
		if qi < len(qRunes) && qRunes[qi] == r {
			qi++
		}
	}
	return qi == len(qRunes)
}

// enterSearchMode activates fuzzy search on the currently focused panel.
func (a *App) enterSearchMode() {
	a.searchMode = true
	a.searchQuery = ""
	a.searchPanel = a.currentPanelIdx
	a.updateSearchStatus()
}

// exitSearchMode restores the full list and maps the current selection
// back to its position in the unfiltered list.
func (a *App) exitSearchMode() {
	a.searchMode = false

	// Find the currently selected item in the visible (filtered) list
	// so we can restore cursor position in the full list.
	switch a.searchPanel {
	case 0:
		var selectedNode *hydra.TreeNode
		idx := a.builderPanel.GetCurrentItem()
		if idx >= 0 && idx < len(a.visibleBuilderItems) {
			selectedNode = a.visibleBuilderItems[idx]
		}
		// Restore full list
		a.visibleBuilderItems = a.flatItems
		a.builderPanel.Clear()
		for i, node := range a.flatItems {
			a.builderPanel.AddItem(ui.RenderItem(node, i == 0, a.theme), "", 0, nil)
		}
		// Restore cursor to the previously selected item
		if selectedNode != nil {
			for i, node := range a.flatItems {
				if node == selectedNode {
					a.builderPanel.SetCurrentItem(i)
					break
				}
			}
		}
	case 1:
		var selectedName string
		idx := a.variantsPanel.GetCurrentItem()
		if idx >= 0 && idx < len(a.visibleVariantFiles) {
			selectedName = a.visibleVariantFiles[idx]
		}
		// Restore full list
		a.visibleVariantFiles = a.variantFiles
		activeValue := ""
		if a.selectedBuilderNode != nil {
			activeValue = a.selectedBuilderNode.Value
		}
		a.variantsPanel.Clear()
		for i, name := range a.variantFiles {
			isDiffFrom := a.diffMode && i == a.diffFromIdx
			a.variantsPanel.AddItem(ui.RenderVariantItem(name, i == 0, name == activeValue, isDiffFrom, a.theme), "", 0, nil)
		}
		// Restore cursor
		if selectedName != "" {
			for i, name := range a.variantFiles {
				if name == selectedName {
					a.variantsPanel.SetCurrentItem(i)
					break
				}
			}
		}
	}

	a.searchQuery = ""
	a.updateStatusBar()
	a.statusBarRight.SetText("[blue::b]Thousand Brains Project[-:-:-] " + version.Version + " ")
}

// applySearch dispatches filtering to the correct panel.
func (a *App) applySearch() {
	switch a.searchPanel {
	case 0:
		a.applyBuilderSearch(a.searchQuery)
	case 1:
		a.applyVariantsSearch(a.searchQuery)
	}
	a.updateSearchStatus()
}

// applyBuilderSearch filters flatItems by fuzzy matching and rebuilds the builder panel.
func (a *App) applyBuilderSearch(query string) {
	if query == "" {
		a.visibleBuilderItems = a.flatItems
	} else {
		var filtered []*hydra.TreeNode
		for _, node := range a.flatItems {
			target := node.Key + ": " + node.Value
			if fuzzyMatch(query, target) {
				filtered = append(filtered, node)
			}
		}
		a.visibleBuilderItems = filtered
	}

	a.builderPanel.Clear()
	for i, node := range a.visibleBuilderItems {
		a.builderPanel.AddItem(ui.RenderItem(node, i == 0, a.theme), "", 0, nil)
	}
	if len(a.visibleBuilderItems) > 0 {
		a.builderPanel.SetCurrentItem(0)
	}
}

// applyVariantsSearch filters variantFiles by fuzzy matching and rebuilds the variants panel.
func (a *App) applyVariantsSearch(query string) {
	if query == "" {
		a.visibleVariantFiles = a.variantFiles
	} else {
		var filtered []string
		for _, name := range a.variantFiles {
			if fuzzyMatch(query, name) {
				filtered = append(filtered, name)
			}
		}
		a.visibleVariantFiles = filtered
	}

	activeValue := ""
	if a.selectedBuilderNode != nil {
		activeValue = a.selectedBuilderNode.Value
	}
	a.variantsPanel.Clear()
	for i, name := range a.visibleVariantFiles {
		isDiffFrom := false // diff-from index doesn't map to filtered list
		a.variantsPanel.AddItem(ui.RenderVariantItem(name, i == 0, name == activeValue, isDiffFrom, a.theme), "", 0, nil)
	}
	if len(a.visibleVariantFiles) > 0 {
		a.variantsPanel.SetCurrentItem(0)
	}
}

// searchList returns the list panel being searched.
func (a *App) searchList() *tview.List {
	if a.searchPanel == 0 {
		return a.builderPanel
	}
	return a.variantsPanel
}

// searchCursorDown moves the cursor down in the filtered list during search mode.
func (a *App) searchCursorDown() {
	list := a.searchList()
	current := list.GetCurrentItem()
	if current < list.GetItemCount()-1 {
		list.SetCurrentItem(current + 1)
	}
}

// searchCursorUp moves the cursor up in the filtered list during search mode.
func (a *App) searchCursorUp() {
	list := a.searchList()
	current := list.GetCurrentItem()
	if current > 0 {
		list.SetCurrentItem(current - 1)
	}
}

// updateSearchStatus sets the status bar to show the search query and match count.
func (a *App) updateSearchStatus() {
	var total, matched int
	switch a.searchPanel {
	case 0:
		total = len(a.flatItems)
		matched = len(a.visibleBuilderItems)
	case 1:
		total = len(a.variantFiles)
		matched = len(a.visibleVariantFiles)
	}

	a.statusBarLeft.SetText(fmt.Sprintf(" Search: %s", a.searchQuery))
	a.statusBarRight.SetText(fmt.Sprintf("%d of %d ", matched, total))
}
