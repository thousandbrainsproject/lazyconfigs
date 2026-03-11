package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func init() {
	tview.Borders.TopLeft = '╭'
	tview.Borders.TopRight = '╮'
	tview.Borders.BottomLeft = '╰'
	tview.Borders.BottomRight = '╯'
}

type App struct {
	app             *tview.Application
	pages           *tview.Pages
	panels          []tview.Primitive // Only [builderPanel, variantsPanel]
	currentPanelIdx int

	builderPanel   *tview.List
	variantsPanel  *tview.List
	viewerPanel    *tview.TextView
	statusBarLeft  *tview.TextView
	statusBarRight *tview.TextView

	rootNodes    []*TreeNode
	flatItems    []*TreeNode
	confDir      string

	selectedBuilderNode *TreeNode
	variantFiles        []string
	variantDir          string

	resolvedMode bool

	diffMode     bool
	diffFromIdx  int
	diffFromFile string

	helpOpen        bool
	confirmOpen     bool
	renameOpen      bool
	pendingDeleteIdx int
}

func newApp() *App {
	builderPanel := tview.NewList().
		ShowSecondaryText(false).
		SetHighlightFullLine(true).
		SetSelectedBackgroundColor(tcell.ColorDefault).
		SetSelectedTextColor(tcell.ColorDefault)
	builderPanel.SetBorder(true).
		SetTitle(" [1] Builder ").
		SetTitleAlign(tview.AlignLeft).
		SetBorderColor(tcell.ColorDefault)

	variantsPanel := tview.NewList().
		ShowSecondaryText(false).
		SetHighlightFullLine(true).
		SetSelectedBackgroundColor(tcell.ColorDefault).
		SetSelectedTextColor(tcell.ColorDefault)
	variantsPanel.SetBorder(true).
		SetTitle(" [2] Variants ").
		SetTitleAlign(tview.AlignLeft).
		SetBorderColor(tcell.ColorDefault)

	viewerPanel := tview.NewTextView().
		SetDynamicColors(true).
		SetWrap(true)
	viewerPanel.SetBorder(true).
		SetTitle(" Viewer ").
		SetTitleAlign(tview.AlignLeft).
		SetBorderColor(tcell.ColorDefault)

	statusBarLeft := tview.NewTextView().
		SetDynamicColors(true)
	statusBarLeft.SetBorder(false)

	statusBarRight := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignRight)
	statusBarRight.SetBorder(false)
	statusBarRight.SetText("[blue::b]Thousand Brains Project[-:-:-] 0.0.1 ")

	exe, _ := os.Executable()
	exeDir := filepath.Dir(exe)
	confDir, _ := filepath.Abs(filepath.Join(exeDir, "..", "conf"))

	a := &App{
		app:           tview.NewApplication(),
		builderPanel:   builderPanel,
		variantsPanel:  variantsPanel,
		viewerPanel:    viewerPanel,
		statusBarLeft:  statusBarLeft,
		statusBarRight: statusBarRight,
		panels:         []tview.Primitive{builderPanel, variantsPanel},
		confDir:        confDir,
	}

	builderPanel.SetChangedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
		if index >= 0 && index < len(a.flatItems) {
			a.refreshBuilderSelection(index)
			node := a.flatItems[index]
			a.selectedBuilderNode = node
			a.populateVariants(node)
			if a.currentPanelIdx == 0 {
				a.updateViewer(node)
			}
		}
	})

	variantsPanel.SetChangedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
		a.refreshVariantsSelection(index)
		if a.currentPanelIdx == 1 {
			if a.diffMode && index >= 0 && index < len(a.variantFiles) {
				currentPath := filepath.Join(a.variantDir, a.variantFiles[index]+".yaml")
				a.updateViewerDiff(a.diffFromFile, currentPath)
			} else if index >= 0 && index < len(a.variantFiles) {
				variantPath := filepath.Join(a.variantDir, a.variantFiles[index]+".yaml")
				a.updateViewer(&TreeNode{FilePath: variantPath})
			}
		}
	})

	// Build layout
	leftFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(a.builderPanel, 0, 1, true).
		AddItem(a.variantsPanel, 0, 1, false)

	mainFlex := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(leftFlex, 0, 2, true).
		AddItem(a.viewerPanel, 0, 3, false)

	statusFlex := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(a.statusBarLeft, 0, 3, false).
		AddItem(a.statusBarRight, 0, 1, false)

	rootFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(mainFlex, 0, 1, true).
		AddItem(statusFlex, 1, 0, false)

	a.pages = tview.NewPages().
		AddPage("main", rootFlex, true, true)

	a.refreshAll()
	a.setupKeybindings()
	a.updateBorderColors()
	a.updateStatusBar()

	return a
}

