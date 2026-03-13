// ABOUTME: Main application struct and logic for the lazyconfigs TUI.
// ABOUTME: Orchestrates panels, keybindings, modals, and variant operations.
package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"lazyconfigs/internal/config"
	"lazyconfigs/internal/hydra"
	"lazyconfigs/internal/ui"
	"lazyconfigs/internal/version"
)

// App holds all state for the lazyconfigs TUI application.
type App struct {
	app             *tview.Application
	pages           *tview.Pages
	panels          []tview.Primitive // Only [builderPanel, variantsPanel]
	currentPanelIdx int

	cfg      config.Config
	theme    config.ThemeColors
	bindings config.CompiledBindings

	builderPanel   *tview.List
	variantsPanel  *tview.List
	viewerPanel    *tview.TextView
	statusBarLeft  *tview.TextView
	statusBarRight *tview.TextView

	rootNodes []*hydra.TreeNode
	flatItems []*hydra.TreeNode
	confDir   string

	visibleBuilderItems []*hydra.TreeNode
	visibleVariantFiles []string

	selectedBuilderNode *hydra.TreeNode
	variantFiles        []string
	variantDir          string

	resolvedMode bool

	diffMode     bool
	diffFromIdx  int
	diffFromFile string

	helpOpen             bool
	confirmOpen          bool
	refsOpen             bool
	renameOpen           bool
	pendingDeleteIdx     int
	pendingConfirmAction config.ConfirmAction
	pendingReassignValue string

	expPath string

	searchMode  bool
	searchQuery string
	searchPanel int
}

