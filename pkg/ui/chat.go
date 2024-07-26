package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func (ui *UI) newChatView() tview.Primitive {
	userInputPane := ui.newUserInput()

	messagePane := tview.NewBox().SetBorder(true)

	chatPane := tview.NewFlex()
	chatPane.SetDirection(tview.FlexRow)
	chatPane.AddItem(messagePane, 0, 3, false)
	chatPane.AddItem(userInputPane, 0, 1, false)

	flex := tview.NewFlex().
		AddItem(tview.NewBox().SetBorder(true).SetTitle("Conversations"), 0, 1, false).
		AddItem(chatPane, 0, 3, false)

	flex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case 'q':
			ui.OpenHome()
		}

		return event
	})

	return ui.newScreen(flex, screenArgs{
		title: "Chat",
		keys: []keyBinding{
			{'q', "home"},
		},
	})
}

func (ui *UI) newUserInput() tview.Primitive {
	input := tview.NewTextArea()
	input.SetBorder(true)
	input.SetTitle("Type your message here")

	return input
}