func (a *App) refreshAll() {
	// Save expanded state before rebuilding
	expanded := collectExpanded(a.rootNodes)

	// Load tree from hardcoded experiment
	expPath := filepath.Join(a.confDir, "experiment", "base_config_10distinctobj_dist_agent.yaml")
	roots, err := buildTree(expPath, a.confDir)
	if err != nil {
		a.builderPanel.Clear()
		a.builderPanel.AddItem(fmt.Sprintf("[red]Error: %s[-]", err.Error()), "", 0, nil)
	} else {
		restoreExpanded(roots, expanded)
		a.rootNodes = roots
		a.rebuildBuilderList()
	}

	// Populate variants from selected builder node (or clear)
	if a.selectedBuilderNode != nil {
		a.populateVariants(a.selectedBuilderNode)
	} else {
		a.variantsPanel.Clear()
		a.variantFiles = nil
		a.variantDir = ""
	}

	// Viewer placeholder only if nothing is selected
	if len(a.flatItems) == 0 {
		a.viewerPanel.SetText("[darkgray]Select a config to preview[-]")
	}
}

func (a *App) rebuildBuilderList() {
	currentIdx := a.builderPanel.GetCurrentItem()
	a.flatItems = flattenTree(a.rootNodes)
	a.builderPanel.Clear()
	for i, node := range a.flatItems {
		a.builderPanel.AddItem(renderItem(node, i == currentIdx), "", 0, nil)
	}
	if currentIdx >= len(a.flatItems) {
		currentIdx = len(a.flatItems) - 1
	}
	if currentIdx >= 0 {
		a.builderPanel.SetCurrentItem(currentIdx)
	}
}

// refreshBuilderSelection re-renders all builder items to update the selection marker.
func (a *App) refreshBuilderSelection(selectedIdx int) {
	count := a.builderPanel.GetItemCount()
	for i, node := range a.flatItems {
		if i >= count {
			break
		}
		a.builderPanel.SetItemText(i, renderItem(node, i == selectedIdx), "")
	}
}

// refreshVariantsSelection re-renders all variant items to update the selection marker.
func (a *App) refreshVariantsSelection(selectedIdx int) {
	activeValue := ""
	if a.selectedBuilderNode != nil {
		activeValue = a.selectedBuilderNode.Value
	}
	for i, name := range a.variantFiles {
		if i >= a.variantsPanel.GetItemCount() {
			break
		}
		isDiffFrom := a.diffMode && i == a.diffFromIdx
		a.variantsPanel.SetItemText(i, renderVariantItem(name, i == selectedIdx, name == activeValue, isDiffFrom), "")
	}
}

