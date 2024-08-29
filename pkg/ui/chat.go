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

	"github.com/sysread/fnord/pkg/chat_manager"
	"github.com/sysread/fnord/pkg/debug"
	"github.com/sysread/fnord/pkg/markdown"
	"github.com/sysread/fnord/pkg/messages"
)

const slashHelp = `
Slash Commands
--------------
\f - Send a file contents
\x - Send command output
--------------
escape closes
`

type chatView struct {
	*tview.Frame

	ui      *UI
	chatMgr *chat_manager.ChatManager

	container *tview.Flex

	chatFlex    *tview.Flex
	messageList *textsel.TextSel
	userInput   *tview.TextArea

	receivingBuffer *tview.TextView
	isReceiving     bool

	helpModal       *tview.Modal
	helpModalIsOpen bool
}

func (ui *UI) newChatView() *chatView {
	cv := &chatView{
		ui:      ui,
		chatMgr: chat_manager.NewChatManager(ui.Context),
	}

	cv.container = tview.NewFlex().
		SetDirection(tview.FlexRow)

	cv.helpModal = tview.NewModal()
	cv.helpModal.SetText(slashHelp).
		SetDoneFunc(func(_ int, _ string) {
			cv.toggleHelp()
		})

	cv.receivingBuffer = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWordWrap(true)

	cv.userInput = cv.newChatInput()

	cv.messageList = textsel.NewTextSel()
	cv.messageList.
		SetScrollable(true).
		SetWordWrap(true)

	cv.messageList.SetSelectFunc(func(s string) {
		// Write to paste buffer
		copyText := stripTviewTags(strings.TrimSpace(s))
		clipboard.WriteAll(copyText)

		// Reset focus to chat input
		cv.FocusUserInput()
	})

	cv.chatFlex = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(cv.messageList, 0, 5, false).
		AddItem(cv.userInput, 0, 1, false)

	cv.container.AddItem(cv.chatFlex, 0, 1, false)

	cv.Frame = ui.newScreen(cv.container, screenArgs{
		title: "Chat: " + ui.Context.Config.Box,
		keys: []keyBinding{
			{"ctrl-space", "sends"},
			{"shift-tab", "switches focus"},
			{"space, enter", "select, copy (in msgs)"},
			{"ctrl-/", "help"},
			{"esc", "home"},
		},
	})

	cv.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			if cv.helpModalIsOpen {
				cv.toggleHelp()
				return nil
			}

			ui.OpenHome()
			return nil
		case tcell.KeyBacktab:
			if cv.ui.app.GetFocus() == cv.userInput {
				cv.FocusMessageList()
			} else {
				cv.FocusUserInput()
			}
			return nil
		// This is actually Ctrl-/
		case tcell.KeyCtrlUnderscore:
			cv.toggleHelp()
		}

		return event
	})

	return cv
}

func (cv *chatView) GetInitialFocus() tview.Primitive {
	return cv.userInput
}

func (cv *chatView) toggleHelp() {
	if cv.helpModalIsOpen {
		cv.helpModalIsOpen = false
		cv.container.RemoveItem(cv.helpModal)
		cv.container.AddItem(cv.chatFlex, 0, 1, false)
		cv.ui.app.SetFocus(cv.userInput)
	} else {
		cv.helpModalIsOpen = true
		cv.container.RemoveItem(cv.chatFlex)
		cv.container.AddItem(cv.helpModal, 0, 1, false)
		cv.ui.app.SetFocus(cv.helpModal)
	}
}

// Builds the chatInput component, which is a text area that captures user
// input and sends it to the assistant when the user presses Ctrl+Space.
func (cv *chatView) newChatInput() *tview.TextArea {
	chatInput := tview.NewTextArea()
	chatInput.SetBorder(true)
	chatInput.SetTitle("Type your message here")

	chatInput.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if chatInput.GetDisabled() {
			return event
		}

		if event.Key() == tcell.KeyCtrlSpace {
			go cv.onSubmit()

			// Return nil to indicate the event has been handled
			return nil
		}

		return event
	})

	return chatInput
}

