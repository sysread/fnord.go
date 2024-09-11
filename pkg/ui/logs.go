package ui

import (
	"time"

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

		// TODO implement keybindings
		Frame: ui.newScreen(flex, screenArgs{
			title: "Logs",
			keys: []keyBinding{
				{"c", "chat"},
				{"?", "help"},
				{"q, esc", "quit"},
			},
		}),
	}

	go lv.startLogReader()

	return lv
}

func (lv *logView) startLogReader() {
	// Wait until the log channel is initialized
	for {
		if debug.LogChannel == nil {
			debug.Log("Waiting for log channel to be initialized")
			time.Sleep(200 * time.Millisecond)
		}
	}

	out := lv.logBuffer.BatchWriter()
	for line := range debug.LogChannel {
		out.Write([]byte(line + "\n"))
	}
}