// populateVariants reads variant files from the node's package directory
// and populates the variants panel.
func (a *App) populateVariants(node *TreeNode) {
	a.variantsPanel.Clear()
	a.variantFiles = nil
	a.variantDir = ""

	if node == nil {
		return
	}

	dir := node.packageDir(a.confDir)
	variants, err := listVariants(dir)
	if err != nil {
		a.variantsPanel.AddItem(fmt.Sprintf("[red]Error: %s[-]", err.Error()), "", 0, nil)
		return
	}

	a.variantDir = dir
	a.variantFiles = variants

	activeIdx := -1
	for i, name := range variants {
		isActive := name == node.Value
		a.variantsPanel.AddItem(renderVariantItem(name, i == 0, isActive, false), "", 0, nil)
		if isActive {
			activeIdx = i
		}
	}

	// Auto-scroll to active variant
	if activeIdx >= 0 {
		a.variantsPanel.SetCurrentItem(activeIdx)
	}
}

// selectVariant selects the highlighted variant, updates the YAML config file,
// and refreshes the tree.
func (a *App) selectVariant() {
	idx := a.variantsPanel.GetCurrentItem()
	if idx < 0 || idx >= len(a.variantFiles) || a.selectedBuilderNode == nil {
		return
	}

	newValue := a.variantFiles[idx]
	node := a.selectedBuilderNode

	if err := updateDefaultValue(node.SourceFilePath, node.RawKey, newValue); err != nil {
		a.viewerPanel.SetText(fmt.Sprintf("[red]Error updating config: %s[-]", err.Error()))
		return
	}

	// Save identifiers for cursor restoration
	sourceFile := node.SourceFilePath
	key := node.Key

	a.refreshAll()

	// Restore builder cursor to the same node
	restoredIdx := a.findBuilderNodeIndex(sourceFile, key)
	if restoredIdx >= 0 {
		a.builderPanel.SetCurrentItem(restoredIdx)
		a.selectedBuilderNode = a.flatItems[restoredIdx]
		a.populateVariants(a.selectedBuilderNode)
	}
}

// findBuilderNodeIndex finds the index of a node in flatItems by SourceFilePath and Key.
func (a *App) findBuilderNodeIndex(sourceFile, key string) int {
	for i, node := range a.flatItems {
		if node.SourceFilePath == sourceFile && node.Key == key {
			return i
		}
	}
	return -1
}

// refreshCurrentViewer re-renders the viewer based on the current panel and selection.
func (a *App) refreshCurrentViewer() {
	switch a.currentPanelIdx {
	case 0:
		idx := a.builderPanel.GetCurrentItem()
		if idx >= 0 && idx < len(a.flatItems) {
			a.updateViewer(a.flatItems[idx])
		}
	case 1:
		variantIdx := a.variantsPanel.GetCurrentItem()
		if variantIdx >= 0 && variantIdx < len(a.variantFiles) {
			if a.diffMode {
				currentPath := filepath.Join(a.variantDir, a.variantFiles[variantIdx]+".yaml")
				a.updateViewerDiff(a.diffFromFile, currentPath)
			} else {
				variantPath := filepath.Join(a.variantDir, a.variantFiles[variantIdx]+".yaml")
				a.updateViewer(&TreeNode{FilePath: variantPath})
			}
		}
	}
}

func (a *App) enterDiffMode() {
	idx := a.variantsPanel.GetCurrentItem()
	if idx < 0 || idx >= len(a.variantFiles) {
		return
	}
	a.diffMode = true
	a.diffFromIdx = idx
	a.diffFromFile = filepath.Join(a.variantDir, a.variantFiles[idx]+".yaml")
	a.variantsPanel.SetTitle(" [2] Variants [yellow::b]diff[-:-:-] ")
	a.refreshVariantsSelection(idx)
	a.viewerPanel.SetText("[darkgray]Navigate to another variant to see diff[-]")
	a.updateStatusBar()
}

func (a *App) exitDiffMode() {
	a.diffMode = false
	a.diffFromIdx = -1
	a.diffFromFile = ""
	a.variantsPanel.SetTitle(" [2] Variants ")
	a.refreshVariantsSelection(a.variantsPanel.GetCurrentItem())
	a.refreshCurrentViewer()
	a.updateStatusBar()
}

