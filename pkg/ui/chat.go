package ui

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/atotto/clipboard"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/sysread/textsel"
	"github.com/charmbracelet/glamour"

	"github.com/sysread/fnord/pkg/chat"
	"github.com/sysread/fnord/pkg/gpt"
	"github.com/sysread/fnord/pkg/messages"
)

type chatInput struct {
	chatView *chatView
	*tview.TextArea
}

type chatView struct {
	*tview.Frame
	ui              *UI
	gptClient       *gpt.OpenAIClient
	chat            *chat.Chat
	container       *tview.Flex
	chatFlex        *tview.Flex
	messageList     *textsel.TextSel
	userInput       *chatInput
	receivingBuffer *tview.TextView
	isReceiving     bool
}

func (ui *UI) newChatView() *chatView {
	cv := &chatView{
		ui:        ui,
		gptClient: gpt.NewOpenAIClient(),
		chat:      chat.NewChat(),
	}

	cv.container = tview.NewFlex().
		SetDirection(tview.FlexRow)

	cv.receivingBuffer = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWordWrap(true)

	cv.userInput = cv.newChatInput()

	cv.messageList = textsel.NewTextSel()
	cv.messageList.
		SetDynamicColors(true).
		SetScrollable(true).
		SetWordWrap(true)

	cv.messageList.SetSelectFunc(func(s string) {
		// Write to paste buffer
		copyText := stripTviewTags(strings.TrimSpace(s))
		clipboard.WriteAll(copyText)

		// Reset focus to chat input
		cv.ui.app.SetFocus(cv.messageList)
		cv.messageList.MoveToLastLine()
	})

	cv.chatFlex = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(cv.messageList, 0, 5, false).
		AddItem(cv.userInput, 0, 1, false)

	cv.container.AddItem(cv.chatFlex, 0, 1, false)

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

func (cv *chatView) ToggleReceiving() {
	if cv.isReceiving {
		lastMessage := cv.chat.LastMessage()
		if lastMessage != nil {
			cv.container.RemoveItem(cv.receivingBuffer)
			cv.container.AddItem(cv.chatFlex, 0, 1, false)
			cv.isReceiving = false

			cv.queueAppendText("[green::b]Assistant:[-:-:-]\n")
			cv.queueAppendText(glamourize(lastMessage.Content))
			cv.messageList.ScrollToEnd()
		}
	} else {
		cv.receivingBuffer.SetText(cv.messageList.GetText(false))
		cv.container.RemoveItem(cv.chatFlex)
		cv.container.AddItem(cv.receivingBuffer, 0, 1, false)
		cv.isReceiving = true
		cv.receivingBuffer.ScrollToEnd()
	}
}

func (ci *chatInput) onSubmit() {
	// Disable the chat input while the assistant is responding
	ci.SetDisabled(true)

	var msgs []messages.ChatMessage
	messageText := ci.GetText()

	// Parse the user message
	for {
		parsed, err := messages.ParseMessage(messages.You, messageText)

		if err == nil {
			msgs = parsed
			break
		}

		if fileDoesNotExist, ok := err.(*messages.FileDoesNotExist); ok {
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

	if len(msgs) == 0 {
		ci.SetDisabled(false)
		return
	}

	// Add the parsed user messages to the chat view and conversation.
	for _, msg := range msgs {
		content := glamourize(msg.Content)
		ci.chatView.queueAppendText("[blue::b]You:[-:-:-]\n" + content)
		ci.chatView.chat.AddMessage(msg)
		ci.chatView.messageList.ScrollToEnd()
		ci.chatView.messageList.MoveToLastLine()
	}

	// Get the assistant's response
	ci.chatView.ToggleReceiving()
	ci.chatView.queueAppendText("[green::b]Assistant:[-:-:-]\n")
	ci.chatView.chat.RequestResponse(func(chunk string) {
		// Append the assistant's response to the chat view
		ci.chatView.queueAppendText(chunk)
	})

	// Now that the response is complete, append a few newlines to separate it
	// from the next user message and scroll to the end of the chat view.
	ci.chatView.ToggleReceiving()

	// Re-enable the chat input
	ci.SetDisabled(false)
}

// Appends text to the chat view.
func (cv *chatView) queueAppendText(text string) {
	if cv.isReceiving {
		cv.ui.app.QueueUpdateDraw(func() {
			cv.receivingBuffer.SetText(cv.receivingBuffer.GetText(false) + asciiDamnit(text))
			cv.receivingBuffer.ScrollToEnd()
		})
	} else {
		cv.ui.app.QueueUpdateDraw(func() {
			cv.messageList.SetText(cv.messageList.GetText(false) + asciiDamnit(text))
			cv.messageList.ScrollToEnd()
		})
	}
}

// stripTviewTags removes tview tags from a string.
func stripTviewTags(input string) string {
	re := regexp.MustCompile(`\[[^\[\]]*\]`)
	return re.ReplaceAllString(input, "")
}

// glamourize converts markdown to tview tags by first rendering the markdown
// as ANSI using glamour and then translating the ANSI to tview tags.
func glamourize(content string) string {
	// Render the markdown content with ANSI escapes using glamour
	rendered, _ := glamour.Render(content, "dark")

	// Translate the ANSI-escaped content to tview tags
	rendered = tview.TranslateANSI(rendered)

	// Glamour-rendered markdown is indented by two spaces, so we will remove
	// up to two spaces at the beginning of each line.
	leadingSpacesRe := regexp.MustCompile(`(?m)^ {1,2}`)
	rendered = leadingSpacesRe.ReplaceAllString(rendered, "")

	return rendered
}

// asciiDamnit converts the raw bytes of box-drawing characters into their
// ASCII equivalents. tview's ANSIWriter and github.com/rivo/uniseg handle some
// ANSI escape codes, but it does not box drawing characters, such as those
// output by tree(1).
func asciiDamnit(input string) string {
	bytes := []byte(input)

	var sb strings.Builder

	for len(bytes) > 0 {
		r, size := utf8.DecodeRune(bytes)
		sb.WriteString(unicodeToASCII(r))
		bytes = bytes[size:]
	}

	return sb.String()
}

// unicodeToASCII maps Unicode box-drawing characters to their ASCII approximations.
func unicodeToASCII(r rune) string {
	switch r {
	case '•':
		return "*"
	case '…':
		return "..."
	case '─': // U+2500
		return "-"
	case '│': // U+2502
		return "|"
	case '┌': // U+250C
		return "+"
	case '┐': // U+2510
		return "+"
	case '└': // U+2514
		return "+"
	case '┘': // U+2518
		return "+"
	case '├': // U+251C
		return "+"
	case '┤': // U+2524
		return "+"
	case '┬': // U+252C
		return "+"
	case '┴': // U+2534
		return "+"
	case '┼': // U+253C
		return "+"
	case '═': // U+2550
		return "="
	case '║': // U+2551
		return "||"
	case '╔': // U+2554
		return "+"
	case '╗': // U+2557
		return "+"
	case '╚': // U+255A
		return "+"
	case '╝': // U+255D
		return "+"
	case '╠': // U+2560
		return "+"
	case '╣': // U+2563
		return "+"
	case '╦': // U+2566
		return "+"
	case '╩': // U+2569
		return "+"
	case '╬': // U+256C
		return "+"
	case '\u00A0': // Non-breaking space U+00A0
		return " "
	case ' ': // ASCII space U+0020
		return " "
	default:
		return string(r) // Return the original character if no approximation found
	}
}
