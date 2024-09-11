package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func (ui *UI) newHomeView() tview.Primitive {
	home := tview.NewFlex()

	home.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			ui.Quit()

		default:
			switch event.Rune() {
			case 'q':
				ui.Quit()

			case '?':
				ui.OpenHelp()

			case 'c':
				ui.OpenChat()
			}
		}

		return event
	})

	return ui.newScreen(home, screenArgs{
		title: "Fnord",
		keys: []keyBinding{
			{"c", "chat"},
			{"?", "help"},
			{"F10", "logs"},
			{"q, esc", "quit"},
		},
	})
}