func (a *App) toggleExpand() {
	idx := a.builderPanel.GetCurrentItem()
	if idx < 0 || idx >= len(a.flatItems) {
		return
	}
	node := a.flatItems[idx]
	if node.IsLeaf {
		return
	}
	node.Expanded = !node.Expanded
	a.rebuildBuilderList()
	// Find the toggled node's new index and restore cursor
	for i, n := range a.flatItems {
		if n == node {
			a.builderPanel.SetCurrentItem(i)
			break
		}
	}
}

func (a *App) setupKeybindings() {
	a.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// MODAL PRIORITY CHAIN — must be first
		// Rename modal: passthrough so InputField receives characters
		if a.renameOpen {
			if event.Key() == tcell.KeyEsc {
				a.closeRename()
				return nil
			}
			return event
		}

		// Confirm modal: only handle y/n/Esc/q
		if a.confirmOpen {
			switch event.Key() {
			case tcell.KeyRune:
				switch event.Rune() {
				case 'y', 'Y':
					a.executeDelete()
					return nil
				case 'n', 'N', 'q':
					a.closeConfirm()
					return nil
				}
			case tcell.KeyEsc:
				a.closeConfirm()
				return nil
			}
			return nil
		}

		if a.helpOpen {
			if event.Key() == tcell.KeyEsc {
				a.closeHelp()
				return nil
			}
			return event
		}

		// MAIN KEYBINDINGS — only when no modal open
		switch event.Key() {
		case tcell.KeyRune:
			switch event.Rune() {
			case 'q':
				a.app.Stop()
				return nil
			case '1':
				a.focusPanel(0)
				return nil
			case '2':
				a.focusPanel(1)
				return nil
			case 'h':
				a.prevPanel()
				return nil
			case 'l':
				a.nextPanel()
				return nil
			case 'j':
				a.cursorDown()
				return nil
			case 'k':
				a.cursorUp()
				return nil
			case 'J':
				a.scrollViewerDown()
				return nil
			case 'K':
				a.scrollViewerUp()
				return nil
			case '?':
				a.showHelp()
				return nil
			case ' ':
				if a.currentPanelIdx == 1 && !a.diffMode {
					a.selectVariant()
					return nil
				}
			case 'd':
				if a.currentPanelIdx == 1 && !a.diffMode {
					a.duplicateVariant()
					return nil
				}
			case 'r':
				if a.currentPanelIdx == 1 && !a.diffMode {
					a.showRenameModal()
					return nil
				}
			case 'D':
				if a.currentPanelIdx == 1 && !a.diffMode {
					a.showDeleteConfirm()
					return nil
				}
			case 'e':
				if a.currentPanelIdx == 1 && !a.diffMode {
					a.editVariantInEditor()
					return nil
				}
			case 'w':
				if a.currentPanelIdx == 1 {
					a.enterDiffMode()
					return nil
				}
			case 'v':
				a.resolvedMode = !a.resolvedMode
				a.refreshCurrentViewer()
				return nil
			}
		case tcell.KeyEnter:
			if a.currentPanelIdx == 0 {
				a.toggleExpand()
				return nil
			}
		case tcell.KeyTab:
			a.nextPanel()
			return nil
		case tcell.KeyBacktab:
			a.prevPanel()
			return nil
		case tcell.KeyEsc:
			if a.diffMode {
				a.exitDiffMode()
				return nil
			}
			a.app.Stop()
			return nil
		}
		return event
	})
}

func (a *App) focusPanel(idx int) {
	if idx < 0 || idx >= len(a.panels) {
		return
	}
	if a.diffMode && idx != 1 {
		a.exitDiffMode()
	}
	a.currentPanelIdx = idx
	a.app.SetFocus(a.panels[a.currentPanelIdx])
	a.updateBorderColors()
	a.updateStatusBar()

	// Switch viewer content based on focused panel
	a.refreshCurrentViewer()
}

