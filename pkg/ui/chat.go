package ui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/sysread/fnord/pkg/gpt"
)

type chatInput struct {
	*tview.TextArea
}

type chatView struct {
	ui *UI

	gptClient *gpt.OpenAIClient

	conversation gpt.Conversation

	*tview.Frame
	container   *tview.Flex
	messageList *tview.TextView
	userInput   *chatInput
}

func (ui *UI) newChatView() chatView {
	cv := chatView{
		ui:           ui,
		gptClient:    gpt.NewOpenAIClient(),
		conversation: []gpt.ChatMessage{},
	}

	cv.userInput = cv.newChatInput()

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
func (cv *chatView) SetFocus(ui *UI) {
	ui.app.SetFocus(cv.userInput)
}

// Builds the chatInput component, which is a text area that captures user
// input and sends it to the assistant when the user presses Ctrl+Space.
func (cv *chatView) newChatInput() *chatInput {
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
			// Disable the chat input while the assistant is responding
			chatInput.SetDisabled(true)

			// TODO handle errors:
			//   - if a file in the message(s) doesn't exist, display file picker
			messages, _ := gpt.ParseMessage(gpt.You, chatInput.GetText())

			// Clear the chat input after the user has sent the message
			chatInput.SetText("", false)

			// Create a channel to signal when the assistant has finished
			done := make(chan struct{})

			// Send the user message to the assistant and get the response
			go func() {
				for _, message := range messages {
					// Add the user message to the chat view and conversation.
					cv.ui.app.QueueUpdateDraw(func() {
						cv.addMessage(message)
					})
				}

				// Get the assistant's response
				response, _ := cv.gptClient.GetCompletion(cv.conversation)
				responseMessage := gpt.ChatMessage{
					From:    gpt.Assistant,
					Content: response,
				}

				// Add the assistant's response to the chat view and
				// conversation.
				cv.ui.app.QueueUpdateDraw(func() {
					cv.addMessage(responseMessage)
				})

				done <- struct{}{}
			}()

			// Re-enable the chat input when the assistant has finished
			go func() {
				<-done

				cv.ui.app.QueueUpdateDraw(func() {
					chatInput.SetDisabled(false)
				})
			}()

			// Return nil to indicate the event has been handled
			return nil
		}

		return event
	})

	return chatInput
}

// Adds a message to the chat view. This is the function that is called when a
// new message is received from the chat input or a new response is generated
// by the assistant.
func (cv *chatView) addMessage(msg gpt.ChatMessage) {
	// Append the message to the conversation
	cv.conversation = append(cv.conversation, msg)

	// Action messages may include messages that are not to be displayed, like
	// the contents of a file chunk. In this case, we don't want to add the
	// message to the visible message list. However, we still want to add it to
	// the conversation.
	if msg.IsHidden {
		return
	}

	// Create a new message view and add it to the message list
	color := "blue"
	if msg.From != gpt.You {
		color = "green"
	}

	fmt.Fprintf(cv.messageList, "[%s]%s:\n\n[white]%s\n\n", color, msg.From, msg.Content)

	// Scroll to the last message when a new message is added
	cv.messageList.ScrollToEnd()
}
