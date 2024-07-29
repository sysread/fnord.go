package ui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/sysread/fnord/pkg/common"
	"github.com/sysread/fnord/pkg/debug"
	"github.com/sysread/fnord/pkg/gpt"
)

type chatView struct {
	ui *UI

	gptClient *gpt.OpenAIClient

	conversation []common.ChatMessage

	*tview.Frame
	container   *tview.Flex
	messagePane *tview.TextView
	userInput   *chatInput
}

func (ui *UI) newChatView() chatView {
	cv := chatView{
		ui:           ui,
		gptClient:    gpt.NewOpenAIClient(),
		conversation: []common.ChatMessage{},
	}

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

func (cv chatView) SetFocus(ui *UI) {
	ui.app.SetFocus(cv.userInput)
}

func (cv *chatView) buildChatUserInput() *chatInput {
	return newChatInput(func(msg common.ChatMessage) {
		cv.addMessage(msg)
		cv.getResponse()
	})
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

func (cv *chatView) newChatMessage(msg common.ChatMessage) tview.Primitive {
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

	fmt.Fprintf(cv.messagePane, "[%s]%s:\n\n[white]%s\n\n", color, msg.From, msg.Content)
}

func (cv *chatView) getResponse() {
	debug.Log("Getting response from GPT")

	response, err := cv.gptClient.GetCompletion(cv.conversation)

	debug.Log("Response from GPT: %s", response)
	debug.Log("Error from GPT: %v", err)

	msg := common.ChatMessage{
		From:    common.Assistant,
		Content: response,
	}

	cv.ui.app.QueueUpdateDraw(func() {
		cv.addMessage(msg)
	})
}
