package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func (ui *UI) newHomeView() tview.Primitive {
	home := tview.NewTextView()

	home.SetTextAlign(tview.AlignCenter)
	home.SetDynamicColors(true)

	home.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case '?':
			ui.OpenHelp()
		case 'q':
			ui.Quit()
		case 'c':
			ui.OpenChat()
		}

		return event
	})

	return ui.newScreen(home, screenArgs{
		title: "Fnord",
		keys: []keyBinding{
			{'c', "chat"},
			{'?', "help"},
			{'q', "quit"},
		},
	})
}