// New creates and initializes a new App instance.
func New() *App {
	tview.Borders.TopLeft = '╭'
	tview.Borders.TopRight = '╮'
	tview.Borders.BottomLeft = '╰'
	tview.Borders.BottomRight = '╯'

	builderPanel := tview.NewList().
		ShowSecondaryText(false).
		SetHighlightFullLine(true).
		SetSelectedBackgroundColor(tcell.ColorDefault).
		SetSelectedTextColor(tcell.ColorDefault)
	cfg := config.LoadConfig()
	theme := config.CompileTheme(cfg.Colors)
	bindings := config.CompileBindings(cfg.Keybindings)

	var confDir string
	if cfg.ConfDir != "" {
		confDir = cfg.ConfDir
	} else {
		var err error
		confDir, err = config.FindGitRoot()
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: %v, falling back to current directory\n", err)
			confDir, _ = os.Getwd()
		}
	}

	builderPanel.SetBorder(true).
		SetTitle(" [1] Builder ").
		SetTitleAlign(tview.AlignLeft).
		SetBorderColor(theme.BorderUnfocused)

	variantsPanel := tview.NewList().
		ShowSecondaryText(false).
		SetHighlightFullLine(true).
		SetSelectedBackgroundColor(tcell.ColorDefault).
		SetSelectedTextColor(tcell.ColorDefault)
	variantsPanel.SetBorder(true).
		SetTitle(" [2] Variants ").
		SetTitleAlign(tview.AlignLeft).
		SetBorderColor(theme.BorderUnfocused)

	viewerPanel := tview.NewTextView().
		SetDynamicColors(true).
		SetWrap(true)
	viewerPanel.SetBorder(true).
		SetTitle(" Viewer ").
		SetTitleAlign(tview.AlignLeft).
		SetBorderColor(theme.BorderUnfocused)

	statusBarLeft := tview.NewTextView().
		SetDynamicColors(true)
	statusBarLeft.SetBorder(false)

	statusBarRight := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignRight)
	statusBarRight.SetBorder(false)
	statusBarRight.SetText("[blue::b]Thousand Brains Project[-:-:-] " + version.Version + " ")

	a := &App{
		app:            tview.NewApplication(),
		cfg:            cfg,
		theme:          theme,
		bindings:       bindings,
		builderPanel:   builderPanel,
		variantsPanel:  variantsPanel,
		viewerPanel:    viewerPanel,
		statusBarLeft:  statusBarLeft,
		statusBarRight: statusBarRight,
		panels:         []tview.Primitive{builderPanel, variantsPanel},
		confDir:        confDir,
	}

	builderPanel.SetChangedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
		if index >= 0 && index < len(a.visibleBuilderItems) {
			a.refreshBuilderSelection(index)
			node := a.visibleBuilderItems[index]
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
			if a.diffMode && index >= 0 && index < len(a.visibleVariantFiles) {
				currentPath := filepath.Join(a.variantDir, a.visibleVariantFiles[index]+".yaml")
				a.updateViewerDiff(a.diffFromFile, currentPath)
			} else if index >= 0 && index < len(a.visibleVariantFiles) {
				variantPath := filepath.Join(a.variantDir, a.visibleVariantFiles[index]+".yaml")
				a.updateViewer(&hydra.TreeNode{FilePath: variantPath})
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

	// Trigger initial selection so variants and viewer populate on startup.
	if len(a.flatItems) > 0 {
		a.selectedBuilderNode = a.flatItems[0]
		a.populateVariants(a.selectedBuilderNode)
		a.updateViewer(a.selectedBuilderNode)
	}

	a.setupKeybindings()
	a.updateBorderColors()
	a.updateStatusBar()

	return a
}

// Run starts the tview application event loop.
func (a *App) Run() error {
	a.app.SetRoot(a.pages, true).EnableMouse(false)
	return a.app.Run()
}

func (a *App) refreshAll() {
	// Save expanded state before rebuilding
	expanded := hydra.CollectExpanded(a.rootNodes)

	// Load tree from root config
	expPath := filepath.Join(a.confDir, "experiment.yaml")
	absExpPath, err := filepath.Abs(expPath)
	if err == nil {
		a.expPath = absExpPath
	}
	roots, err := hydra.BuildTree(expPath, a.confDir)
	if err != nil {
		a.builderPanel.Clear()
		a.builderPanel.AddItem(fmt.Sprintf("[red]Error: %s[-]", err.Error()), "", 0, nil)
	} else {
		hydra.RestoreExpanded(roots, expanded)
		a.rootNodes = roots
		a.rebuildBuilderList()
	}

	// Populate variants from selected builder node (or clear)
	if a.selectedBuilderNode != nil {
		a.populateVariants(a.selectedBuilderNode)
	} else {
		a.variantsPanel.Clear()
		a.variantFiles = nil
		a.visibleVariantFiles = nil
		a.variantDir = ""
	}

	// Viewer placeholder only if nothing is selected
	if len(a.flatItems) == 0 {
		a.viewerPanel.SetText("[darkgray]Select a config to preview[-]")
	}
}

func (a *App) rebuildBuilderList() {
	currentIdx := a.builderPanel.GetCurrentItem()
	a.flatItems = hydra.FlattenTree(a.rootNodes)
	// Update visibleBuilderItems before modifying the panel so that
	// SetChangedFunc callbacks (triggered by AddItem/SetCurrentItem)
	// see the correct item list rather than stale data.
	a.visibleBuilderItems = a.flatItems
	a.builderPanel.Clear()
	for i, node := range a.flatItems {
		a.builderPanel.AddItem(ui.RenderItem(node, i == currentIdx, a.theme), "", 0, nil)
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
	for i, node := range a.visibleBuilderItems {
		if i >= count {
			break
		}
		a.builderPanel.SetItemText(i, ui.RenderItem(node, i == selectedIdx, a.theme), "")
	}
}

// refreshVariantsSelection re-renders all variant items to update the selection marker.
func (a *App) refreshVariantsSelection(selectedIdx int) {
	activeValue := ""
	if a.selectedBuilderNode != nil {
		activeValue = a.selectedBuilderNode.Value
	}
	for i, name := range a.visibleVariantFiles {
		if i >= a.variantsPanel.GetItemCount() {
			break
		}
		isDiffFrom := a.diffMode && i == a.diffFromIdx
		a.variantsPanel.SetItemText(i, ui.RenderVariantItem(name, i == selectedIdx, name == activeValue, isDiffFrom, a.theme), "")
	}
}

// populateVariants reads variant files from the node's package directory
// and populates the variants panel.
func (a *App) populateVariants(node *hydra.TreeNode) {
	// Exit search if active on the variants panel — the variant list is about
	// to be replaced, so filtered state would become stale.
	if a.searchMode && a.searchPanel == 1 {
		a.searchMode = false
		a.searchQuery = ""
		a.updateStatusBar()
		a.statusBarRight.SetText("[blue::b]Thousand Brains Project[-:-:-] " + version.Version + " ")
	}

	// Reset diff state — the variant list is about to change, so diffFromIdx
	// and diffFromFile would become stale.
	if a.diffMode {
		a.diffMode = false
		a.diffFromIdx = -1
		a.diffFromFile = ""
		a.variantsPanel.SetTitle(" [2] Variants ")
		a.updateStatusBar()
	}

	a.variantsPanel.Clear()
	a.variantFiles = nil
	a.visibleVariantFiles = nil
	a.variantDir = ""

	if node == nil {
		return
	}

	dir := node.PackageDir(a.confDir)
	variants, err := hydra.ListVariants(dir)
	if err != nil {
		a.variantsPanel.AddItem(fmt.Sprintf("[red]Error: %s[-]", err.Error()), "", 0, nil)
		return
	}

	a.variantDir = dir
	a.variantFiles = variants

	activeIdx := -1
	for i, name := range variants {
		if name == node.Value {
			activeIdx = i
		}
	}

	cursorIdx := activeIdx
	if cursorIdx < 0 {
		cursorIdx = 0
	}
	for i, name := range variants {
		isActive := name == node.Value
		a.variantsPanel.AddItem(ui.RenderVariantItem(name, i == cursorIdx, isActive, false, a.theme), "", 0, nil)
	}

	// Auto-scroll to active variant
	if activeIdx >= 0 {
		a.variantsPanel.SetCurrentItem(activeIdx)
	}
	a.visibleVariantFiles = a.variantFiles
}

// selectVariant selects the highlighted variant, updates the YAML config file,
// and refreshes the tree.
func (a *App) selectVariant() {
	idx := a.variantsPanel.GetCurrentItem()
	if idx < 0 || idx >= len(a.visibleVariantFiles) || a.selectedBuilderNode == nil {
		return
	}

	newValue := a.visibleVariantFiles[idx]
	node := a.selectedBuilderNode

	if newValue == node.Value {
		return
	}

	// Top-level nodes modify the experiment file directly.
	if node.SourceFilePath == a.expPath {
		a.executeReassign(newValue)
		return
	}

	// Deep node: reassignment modifies a shared config file.
	if !a.cfg.Warnings.ShouldWarn(config.ConfirmReassign) {
		a.executeReassign(newValue)
		return
	}

	// Check if other experiments also use this config file.
	allRefs, err := hydra.FindFileReferences(a.confDir, node.SourceFilePath)
	if err != nil {
		a.viewerPanel.SetText(fmt.Sprintf("[red]Error finding references: %s[-]", err.Error()))
		return
	}
	currentExp := a.assignedExperimentName()
	var otherRefs []string
	for _, r := range allRefs {
		if r != currentExp {
			otherRefs = append(otherRefs, r)
		}
	}
	if len(otherRefs) == 0 {
		a.executeReassign(newValue)
		return
	}

	a.showReassignConfirm(node.Value, newValue, otherRefs)
}

// executeReassign performs the actual variant reassignment.
func (a *App) executeReassign(newValue string) {
	node := a.selectedBuilderNode
	if node == nil {
		return
	}

	if err := hydra.UpdateDefaultValue(node.SourceFilePath, node.RawKey, newValue); err != nil {
		a.viewerPanel.SetText(fmt.Sprintf("[red]Error updating config: %s[-]", err.Error()))
		return
	}

	sourceFile := node.SourceFilePath
	key := node.Key

	a.refreshAll()

	restoredIdx := a.findBuilderNodeIndex(sourceFile, key)
	if restoredIdx >= 0 {
		a.builderPanel.SetCurrentItem(restoredIdx)
		a.selectedBuilderNode = a.flatItems[restoredIdx]
		a.populateVariants(a.selectedBuilderNode)
	}
}

// assignedExperimentName returns the currently assigned experiment variant
// name by inspecting the root tree nodes, or empty string if unassigned.
func (a *App) assignedExperimentName() string {
	for _, node := range a.rootNodes {
		if node.Key == "experiment" {
			return node.Value
		}
	}
	return ""
}

// showReassignConfirm displays a confirmation modal listing other experiments
// that reference the current variant before allowing reassignment.
func (a *App) showReassignConfirm(currentVariant, newVariant string, otherRefs []string) {
	a.pendingReassignValue = newVariant
	a.pendingConfirmAction = config.ConfirmReassign
	a.showWarningModal(warningModalConfig{
		title:       " Confirm Reassign ",
		borderColor: a.theme.ModalWarningBorder,
		headerText:  fmt.Sprintf("\n[yellow]%s[-] is also used by:\n", currentVariant),
		footerText:  fmt.Sprintf("\nReassign to [yellow]%s[-]? [green](y/n)[-]", newVariant),
		refs:        otherRefs,
	})
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
		if idx >= 0 && idx < len(a.visibleBuilderItems) {
			a.updateViewer(a.visibleBuilderItems[idx])
		}
	case 1:
		variantIdx := a.variantsPanel.GetCurrentItem()
		if variantIdx >= 0 && variantIdx < len(a.visibleVariantFiles) {
			if a.diffMode {
				currentPath := filepath.Join(a.variantDir, a.visibleVariantFiles[variantIdx]+".yaml")
				a.updateViewerDiff(a.diffFromFile, currentPath)
			} else {
				variantPath := filepath.Join(a.variantDir, a.visibleVariantFiles[variantIdx]+".yaml")
				a.updateViewer(&hydra.TreeNode{FilePath: variantPath})
			}
		}
	}
}

func (a *App) enterDiffMode() {
	idx := a.variantsPanel.GetCurrentItem()
	if idx < 0 || idx >= len(a.visibleVariantFiles) {
		return
	}
	a.diffMode = true
	a.diffFromIdx = idx
	a.diffFromFile = filepath.Join(a.variantDir, a.visibleVariantFiles[idx]+".yaml")
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
					switch a.pendingConfirmAction {
					case config.ConfirmDelete:
						a.executeDelete()
					case config.ConfirmReassign:
						a.executeReassign(a.pendingReassignValue)
						a.closeConfirm()
					case config.ConfirmEdit:
						a.closeConfirm()
						a.executeEditVariant()
					case config.ConfirmRename:
						a.closeConfirm()
						a.showRenameInput()
					case config.ConfirmUnassign:
						a.executeUnassign()
						a.closeConfirm()
					}
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

		if a.refsOpen {
			if event.Key() == tcell.KeyEsc {
				a.closeReferences()
				return nil
			}
			return event // allow scrolling with arrow keys
		}

		// Search mode: capture all input for the search query
		if a.searchMode {
			switch event.Key() {
			case tcell.KeyEsc:
				a.exitSearchMode()
				return nil
			case tcell.KeyBackspace, tcell.KeyBackspace2:
				if len(a.searchQuery) > 0 {
					runes := []rune(a.searchQuery)
					a.searchQuery = string(runes[:len(runes)-1])
					a.applySearch()
				}
				return nil
			case tcell.KeyRune:
				a.searchQuery += string(event.Rune())
				a.applySearch()
				return nil
			case tcell.KeyEnter:
				a.exitSearchMode()
				return nil
			case tcell.KeyDown:
				a.searchCursorDown()
				return nil
			case tcell.KeyUp:
				a.searchCursorUp()
				return nil
			}
			return nil
		}

		// MAIN KEYBINDINGS — lookup-based dispatch
		id := config.KeyID{Key: event.Key(), Rune: event.Rune()}
		if id.Key != tcell.KeyRune {
			id.Rune = 0
		}

		// Context-specific bindings first (higher priority)
		switch a.currentPanelIdx {
		case 0:
			if action, ok := a.bindings.BuilderByKey[id]; ok {
				if result := a.dispatchBuilderAction(action); result == nil {
					return nil
				}
			}
		case 1:
			if action, ok := a.bindings.VariantsByKey[id]; ok {
				if result := a.dispatchVariantsAction(action); result == nil {
					return nil
				}
			}
		}

		// General bindings
		if action, ok := a.bindings.GeneralByKey[id]; ok {
			return a.dispatchGeneralAction(action)
		}

		return event
	})
}

