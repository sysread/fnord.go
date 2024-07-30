package ui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/sysread/fnord/pkg/common"
	"github.com/sysread/fnord/pkg/debug"
	"github.com/sysread/fnord/pkg/gpt"
)

type chatInput struct {
	*tview.TextArea
}

type chatView struct {
	ui *UI

	gptClient *gpt.OpenAIClient

	conversation []common.ChatMessage

	*tview.Frame
	container   *tview.Flex
	messageList *tview.TextView
	userInput   *chatInput
}

func (ui *UI) newChatView() chatView {
	cv := chatView{
		ui:           ui,
		gptClient:    gpt.NewOpenAIClient(),
		conversation: []common.ChatMessage{},
	}

	cv.userInput = newChatInput(cv.getNextAssistantResponse)

	cv.messageList = tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetScrollable(true)

	cv.container = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(cv.messageList, 0, 5, false).
		AddItem(cv.userInput, 0, 1, false)

	cv.Frame = ui.newScreen(cv.container, screenArgs{
		title: "Chat",
		keys: []keyBinding{
			{"ctrl-space", "sends"},
			{"esc", "home"},
		},
	})

	cv.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			ui.OpenHome()
		}

		return event
	})

	return cv
}

// Focuses the chat input when the chat view is opened
func (cv chatView) SetFocus(ui *UI) {
	ui.app.SetFocus(cv.userInput)
}

// Builds the chatInput component, which is a text area that captures user
// input and sends it to the assistant when the user presses Ctrl+Space.
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

			chatInput.SetText("", false)

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

// Callback used by chatInput to get the next chat completion response from GPT.
func (cv *chatView) getNextAssistantResponse(msg common.ChatMessage) {
	cv.addMessage(msg)

	response, _ := cv.gptClient.GetCompletion(cv.conversation)

	newMsg := common.ChatMessage{
		From:    common.Assistant,
		Content: response,
	}

	cv.ui.app.QueueUpdateDraw(func() {
		cv.addMessage(newMsg)
	})
}

// Creates a new message view with the sender and message content. This is the
// primitive that is added to the chat view.
func (cv *chatView) newMessage(msg common.ChatMessage) tview.Primitive {
	senderBox := tview.NewTextView()
	senderBox.SetText(string(msg.From))
	senderBox.SetBackgroundColor(tcell.ColorLightGreen)
	senderBox.SetTextColor(tcell.ColorBlack)

	messageBox := tview.NewTextView()
	messageBox.SetText(msg.Content)

	flex := tview.NewFlex()
	flex.SetDirection(tview.FlexRow)
	flex.AddItem(senderBox, 1, 0, false)
	flex.AddItem(messageBox, 0, 8, false)

	return flex
}

// Adds a message to the chat view. This is the function that is called when a
// new message is received from the chat input or a new response is generated
// by the assistant.
func (cv *chatView) addMessage(msg common.ChatMessage) {
	debug.Log("Adding message to chat: %v", msg)

	if msg.Content == "" {
		return
	}

	cv.conversation = append(cv.conversation, msg)

	color := "blue"
	if msg.From != common.You {
		color = "green"
	}

	fmt.Fprintf(cv.messageList, "[%s]%s:\n\n[white]%s\n\n", color, msg.From, msg.Content)

	// Scroll to the last message when a new message is added
	cv.messageList.ScrollToEnd()
}
