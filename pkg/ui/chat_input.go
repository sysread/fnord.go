package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/sysread/fnord/pkg/common"
)

type chatInput struct {
	*tview.TextArea
}

func newChatInput(onNewMessage func(common.ChatMessage)) *chatInput {
	chatInput := &chatInput{
		TextArea: tview.NewTextArea(),
	}

	chatInput.SetBorder(true)
	chatInput.SetTitle("Type your message here")

	chatInput.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if chatInput.GetDisabled() {
			return event
		}

		if event.Key() == tcell.KeyCtrlSpace {
			chatInput.SetDisabled(true)

			msg := common.ChatMessage{
				From:    common.You,
				Content: chatInput.GetText(),
			}

			chatInput.Clear()

			done := make(chan struct{})

			go func() {
				onNewMessage(msg)
				done <- struct{}{}
			}()

			go func() {
				<-done
				chatInput.SetDisabled(false)
			}()

			// Return nil to indicate the event has been handled
			return nil
		}

		return event
	})

	return chatInput
}

func (ci *chatInput) Clear() {
	ci.SetText("", false)
}