// dispatchGeneralAction handles actions from the general keybinding group.
func (a *App) dispatchGeneralAction(action string) *tcell.EventKey {
	switch action {
	case "quit":
		a.app.Stop()
	case "help":
		a.showHelp()
	case "focus_builder":
		a.focusPanel(0)
	case "focus_variants":
		a.focusPanel(1)
	case "panel_next":
		a.nextPanel()
	case "panel_prev":
		a.prevPanel()
	case "panel_cycle_next":
		a.nextPanel()
	case "panel_cycle_prev":
		a.prevPanel()
	case "cursor_down":
		a.cursorDown()
	case "cursor_up":
		a.cursorUp()
	case "scroll_viewer_down":
		a.scrollViewerDown()
	case "scroll_viewer_up":
		a.scrollViewerUp()
	case "toggle_resolved":
		a.resolvedMode = !a.resolvedMode
		a.refreshCurrentViewer()
	case "search":
		if a.currentPanelIdx == 0 || a.currentPanelIdx == 1 {
			a.enterSearchMode()
		}
	case "escape":
		if a.diffMode {
			a.exitDiffMode()
		} else {
			a.app.Stop()
		}
	default:
		return &tcell.EventKey{}
	}
	return nil
}

// dispatchBuilderAction handles actions from the builder keybinding group.
func (a *App) dispatchBuilderAction(action string) *tcell.EventKey {
	switch action {
	case "expand_collapse":
		a.toggleExpand()
	case "unassign":
		a.unassignBuilderNode()
	default:
		return &tcell.EventKey{}
	}
	return nil
}

