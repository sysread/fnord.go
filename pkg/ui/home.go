package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func (ui *UI) newHomeView() tview.Primitive {
	home := tview.NewFlex()

	home.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			ui.Quit()
		} else {
			switch event.Rune() {
			case 'c':
				ui.OpenChat()
			case '?':
				ui.OpenHelp()
			}
		}

		return event
	})

	return ui.newScreen(home, screenArgs{
		title: "Fnord",
		keys: []keyBinding{
			{"c", "chat"},
			{"?", "help"},
			{"esc", "quit"},
		},
	})
}