func (a *App) nextPanel() {
	a.focusPanel((a.currentPanelIdx + 1) % len(a.panels))
}

func (a *App) prevPanel() {
	a.focusPanel((a.currentPanelIdx - 1 + len(a.panels)) % len(a.panels))
}

func (a *App) updateBorderColors() {
	for i, panel := range a.panels {
		list := panel.(*tview.List)
		if i == a.currentPanelIdx {
			list.SetBorderColor(tcell.ColorGreen)
		} else {
			list.SetBorderColor(tcell.ColorDefault)
		}
	}
}

func (a *App) cursorDown() {
	if list, ok := a.panels[a.currentPanelIdx].(*tview.List); ok {
		current := list.GetCurrentItem()
		if current < list.GetItemCount()-1 {
			list.SetCurrentItem(current + 1)
		}
	}
}

func (a *App) cursorUp() {
	if list, ok := a.panels[a.currentPanelIdx].(*tview.List); ok {
		current := list.GetCurrentItem()
		if current > 0 {
			list.SetCurrentItem(current - 1)
		}
	}
}

func (a *App) scrollViewerDown() {
	row, col := a.viewerPanel.GetScrollOffset()
	a.viewerPanel.ScrollTo(row+1, col)
}

func (a *App) scrollViewerUp() {
	row, col := a.viewerPanel.GetScrollOffset()
	if row > 0 {
		a.viewerPanel.ScrollTo(row-1, col)
	}
}

func (a *App) updateStatusBar() {
	switch a.currentPanelIdx {
	case 0:
		a.statusBarLeft.SetText(" Navigate: j/k | Expand: Enter | Panels: h/l | Scroll: J/K | Resolve: v | Help: ? | Quit: q")
	case 1:
		if a.diffMode {
			a.statusBarLeft.SetText(" Navigate: j/k | Scroll: J/K | Resolve: v | Exit diff: Esc | Help: ? | Quit: q")
		} else {
			a.statusBarLeft.SetText(" Navigate: j/k | Select: Space | Dup: d | Rename: r | Del: D | Edit: e | Resolve: v | Diff: w | Help: ?")
		}
	default:
		a.statusBarLeft.SetText(fmt.Sprintf(" Panel %d", a.currentPanelIdx))
	}
}

func modal(content tview.Primitive, width, height int) tview.Primitive {
	return tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(content, height, 0, true).
			AddItem(nil, 0, 1, false), width, 0, true).
		AddItem(nil, 0, 1, false)
}

var panelHelpTexts = map[int]string{
	0: `[yellow::b]Builder — Help[-:-:-]

[green]Navigation:[-]
  j / k         Move cursor up/down
  Enter         Expand/collapse node
  1             Jump to this panel
  h / l         Switch panels
  Tab / S-Tab   Cycle panels

[green]Viewer:[-]
  J / K         Scroll viewer
  v             Toggle resolved view

[green]General:[-]
  ?             This help
  Esc           Close overlay
  q             Quit

[darkgray]Press Escape to close[-]`,

	1: `[yellow::b]Variants — Help[-:-:-]

[green]Navigation:[-]
  j / k         Move cursor up/down
  2             Jump to this panel
  h / l         Switch panels
  Tab / S-Tab   Cycle panels

[green]Actions:[-]
  Space         Select this variant
  d             Duplicate variant
  r             Rename variant
  D             Delete variant (confirm)
  e             Edit in $EDITOR
  v             Toggle resolved view
  w             Diff from this variant

[green]Viewer:[-]
  J / K         Scroll viewer

[green]General:[-]
  ?             This help
  Esc           Exit diff / Close overlay
  q             Quit

[darkgray]Press Escape to close[-]`,
}

var panelHelpTitles = map[int]string{
	0: " Builder Help ",
	1: " Variants Help ",
}