// dispatchVariantsAction handles actions from the variants keybinding group.
func (a *App) dispatchVariantsAction(action string) *tcell.EventKey {
	if a.diffMode {
		return &tcell.EventKey{}
	}
	switch action {
	case "select":
		a.selectVariant()
	case "duplicate":
		a.duplicateVariant()
	case "rename":
		a.renameVariant()
	case "delete":
		a.showDeleteConfirm()
	case "edit":
		a.editVariantInEditor()
	case "diff":
		a.enterDiffMode()
	case "references":
		a.showReferences()
	default:
		return &tcell.EventKey{}
	}
	return nil
}

func (a *App) focusPanel(idx int) {
	if idx < 0 || idx >= len(a.panels) {
		return
	}
	if a.searchMode {
		a.exitSearchMode()
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
		list, ok := panel.(*tview.List)
		if !ok {
			continue
		}
		if i == a.currentPanelIdx {
			list.SetBorderColor(a.theme.BorderFocused)
		} else {
			list.SetBorderColor(a.theme.BorderUnfocused)
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
	a.statusBarLeft.SetText(config.GenerateStatusBarText(a.currentPanelIdx, a.diffMode, a.bindings))
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

func (a *App) showHelp() {
	a.helpOpen = true

	titles := map[int]string{0: " Builder Help ", 1: " Variants Help "}
	helpView := tview.NewTextView().
		SetDynamicColors(true).
		SetText(config.GenerateHelpText(a.currentPanelIdx, a.bindings))
	helpView.SetBorder(true).
		SetTitle(titles[a.currentPanelIdx]).
		SetBorderColor(a.theme.ModalHelpBorder)

	a.pages.AddPage("help", modal(helpView, 55, 22), true, true)
	a.app.SetFocus(helpView)
}

func (a *App) closeHelp() {
	a.helpOpen = false
	a.pages.RemovePage("help")
	a.app.SetFocus(a.panels[a.currentPanelIdx])
	a.updateBorderColors()
}

func (a *App) showReferences() {
	idx := a.variantsPanel.GetCurrentItem()
	if idx < 0 || idx >= len(a.visibleVariantFiles) || a.variantDir == "" {
		return
	}

	variantName := a.visibleVariantFiles[idx]
	refs, err := hydra.FindVariantReferences(a.confDir, a.variantDir, variantName)

	a.refsOpen = true

	var text string
	if err != nil {
		text = fmt.Sprintf("[red]Error scanning experiments: %s[-]", err.Error())
	} else if len(refs) == 0 {
		text = "[darkgray]No experiments reference this variant.[-]"
	} else {
		var lines []string
		for i, name := range refs {
			lines = append(lines, fmt.Sprintf("  [green]%d.[-] %s", i+1, name))
		}
		text = strings.Join(lines, "\n")
	}

	refsView := tview.NewTextView().
		SetDynamicColors(true).
		SetText(text)
	refsView.SetBorder(true).
		SetTitle(fmt.Sprintf(" References: %s ", variantName)).
		SetBorderColor(a.theme.ModalRefsBorder)

	w := len(variantName) + 18
	for _, ref := range refs {
		if lineW := len(ref) + 8; lineW > w {
			w = lineW
		}
	}
	if w < 50 {
		w = 50
	}
	if w > 80 {
		w = 80
	}
	h := len(refs) + 4
	if h < 5 {
		h = 5
	}
	if h > 20 {
		h = 20
	}

	a.pages.AddPage("refs", modal(refsView, w, h), true, true)
	a.app.SetFocus(refsView)
}

func (a *App) closeReferences() {
	a.refsOpen = false
	a.pages.RemovePage("refs")
	a.app.SetFocus(a.panels[a.currentPanelIdx])
	a.updateBorderColors()
}

// duplicateVariant copies the selected variant file with a _copy suffix.
func (a *App) duplicateVariant() {
	idx := a.variantsPanel.GetCurrentItem()
	if idx < 0 || idx >= len(a.visibleVariantFiles) {
		return
	}

	name := a.visibleVariantFiles[idx]
	src := filepath.Join(a.variantDir, name+".yaml")

	dst := filepath.Join(a.variantDir, name+"_copy.yaml")
	suffix := 2
	for {
		_, err := os.Stat(dst)
		if os.IsNotExist(err) {
			break
		}
		if err != nil {
			a.viewerPanel.SetText(fmt.Sprintf("[red]Error checking file: %s[-]", err.Error()))
			return
		}
		dst = filepath.Join(a.variantDir, fmt.Sprintf("%s_copy%d.yaml", name, suffix))
		suffix++
	}

	data, err := os.ReadFile(src)
	if err != nil {
		a.viewerPanel.SetText(fmt.Sprintf("[red]Error reading file: %s[-]", err.Error()))
		return
	}
	// Preserve source file permissions.
	srcInfo, err := os.Stat(src)
	if err != nil {
		a.viewerPanel.SetText(fmt.Sprintf("[red]Error checking file: %s[-]", err.Error()))
		return
	}
	if err := os.WriteFile(dst, data, srcInfo.Mode()); err != nil {
		a.viewerPanel.SetText(fmt.Sprintf("[red]Error writing file: %s[-]", err.Error()))
		return
	}

	a.populateVariants(a.selectedBuilderNode)
}

// showDeleteConfirm shows a confirmation modal for deleting the selected variant.
func (a *App) showDeleteConfirm() {
	idx := a.variantsPanel.GetCurrentItem()
	if idx < 0 || idx >= len(a.visibleVariantFiles) {
		return
	}

	a.pendingDeleteIdx = idx
	a.pendingConfirmAction = config.ConfirmDelete

	if !a.cfg.Warnings.ShouldWarn(config.ConfirmDelete) {
		a.executeDelete()
		return
	}

	name := a.visibleVariantFiles[idx]

	// Find all experiments referencing this variant (including current).
	refs, err := hydra.FindVariantReferences(a.confDir, a.variantDir, name)
	if err != nil {
		a.viewerPanel.SetText(fmt.Sprintf("[red]Error finding references: %s[-]", err.Error()))
		return
	}

	a.showWarningModal(warningModalConfig{
		title:       " Confirm Delete ",
		borderColor: a.theme.ModalDeleteBorder,
		headerText:  fmt.Sprintf("\nDelete [yellow]%s.yaml[-]?\n", name),
		footerText:  "\n[green](y/n)[-]",
		refs:        refs,
	})
}

// executeDelete performs the actual file deletion after confirmation.
func (a *App) executeDelete() {
	idx := a.pendingDeleteIdx
	if idx < 0 || idx >= len(a.visibleVariantFiles) {
		a.closeConfirm()
		return
	}

	name := a.visibleVariantFiles[idx]
	filePath := filepath.Join(a.variantDir, name+".yaml")

	if err := os.Remove(filePath); err != nil {
		a.viewerPanel.SetText(fmt.Sprintf("[red]Error deleting file: %s[-]", err.Error()))
		a.closeConfirm()
		return
	}

	// If the deleted variant was the active one, set value to "??"
	if a.selectedBuilderNode != nil && a.selectedBuilderNode.Value == name {
		if err := hydra.UpdateDefaultValue(a.selectedBuilderNode.SourceFilePath, a.selectedBuilderNode.RawKey, "??"); err != nil {
			a.viewerPanel.SetText(fmt.Sprintf("[red]Error updating config: %s[-]", err.Error()))
		}

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

// unassignBuilderNode sets the selected builder node's value to "??" in the
// config file, effectively unassigning the package.
func (a *App) unassignBuilderNode() {
	node := a.selectedBuilderNode
	if node == nil || node.Value == "??" {
		return
	}

	// Top-level: no warning needed
	if node.SourceFilePath == a.expPath {
		a.executeUnassign()
		return
	}

	if !a.cfg.Warnings.ShouldWarn(config.ConfirmUnassign) {
		a.executeUnassign()
		return
	}

	// Deep node: check who uses the containing file
	allRefs, err := hydra.FindFileReferences(a.confDir, node.SourceFilePath)
	if err != nil {
		a.viewerPanel.SetText(fmt.Sprintf("[red]Error finding references: %s[-]", err.Error()))
		return
	}
	currentExp := a.assignedExperimentName()
	var otherRefs []string
	for _, r := range allRefs {
		if r != currentExp {
			otherRefs = append(otherRefs, r)
		}
	}
	if len(otherRefs) == 0 {
		a.executeUnassign()
		return
	}

	a.pendingConfirmAction = config.ConfirmUnassign
	a.showWarningModal(warningModalConfig{
		title:       " Confirm Unassign ",
		borderColor: a.theme.ModalWarningBorder,
		headerText:  fmt.Sprintf("\nUnassigning [yellow]%s[-] also affects:\n", node.Key),
		footerText:  "\nUnassign anyway? [green](y/n)[-]",
		refs:        otherRefs,
	})
}

// executeUnassign performs the actual unassign operation.
func (a *App) executeUnassign() {
	node := a.selectedBuilderNode
	if node == nil {
		return
	}

	if err := hydra.UpdateDefaultValue(node.SourceFilePath, node.RawKey, "??"); err != nil {
		a.viewerPanel.SetText(fmt.Sprintf("[red]Error updating config: %s[-]", err.Error()))
		return
	}

	sourceFile := node.SourceFilePath
	key := node.Key
	a.refreshAll()
	restoredIdx := a.findBuilderNodeIndex(sourceFile, key)
	if restoredIdx >= 0 {
		a.builderPanel.SetCurrentItem(restoredIdx)
		a.selectedBuilderNode = a.flatItems[restoredIdx]
	}
	a.populateVariants(a.selectedBuilderNode)
}

// closeConfirm closes the confirmation modal and resets pending state.
func (a *App) closeConfirm() {
	a.confirmOpen = false
	a.pendingConfirmAction = config.ConfirmDelete
	a.pendingReassignValue = ""
	a.pages.RemovePage("confirm")
	a.app.SetFocus(a.panels[a.currentPanelIdx])
	a.updateBorderColors()
}

// warningModalConfig holds parameters for building a shared warning modal.
type warningModalConfig struct {
	title       string
	borderColor tcell.Color
	headerText  string   // text before ref list
	footerText  string   // text after ref list
	refs        []string // experiment names
}

// showWarningModal displays a confirmation modal with a truncated list of
// experiment references.
func (a *App) showWarningModal(cfg warningModalConfig) {
	a.confirmOpen = true

	const maxVisible = 10
	shown := cfg.refs
	if len(shown) > maxVisible {
		shown = shown[:maxVisible]
	}
	var lines []string
	for i, name := range shown {
		lines = append(lines, fmt.Sprintf("  [green]%d.[-] %s", i+1, name))
	}
	if remaining := len(cfg.refs) - maxVisible; remaining > 0 {
		lines = append(lines, fmt.Sprintf("  [darkgray]and %d more experiments[-]", remaining))
	}

	var text string
	if len(cfg.refs) == 0 {
		text = fmt.Sprintf("%s%s", cfg.headerText, cfg.footerText)
	} else {
		text = fmt.Sprintf("%s\n%s%s", cfg.headerText, strings.Join(lines, "\n"), cfg.footerText)
	}

	confirmView := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter).
		SetText(text)
	confirmView.SetBorder(true).
		SetTitle(cfg.title).
		SetBorderColor(cfg.borderColor)

	displayCount := len(shown)
	if len(cfg.refs) > maxVisible {
		displayCount++
	}
	w := 50
	if titleW := len(cfg.title) + 4; titleW > w {
		w = titleW
	}
	for _, ref := range shown {
		if lineW := len(ref) + 10; lineW > w {
			w = lineW
		}
	}
	if w > 80 {
		w = 80
	}
	h := displayCount + 7
	if h < 7 {
		h = 7
	}
	if h > 22 {
		h = 22
	}

	a.pages.AddPage("confirm", modal(confirmView, w, h), true, true)
	a.app.SetFocus(confirmView)
}

// renameVariant checks for references before showing the rename input.
func (a *App) renameVariant() {
	idx := a.variantsPanel.GetCurrentItem()
	if idx < 0 || idx >= len(a.visibleVariantFiles) {
		return
	}

	if !a.cfg.Warnings.ShouldWarn(config.ConfirmRename) {
		a.showRenameInput()
		return
	}

	variantName := a.visibleVariantFiles[idx]
	refs, err := hydra.FindVariantReferences(a.confDir, a.variantDir, variantName)
	if err != nil {
		a.viewerPanel.SetText(fmt.Sprintf("[red]Error finding references: %s[-]", err.Error()))
		return
	}
	if len(refs) == 0 {
		a.showRenameInput()
		return
	}

	a.pendingConfirmAction = config.ConfirmRename
	a.showWarningModal(warningModalConfig{
		title:       " Confirm Rename ",
		borderColor: a.theme.ModalWarningBorder,
		headerText:  fmt.Sprintf("\nRenaming [yellow]%s[-] will update references in:\n", variantName),
		footerText:  "\nProceed? [green](y/n)[-]",
		refs:        refs,
	})
}

// showRenameInput displays the InputField modal for entering a new variant name.
func (a *App) showRenameInput() {
	idx := a.variantsPanel.GetCurrentItem()
	if idx < 0 || idx >= len(a.visibleVariantFiles) {
		return
	}

	a.renameOpen = true
	name := a.visibleVariantFiles[idx]

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
		SetBorderColor(a.theme.ModalHelpBorder)

	w := len(name) + 24
	if w < 50 {
		w = 50
	}
	a.pages.AddPage("rename", modal(inputField, w, 3), true, true)
	a.app.SetFocus(inputField)
}

// executeRename renames the variant file and propagates the rename across all
// experiments that reference it.
func (a *App) executeRename(idx int, newName string) {
	if idx < 0 || idx >= len(a.visibleVariantFiles) || newName == "" {
		a.closeRename()
		return
	}

	oldName := a.visibleVariantFiles[idx]
	if oldName == newName {
		a.closeRename()
		return
	}

	// Collect detailed refs BEFORE renaming the file on disk.
	detailedRefs, err := hydra.FindVariantReferencesDetailed(a.confDir, a.variantDir, oldName)
	if err != nil {
		a.viewerPanel.SetText(fmt.Sprintf("[red]Error finding references: %s[-]", err.Error()))
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

	// Update all referencing YAML files.
	for _, ref := range detailedRefs {
		if err := hydra.UpdateDefaultValue(ref.SourceFilePath, ref.RawKey, newName); err != nil {
			// Rollback file rename on failure.
			_ = os.Rename(newPath, oldPath)
			a.viewerPanel.SetText(fmt.Sprintf("[red]Error updating %s: %s[-]", ref.SourceFilePath, err.Error()))
			a.closeRename()
			return
		}
	}

	a.closeRename()

	sourceFile := ""
	key := ""
	if a.selectedBuilderNode != nil {
		sourceFile = a.selectedBuilderNode.SourceFilePath
		key = a.selectedBuilderNode.Key
	}
	a.refreshAll()
	if sourceFile != "" {
		restoredIdx := a.findBuilderNodeIndex(sourceFile, key)
		if restoredIdx >= 0 {
			a.builderPanel.SetCurrentItem(restoredIdx)
			a.selectedBuilderNode = a.flatItems[restoredIdx]
		}
	}
	a.populateVariants(a.selectedBuilderNode)
}

// closeRename closes the rename input modal.
func (a *App) closeRename() {
	a.renameOpen = false
	a.pages.RemovePage("rename")
	a.app.SetFocus(a.panels[a.currentPanelIdx])
	a.updateBorderColors()
}

// editVariantInEditor checks if the selected variant is referenced by any
// experiments and shows a confirmation modal if so, otherwise opens the editor.
func (a *App) editVariantInEditor() {
	idx := a.variantsPanel.GetCurrentItem()
	if idx < 0 || idx >= len(a.visibleVariantFiles) {
		return
	}

	if !a.cfg.Warnings.ShouldWarn(config.ConfirmEdit) {
		a.executeEditVariant()
		return
	}

	variantName := a.visibleVariantFiles[idx]
	refs, err := hydra.FindVariantReferences(a.confDir, a.variantDir, variantName)
	if err != nil {
		a.viewerPanel.SetText(fmt.Sprintf("[red]Error finding references: %s[-]", err.Error()))
		return
	}
	if len(refs) == 0 {
		a.executeEditVariant()
		return
	}

	a.pendingConfirmAction = config.ConfirmEdit
	a.showWarningModal(warningModalConfig{
		title:       " Confirm Edit ",
		borderColor: a.theme.ModalWarningBorder,
		headerText:  fmt.Sprintf("\n[yellow]%s[-] is used by:\n", variantName),
		footerText:  "\nEdit anyway? [green](y/n)[-]",
		refs:        refs,
	})
}

// executeEditVariant opens the selected variant file in $EDITOR.
func (a *App) executeEditVariant() {
	idx := a.variantsPanel.GetCurrentItem()
	if idx < 0 || idx >= len(a.visibleVariantFiles) {
		return
	}

	filePath := filepath.Join(a.variantDir, a.visibleVariantFiles[idx]+".yaml")
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = a.cfg.Editor
	}

	var editorErr error
	a.app.Suspend(func() {
		cmd := exec.Command(editor, filePath)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		editorErr = cmd.Run()
	})
	if editorErr != nil {
		a.viewerPanel.SetText(fmt.Sprintf("[red]Editor exited with error: %s[-]", editorErr.Error()))
	}
	a.refreshAll()
	a.populateVariants(a.selectedBuilderNode)
}