func (cv *chatView) FocusUserInput() {
	cv.ui.app.SetFocus(cv.userInput)
	cv.messageList.ScrollToEnd()
	cv.messageList.ResetCursor()
}

func (cv *chatView) FocusMessageList() {
	cv.ui.app.SetFocus(cv.messageList)
	cv.messageList.MoveToLastLine()
}

func (cv *chatView) ToggleReceiving() {
	if cv.isReceiving {
		lastMessage := cv.chatMgr.LastMessage()
		if lastMessage != nil {
			cv.container.RemoveItem(cv.receivingBuffer)
			cv.container.AddItem(cv.chatFlex, 0, 1, false)
			cv.isReceiving = false

			cv.queueAppendText("[green::b]Assistant:[-:-:-]\n\n")
			cv.queueAppendText(cv.renderMarkdown(lastMessage.Content) + "\n")
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

func (cv *chatView) onSubmit() {
	// Disable the chat input while the assistant is responding
	cv.userInput.SetDisabled(true)

	var msgs []messages.Message
	messageText := cv.userInput.GetText()

	// Parse the user message
	for {
		parsed, err := chat_manager.ParseMessage(messages.You, messageText)

		if err == nil {
			msgs = parsed
			break
		}

		if messageFileDoesNotExist, ok := err.(*chat_manager.MessageFileDoesNotExist); ok {
			prompt := fmt.Sprintf("File '%s' not found! Please select the file you intended.", messageFileDoesNotExist.FilePath)

			done := make(chan bool)

			cv.ui.app.QueueUpdateDraw(func() {
				cv.ui.OpenFilePicker(prompt, ".", func(replacementFilePath string) {
					messageText = strings.Replace(messageText, "\\f "+messageFileDoesNotExist.FilePath, "\\f "+replacementFilePath, 1)
					cv.ui.OpenChat()
					done <- true
				})
			})

			<-done
		}
	}

	// Clear the chat input after the user has sent the message
	cv.userInput.SetText("", false)

	if len(msgs) == 0 {
		cv.userInput.SetDisabled(false)
		return
	}

	// Add the parsed user messages to the chat view and conversation.
	for _, msg := range msgs {
		if !msg.IsHidden {
			content := cv.renderMarkdown(msg.Content)
			cv.queueAppendText("[blue::b]You:[-:-:-]\n\n" + content + "\n")
			cv.chatMgr.AddMessage(msg)
			cv.messageList.ScrollToEnd()
			cv.messageList.MoveToLastLine()
		}
	}

	// Get the assistant's response
	cv.ToggleReceiving()
	cv.queueAppendText("[green::b]Assistant:[-:-:-]\n\n")
	cv.chatMgr.RequestResponse(func(chunk string) {
		// Append the assistant's response to the chat view
		cv.queueAppendText(chunk)
	})

	// Now that the response is complete, append a few newlines to separate it
	// from the next user message and scroll to the end of the chat view.
	cv.ToggleReceiving()

	// Re-enable the chat input
	cv.userInput.SetDisabled(false)
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

// renderMarkdown converts markdown to tview tags by first rendering the
// markdown as ANSI and then translating the ANSI to tview tags.
func (cv *chatView) renderMarkdown(content string) string {
	debug.Log("Rendering markdown:\n-----\n%s\n-----", content)

	// Render the markdown content as ANSI
	rendered := markdown.Render(content)

	debug.Log("Rendered:\n-----\n%s\n-----", rendered)

	// Translate the ANSI-escaped content to tview tags
	rendered = tview.TranslateANSI(rendered)

	// Glamour-rendered markdown is indented by two spaces, so we will remove
	// up to two spaces at the beginning of each line.
	leadingSpacesRe := regexp.MustCompile(`(?m)^ {1,2}`)
	rendered = leadingSpacesRe.ReplaceAllString(rendered, "")

	return rendered
}

// stripTviewTags removes tview tags from a string.
func stripTviewTags(input string) string {
	re := regexp.MustCompile(`\[[^\[\]]*\]`)
	return re.ReplaceAllString(input, "")
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
	case '┃':
		return "|"
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