func (a *App) showHelp() {
	a.helpOpen = true

	helpView := tview.NewTextView().
		SetDynamicColors(true).
		SetText(panelHelpTexts[a.currentPanelIdx])
	helpView.SetBorder(true).
		SetTitle(panelHelpTitles[a.currentPanelIdx]).
		SetBorderColor(tcell.ColorGreen)

	a.pages.AddPage("help", modal(helpView, 55, 22), true, true)
	a.app.SetFocus(helpView)
}

func (a *App) closeHelp() {
	a.helpOpen = false
	a.pages.RemovePage("help")
	a.app.SetFocus(a.panels[a.currentPanelIdx])
	a.updateBorderColors()
}

// duplicateVariant copies the selected variant file with a _copy suffix.
func (a *App) duplicateVariant() {
	idx := a.variantsPanel.GetCurrentItem()
	if idx < 0 || idx >= len(a.variantFiles) {
		return
	}

	name := a.variantFiles[idx]
	src := filepath.Join(a.variantDir, name+".yaml")

	dst := filepath.Join(a.variantDir, name+"_copy.yaml")
	suffix := 2
	for {
		if _, err := os.Stat(dst); os.IsNotExist(err) {
			break
		}
		dst = filepath.Join(a.variantDir, fmt.Sprintf("%s_copy%d.yaml", name, suffix))
		suffix++
	}

	data, err := os.ReadFile(src)
	if err != nil {
		a.viewerPanel.SetText(fmt.Sprintf("[red]Error reading file: %s[-]", err.Error()))
		return
	}
	if err := os.WriteFile(dst, data, 0644); err != nil {
		a.viewerPanel.SetText(fmt.Sprintf("[red]Error writing file: %s[-]", err.Error()))
		return
	}

	a.populateVariants(a.selectedBuilderNode)
}

// showDeleteConfirm shows a confirmation modal for deleting the selected variant.
func (a *App) showDeleteConfirm() {
	idx := a.variantsPanel.GetCurrentItem()
	if idx < 0 || idx >= len(a.variantFiles) {
		return
	}

	a.pendingDeleteIdx = idx
	a.confirmOpen = true

	name := a.variantFiles[idx]
	confirmView := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter).
		SetText(fmt.Sprintf("\nDelete [yellow]%s.yaml[-]? [green](y/n)[-]", name))
	confirmView.SetBorder(true).
		SetTitle(" Confirm Delete ").
		SetBorderColor(tcell.ColorRed)

	// Width: "Delete " + name + ".yaml? (y/n)" + border padding
	w := len(name) + 28
	if w < 45 {
		w = 45
	}
	a.pages.AddPage("confirm", modal(confirmView, w, 5), true, true)
	a.app.SetFocus(confirmView)
}

// executeDelete performs the actual file deletion after confirmation.
func (a *App) executeDelete() {
	idx := a.pendingDeleteIdx
	if idx < 0 || idx >= len(a.variantFiles) {
		a.closeConfirm()
		return
	}

	name := a.variantFiles[idx]
	filePath := filepath.Join(a.variantDir, name+".yaml")

	if err := os.Remove(filePath); err != nil {
		a.viewerPanel.SetText(fmt.Sprintf("[red]Error deleting file: %s[-]", err.Error()))
		a.closeConfirm()
		return
	}

	// If the deleted variant was the active one, set value to "??"
	if a.selectedBuilderNode != nil && a.selectedBuilderNode.Value == name {
		_ = updateDefaultValue(a.selectedBuilderNode.SourceFilePath, a.selectedBuilderNode.RawKey, "??")

		sourceFile := a.selectedBuilderNode.SourceFilePath
		key := a.selectedBuilderNode.Key
		a.refreshAll()
		restoredIdx := a.findBuilderNodeIndex(sourceFile, key)
		if restoredIdx >= 0 {
			a.builderPanel.SetCurrentItem(restoredIdx)
			a.selectedBuilderNode = a.flatItems[restoredIdx]
		}
	}

	a.closeConfirm()
	a.populateVariants(a.selectedBuilderNode)
}

