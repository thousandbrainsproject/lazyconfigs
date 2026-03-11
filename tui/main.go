package main

import (
	"fmt"
	"os"
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
	variantNames []string
	confDir      string

	helpOpen bool
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
			a.updateViewer(a.flatItems[index])
		}
	})

	variantsPanel.SetChangedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
		a.refreshVariantsSelection(index)
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
	// Load tree from hardcoded experiment
	expPath := filepath.Join(a.confDir, "experiment", "base_config_10distinctobj_dist_agent.yaml")
	roots, err := buildTree(expPath, a.confDir)
	if err != nil {
		a.builderPanel.Clear()
		a.builderPanel.AddItem(fmt.Sprintf("[red]Error: %s[-]", err.Error()), "", 0, nil)
	} else {
		a.rootNodes = roots
		a.rebuildBuilderList()
	}

	// Variants placeholder items
	a.variantNames = []string{"default", "debug", "5lms", "fast", "high_accuracy"}
	a.variantsPanel.Clear()
	for i, item := range a.variantNames {
		a.variantsPanel.AddItem(renderVariantItem(item, i == 0), "", 0, nil)
	}

	// Viewer placeholder
	a.viewerPanel.SetText("[darkgray]Select a config to preview[-]")
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
	for i, name := range a.variantNames {
		if i >= a.variantsPanel.GetItemCount() {
			break
		}
		a.variantsPanel.SetItemText(i, renderVariantItem(name, i == selectedIdx), "")
	}
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
	a.currentPanelIdx = idx
	a.app.SetFocus(a.panels[a.currentPanelIdx])
	a.updateBorderColors()
	a.updateStatusBar()
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

var statusBarTexts = map[int]string{
	0: " Navigate: j/k | Expand/Collapse: Enter | Panels: h/l | Scroll Viewer: J/K | Help: ? | Quit: q",
	1: " Navigate: j/k | Panels: h/l | Scroll Viewer: J/K | Help: ? | Quit: q",
}

func (a *App) updateStatusBar() {
	if text, ok := statusBarTexts[a.currentPanelIdx]; ok {
		a.statusBarLeft.SetText(text)
	} else {
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

[green]Viewer:[-]
  J / K         Scroll viewer

[green]General:[-]
  ?             This help
  Esc           Close overlay
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

	a.pages.AddPage("help", modal(helpView, 55, 18), true, true)
	a.app.SetFocus(helpView)
}

func (a *App) closeHelp() {
	a.helpOpen = false
	a.pages.RemovePage("help")
	a.app.SetFocus(a.panels[a.currentPanelIdx])
	a.updateBorderColors()
}

func main() {
	a := newApp()
	a.app.SetRoot(a.pages, true).EnableMouse(false)
	if err := a.app.Run(); err != nil {
		panic(err)
	}
}
