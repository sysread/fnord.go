package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/sysread/fnord/pkg/debug"
)

type logView struct {
	*tview.Frame

	ui        *UI
	flex      *tview.Flex
	logBuffer *tview.TextView
}

func (ui *UI) newLogsView() *logView {
	flex := tview.NewFlex()

	flex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			ui.Quit()

		case tcell.KeyF10:
			ui.OpenChat()

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

	logBuffer := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true)

	flex.AddItem(logBuffer, 0, 1, true)

	lv := &logView{
		ui:        ui,
		flex:      flex,
		logBuffer: logBuffer,

		Frame: ui.newScreen(flex, screenArgs{
			title: "Logs",
			keys: []keyBinding{
				{"c", "chat"},
				{"?", "help"},
				{"q, esc", "quit"},
			},
		}),
	}

	lv.startLogReader()

	return lv
}

func (lv *logView) startLogReader() {
	go func() {
		for line := range debug.LogChannel {
			lv.ui.app.QueueUpdateDraw(func() {
				lv.logBuffer.Write([]byte(line + "\n"))
			})
		}
	}()
}
