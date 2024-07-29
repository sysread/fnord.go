package ui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type chatView struct {
	*tview.Frame
	messagePane *tview.TextView
	userInput   *tview.TextArea
	container   *tview.Flex
}

func (ui *UI) newChatView() chatView {
	cv := chatView{}

	cv.messagePane = cv.buildChatMessagePane()
	cv.userInput = cv.buildChatUserInput()

	cv.container = tview.NewFlex()
	cv.container.SetDirection(tview.FlexRow)
	cv.container.AddItem(cv.messagePane, 0, 5, false)
	cv.container.AddItem(cv.userInput, 0, 1, false)

	cv.Frame = ui.newScreen(cv.container, screenArgs{
		title: "Chat",
		keys: []keyBinding{
			{"ctrl-space", "sends"},
			{"q", "home"},
		},
	})

	cv.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case 'q':
			ui.OpenHome()
		}

		return event
	})

	return cv
}

func (cv chatView) SetFocus(ui *UI) {
	ui.app.SetFocus(cv.userInput)
}

func (cv *chatView) buildChatUserInput() *tview.TextArea {
	input := tview.NewTextArea()
	input.SetBorder(true)
	input.SetTitle("Type your message here")

	input.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyCtrlSpace {
			text := input.GetText()

			if text != "" {
				fmt.Fprintf(cv.messagePane, "[blue]You:\n\n[white]%s\n\n", text)
			}

			input.SetText("", false)

			// Return nil to indicate the event has been handled
			return nil
		}

		return event
	})

	return input
}

func (cv *chatView) buildChatMessagePane() *tview.TextView {
	chatHistory := tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetScrollable(true)

	// Auto-scroll to the bottom when new content is added
	chatHistory.SetChangedFunc(func() {
		chatHistory.ScrollToEnd()
	})

	return chatHistory
}

func (cv *chatView) newChatMessage(from string, message string) tview.Primitive {
	senderBox := tview.NewTextView()
	senderBox.SetText(from)
	senderBox.SetBackgroundColor(tcell.ColorLightGreen)
	senderBox.SetTextColor(tcell.ColorBlack)

	messageBox := tview.NewTextView()
	messageBox.SetText(message)

	flex := tview.NewFlex()
	flex.SetDirection(tview.FlexRow)
	flex.AddItem(senderBox, 1, 0, false)
	flex.AddItem(messageBox, 0, 8, false)

	return flex
}
