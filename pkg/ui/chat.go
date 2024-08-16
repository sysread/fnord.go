package ui

import (
	"fmt"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/sysread/textsel"

	"github.com/sysread/fnord/pkg/gpt"
)

type chatInput struct {
	chatView *chatView
	*tview.TextArea
}

type chatView struct {
	ui *UI

	gptClient *gpt.OpenAIClient

	conversation gpt.Conversation

	*tview.Frame
	container   *tview.Flex
	messageList *textsel.TextSel
	userInput   *chatInput
}

func (ui *UI) newChatView() *chatView {
	cv := &chatView{
		ui:           ui,
		gptClient:    gpt.NewOpenAIClient(),
		conversation: []gpt.ChatMessage{},
	}

	cv.userInput = cv.newChatInput()

	cv.messageList = textsel.NewTextSel()

	cv.messageList.
		SetScrollable(true).
		SetWordWrap(true)

	cv.messageList.SetSelectFunc(func(s string) {
		clipboard.WriteAll(strings.TrimSpace(s))
	})

	cv.container = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(cv.messageList, 0, 5, false).
		AddItem(cv.userInput, 0, 1, false)

	cv.Frame = ui.newScreen(cv.container, screenArgs{
		title: "Chat",
		keys: []keyBinding{
			{"ctrl-space", "sends"},
			{"shift-tab", "switches focus"},
			{"space, enter", "select, copy (in msgs)"},
			{"esc", "home"},
		},
	})

	cv.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			ui.OpenHome()
			return nil
		case tcell.KeyBacktab:
			if cv.ui.app.GetFocus() == cv.userInput {
				cv.ui.app.SetFocus(cv.messageList)
				cv.messageList.MoveToLastLine()
			} else {
				cv.ui.app.SetFocus(cv.userInput)
				cv.messageList.ScrollToEnd()
				cv.messageList.ResetCursor()
			}
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
		chatView: cv,
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

	var messages []gpt.ChatMessage
	messageText := ci.GetText()

	// Parse the user message
	for {
		parsed, err := gpt.ParseMessage(gpt.You, messageText)

		if err == nil {
			messages = parsed
			break
		}

		if fileDoesNotExist, ok := err.(*gpt.FileDoesNotExist); ok {
			prompt := fmt.Sprintf("File '%s' not found! Please select the file you intended.", fileDoesNotExist.FilePath)

			done := make(chan bool)

			ci.chatView.ui.app.QueueUpdateDraw(func() {
				ci.chatView.ui.OpenFilePicker(prompt, ".", func(replacementFilePath string) {
					messageText = strings.Replace(messageText, "\\f "+fileDoesNotExist.FilePath, "\\f "+replacementFilePath, 1)
					ci.chatView.ui.OpenChat()
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

	// Add the parsed user messages to the chat view and conversation.
	for _, message := range messages {
		ci.chatView.queueAppendText("[blue::b]You:[-:-:-]\n\n" + message.Content + "\n\n")
		ci.chatView.conversation = append(ci.chatView.conversation, message)
		ci.chatView.messageList.ScrollToEnd()
		ci.chatView.messageList.MoveToLastLine()
	}

	// Get the assistant's response
	responseChan := ci.chatView.gptClient.GetCompletionStream(ci.chatView.conversation)

	response := gpt.ChatMessage{
		From:    gpt.Assistant,
		Content: "",
	}

	// Append the response to the chat messages view
	go func() {
		ci.chatView.queueAppendText("[green::b]Assistant:[-:-:-]\n\n")

		for chunk := range responseChan {
			// Add the assistant's response to the chat view
			ci.chatView.queueAppendText(chunk)

			// Update the ChatMessage that will be part of the conversation
			response.Content += chunk
		}

		ci.chatView.queueAppendText("\n\n")
		ci.chatView.messageList.ScrollToEnd()

		ci.chatView.conversation = append(ci.chatView.conversation, response)

		done <- true
	}()

	// Re-enable the chat input when the assistant has finished
	go func() {
		<-done

		ci.chatView.ui.app.QueueUpdateDraw(func() {
			ci.SetDisabled(false)
		})
	}()
}

// Appends text to the chat view.
func (cv *chatView) queueAppendText(text string) {
	cv.ui.app.QueueUpdateDraw(func() {
		cv.messageList.SetText(cv.messageList.GetText(false) + text)
	})
}
