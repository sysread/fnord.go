package ui

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/sysread/fnord/pkg/gpt"
)

type chatInput struct {
	cv *chatView
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

func (ui *UI) newChatView() *chatView {
	cv := &chatView{
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

func (cv *chatView) GetInitialFocus() tview.Primitive {
	return cv.userInput
}

// Builds the chatInput component, which is a text area that captures user
// input and sends it to the assistant when the user presses Ctrl+Space.
func (cv *chatView) newChatInput() *chatInput {
	chatInput := &chatInput{
		cv:       cv,
		TextArea: tview.NewTextArea(),
	}

	chatInput.SetBorder(true)
	chatInput.SetTitle("Type your message here")

	chatInput.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if chatInput.GetDisabled() {
			return event
		}

		if event.Key() == tcell.KeyCtrlSpace {
			go chatInput.onSubmit()

			// Return nil to indicate the event has been handled
			return nil
		}

		return event
	})

	return chatInput
}

func (ci *chatInput) onSubmit() {
	// Disable the chat input while the assistant is responding
	ci.SetDisabled(true)

	messages := []gpt.ChatMessage{}
	messageText := ci.GetText()
	for {
		parsed, err := gpt.ParseMessage(gpt.You, messageText)

		if err == nil {
			messages = parsed
			break
		}

		if fileDoesNotExist, ok := err.(*gpt.FileDoesNotExist); ok {
			prompt := fmt.Sprintf("File '%s' not found! Please select the file you intended.", fileDoesNotExist.FilePath)

			done := make(chan bool)

			ci.cv.ui.app.QueueUpdateDraw(func() {
				ci.cv.ui.OpenFilePicker(prompt, ".", func(replacementFilePath string) {
					messageText = strings.Replace(messageText, "\\f "+fileDoesNotExist.FilePath, "\\f "+replacementFilePath, 1)
					ci.cv.ui.OpenChat()
					done <- true
				})
			})

			<-done
		}
	}

	// Clear the chat input after the user has sent the message
	ci.SetText("", false)

	if len(messages) == 0 {
		ci.SetDisabled(false)
		return
	}

	// Create a channel to signal when the assistant has finished
	done := make(chan bool)

	// Send the user message to the assistant and get the response
	go func() {
		for _, message := range messages {
			// Add the user message to the chat view and conversation.
			ci.cv.ui.app.QueueUpdateDraw(func() {
				ci.cv.addMessage(message)
			})
		}

		// Get the assistant's response
		response, _ := ci.cv.gptClient.GetCompletion(ci.cv.conversation)
		responseMessage := gpt.ChatMessage{
			From:    gpt.Assistant,
			Content: response,
		}

		// Add the assistant's response to the chat view and
		// conversation.
		ci.cv.ui.app.QueueUpdateDraw(func() {
			ci.cv.addMessage(responseMessage)
		})

		done <- true
	}()

	// Re-enable the chat input when the assistant has finished
	go func() {
		<-done

		ci.cv.ui.app.QueueUpdateDraw(func() {
			ci.SetDisabled(false)
		})
	}()
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