// closeConfirm closes the delete confirmation modal.
func (a *App) closeConfirm() {
	a.confirmOpen = false
	a.pages.RemovePage("confirm")
	a.app.SetFocus(a.panels[a.currentPanelIdx])
	a.updateBorderColors()
}

// showRenameModal shows an input modal for renaming the selected variant.
func (a *App) showRenameModal() {
	idx := a.variantsPanel.GetCurrentItem()
	if idx < 0 || idx >= len(a.variantFiles) {
		return
	}

	a.renameOpen = true
	name := a.variantFiles[idx]

	inputField := tview.NewInputField().
		SetLabel("New name: ").
		SetText(name).
		SetFieldWidth(len(name) + 10).
		SetFieldBackgroundColor(tcell.ColorDefault)
	inputField.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			a.executeRename(idx, inputField.GetText())
		} else {
			a.closeRename()
		}
	})

	inputField.SetBorder(true).
		SetTitle(" Rename Variant ").
		SetBorderColor(tcell.ColorGreen)

	// Width: "New name: " + field + border padding
	w := len(name) + 24
	if w < 50 {
		w = 50
	}
	a.pages.AddPage("rename", modal(inputField, w, 3), true, true)
	a.app.SetFocus(inputField)
}

// executeRename renames the variant file and updates the config if it was the active variant.
func (a *App) executeRename(idx int, newName string) {
	if idx < 0 || idx >= len(a.variantFiles) || newName == "" {
		a.closeRename()
		return
	}

	oldName := a.variantFiles[idx]
	if oldName == newName {
		a.closeRename()
		return
	}

	oldPath := filepath.Join(a.variantDir, oldName+".yaml")
	newPath := filepath.Join(a.variantDir, newName+".yaml")

	if err := os.Rename(oldPath, newPath); err != nil {
		a.viewerPanel.SetText(fmt.Sprintf("[red]Error renaming file: %s[-]", err.Error()))
		a.closeRename()
		return
	}

	// If the renamed variant was the active one, update the config
	if a.selectedBuilderNode != nil && a.selectedBuilderNode.Value == oldName {
		_ = updateDefaultValue(a.selectedBuilderNode.SourceFilePath, a.selectedBuilderNode.RawKey, newName)

		sourceFile := a.selectedBuilderNode.SourceFilePath
		key := a.selectedBuilderNode.Key
		a.refreshAll()
		restoredIdx := a.findBuilderNodeIndex(sourceFile, key)
		if restoredIdx >= 0 {
			a.builderPanel.SetCurrentItem(restoredIdx)
			a.selectedBuilderNode = a.flatItems[restoredIdx]
		}
	}

	a.closeRename()
	a.populateVariants(a.selectedBuilderNode)
}

// closeRename closes the rename input modal.
func (a *App) closeRename() {
	a.renameOpen = false
	a.pages.RemovePage("rename")
	a.app.SetFocus(a.panels[a.currentPanelIdx])
	a.updateBorderColors()
}

// editVariantInEditor opens the selected variant file in $EDITOR.
func (a *App) editVariantInEditor() {
	idx := a.variantsPanel.GetCurrentItem()
	if idx < 0 || idx >= len(a.variantFiles) {
		return
	}

	filePath := filepath.Join(a.variantDir, a.variantFiles[idx]+".yaml")
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}

	a.app.Suspend(func() {
		cmd := exec.Command(editor, filePath)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		_ = cmd.Run()
	})
	a.refreshAll()
	a.populateVariants(a.selectedBuilderNode)
}

func main() {
	a := newApp()
	a.app.SetRoot(a.pages, true).EnableMouse(false)
	if err := a.app.Run(); err != nil {
		panic(err)
	}
}
