package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func (ui *UI) newHelpView() tview.Primitive {
	helpText := `
Key bindings:

	[blue]     c[-] - Start a new chat

	[blue]     ?[-] - Show this help
	[blue]   F10[-] - Display logs
	[blue]Escape[-] - Quit the application
	`

	help := tview.NewTextView()

	help.SetText(helpText)
	help.SetTextAlign(tview.AlignLeft)
	help.SetDynamicColors(true)

	help.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			ui.OpenHome()
		} else {
			switch event.Rune() {
			case 'q':
				ui.OpenHome()
			}
		}

		return event
	})

	return ui.newScreen(help, screenArgs{
		title: "Help",
		keys: []keyBinding{
			{"q, esc", "home"},
		},
	})
}
