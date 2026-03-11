package main

import (
	"fmt"

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

	builderPanel  *tview.List
	variantsPanel *tview.List
	viewerPanel   *tview.TextView
	statusBar     *tview.TextView

	helpOpen bool
}

func newApp() *App {
	selectionColor := tcell.NewRGBColor(106, 159, 181) // #6a9fb5

	builderPanel := tview.NewList().
		ShowSecondaryText(false).
		SetHighlightFullLine(true).
		SetSelectedBackgroundColor(selectionColor).
		SetSelectedTextColor(tcell.ColorWhite)
	builderPanel.SetBorder(true).
		SetTitle(" [1] Builder ").
		SetTitleAlign(tview.AlignLeft).
		SetBorderColor(tcell.ColorDefault)

	variantsPanel := tview.NewList().
		ShowSecondaryText(false).
		SetHighlightFullLine(true).
		SetSelectedBackgroundColor(selectionColor).
		SetSelectedTextColor(tcell.ColorWhite)
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

	statusBar := tview.NewTextView().
		SetDynamicColors(true)
	statusBar.SetBorder(false)

	a := &App{
		app:           tview.NewApplication(),
		builderPanel:  builderPanel,
		variantsPanel: variantsPanel,
		viewerPanel:   viewerPanel,
		statusBar:     statusBar,
		panels:        []tview.Primitive{builderPanel, variantsPanel},
	}

	// Build layout
	leftFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(a.builderPanel, 0, 1, true).
		AddItem(a.variantsPanel, 0, 1, false)

	mainFlex := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(leftFlex, 0, 2, true).
		AddItem(a.viewerPanel, 0, 3, false)

	rootFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(mainFlex, 0, 1, true).
		AddItem(a.statusBar, 1, 0, false)

	a.pages = tview.NewPages().
		AddPage("main", rootFlex, true, true)

	a.refreshAll()
	a.setupKeybindings()
	a.updateBorderColors()
	a.updateStatusBar()

	return a
}

func (a *App) refreshAll() {
	// Builder placeholder items
	a.builderPanel.Clear()
	for _, item := range []string{
		"supervised_pre_training_base",
		"evidence_evaluation",
		"feature_matching",
		"monte_carlo_search",
		"multi_object_detection",
	} {
		a.builderPanel.AddItem(item, "", 0, nil)
	}

	// Variants placeholder items
	a.variantsPanel.Clear()
	for _, item := range []string{
		"default",
		"debug",
		"5lms",
		"fast",
		"high_accuracy",
	} {
		a.variantsPanel.AddItem(item, "", 0, nil)
	}

	// Viewer placeholder
	a.viewerPanel.SetText("[darkgray]Select a config to preview[-]")
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
	selectionColor := tcell.NewRGBColor(106, 159, 181) // #6a9fb5

	for i, panel := range a.panels {
		list := panel.(*tview.List)
		if i == a.currentPanelIdx {
			list.SetBorderColor(tcell.ColorGreen)
			list.SetSelectedBackgroundColor(selectionColor)
		} else {
			list.SetBorderColor(tcell.ColorDefault)
			list.SetSelectedBackgroundColor(tcell.ColorDefault)
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
	0: " [::b]Builder[-:-:-]  [j/k] navigate  [h/l] panels  [J/K] scroll viewer  [?] help  [q] quit",
	1: " [::b]Variants[-:-:-]  [j/k] navigate  [h/l] panels  [J/K] scroll viewer  [?] help  [q] quit",
}

func (a *App) updateStatusBar() {
	if text, ok := statusBarTexts[a.currentPanelIdx]; ok {
		a.statusBar.SetText(text)
	} else {
		a.statusBar.SetText(fmt.Sprintf(" Panel %d", a.currentPanelIdx))
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
