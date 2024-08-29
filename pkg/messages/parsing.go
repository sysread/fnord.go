package messages

import (
	"fmt"
	"strings"

<<<<<<<< HEAD:pkg/messages/parsing.go
	"github.com/rivo/tview"
|||||||| parent of 1b81d3d (Simplify conversation data model a little bit):pkg/messages/messages.go
	"github.com/rivo/tview"
	openai "github.com/sashabaranov/go-openai"
========
	openai "github.com/sashabaranov/go-openai"
>>>>>>>> 1b81d3d (Simplify conversation data model a little bit):pkg/messages/messages.go
)

<<<<<<<< HEAD:pkg/messages/parsing.go
// trimMessage trims leading and trailing whitespace from a message's content.
func trimMessage(content string) string {
	content = strings.TrimLeft(content, " \r\n\t\f")
	content = strings.TrimRight(content, " \r\n\t\f")
	return content
}
|||||||| parent of 1b81d3d (Simplify conversation data model a little bit):pkg/messages/messages.go
type Sender string

type ChatMessage struct {
	From     Sender `json:"from"`
	IsHidden bool   `json:"is_hidden"`
	Content  string `json:"content"`
}
========
// OpenAI has a token limit per request. When a file is too large to send as
// part of a conversation message, we can split it up into smaller chunks. 30k
// is a safe limit, lower than needed to avoid hitting the token limit.
const MaxChunkSize = 30_000

//------------------------------------------------------------------------------
// MessageFileDoesNotExist
//------------------------------------------------------------------------------
>>>>>>>> 1b81d3d (Simplify conversation data model a little bit):pkg/messages/messages.go

<<<<<<<< HEAD:pkg/messages/parsing.go
func ParseMessage(from Sender, content string) ([]Message, error) {
	messages := []Message{}
	scanner := bufio.NewScanner(strings.NewReader(content))
|||||||| parent of 1b81d3d (Simplify conversation data model a little bit):pkg/messages/messages.go
type Conversation []ChatMessage

type FileDoesNotExist struct {
	FilePath string
}

const (
	You       Sender = "You"
	Assistant Sender = "Assistant"
	System    Sender = "System"

	// OpenAI has a token limit per request. When a file is too large to send
	// as part of a conversation message, we can split it up into smaller
	// chunks. 30k is a safe limit, lower than needed to avoid hitting the
	// token limit.
	MaxChunkSize = 30_000
)

func (e *FileDoesNotExist) Error() string {
	return fmt.Sprintf("file does not exist: %s", e.FilePath)
}

func NewMessage(from Sender, content string) ChatMessage {
	return ChatMessage{
		From:     from,
		Content:  content,
		IsHidden: false,
	}
}

func (c Conversation) ChatCompletionMessages() []openai.ChatCompletionMessage {
	messages := []openai.ChatCompletionMessage{}

	for _, message := range c {
		messages = append(messages, message.ChatCompletionMessage())
	}

	return messages
}

func (c Conversation) ChatTranscript() string {
	transcript := ""

	for _, message := range c {
		if message.From != System {
			transcript += fmt.Sprintf("%s: %s\n\n", message.From, message.Content)
		}
	}

	return transcript
}

func (c *Conversation) AddMessage(message ChatMessage) {
	*c = append(*c, message)
}

func (m ChatMessage) Role() string {
	role := openai.ChatMessageRoleUser

	if m.From != You {
		role = openai.ChatMessageRoleAssistant
	}

	return role
}

func (m ChatMessage) ChatCompletionMessage() openai.ChatCompletionMessage {
	return openai.ChatCompletionMessage{
		Role:    m.Role(),
		Content: trimMessage(m.Content),
	}
}

func ParseMessage(from Sender, content string) (Conversation, error) {
	messages := Conversation{}
	scanner := bufio.NewScanner(strings.NewReader(content))
========
// MessageFileDoesNotExist is an error type that is returned when a file
// referenced in a slash command (e.g., `\f`) does not exist.
type MessageFileDoesNotExist struct {
	FilePath string
}

// Error returns the error message for a MessageFileDoesNotExist error.
func (e *MessageFileDoesNotExist) Error() string {
	return fmt.Sprintf("file does not exist: %s", e.FilePath)
}

//------------------------------------------------------------------------------
// Sender
//------------------------------------------------------------------------------

const (
	// `You` represents the user.
	You Sender = "You"

	// `Assistant` represents the assistant.
	Assistant Sender = "Assistant"

	// `System` represents a system message.
	System Sender = "System"
)

// Sender represents the sender of a message in a conversation.
type Sender string

//------------------------------------------------------------------------------
// Message
//------------------------------------------------------------------------------

// Message represents an individual message in a conversation.
type Message struct {
	// The message sender.
	//
	// If a message is from the user, the sender is "You", and its content will
	// be displayed if `IsHidden` is false.
	//
	// If a message is from the assistant, the sender is "Assistant", and its
	// content will always be displayed.
	//
	// If a message is a system message, the sender is "System", and its
	// content will never be displayed.
	From Sender `json:"from"`

	// Indicates whether a *user* message's *content* should be displayed in
	// the chat window.
	IsHidden bool `json:"is_hidden"`

	// A message's raw, unformatted content, as it was entered by the user or
	// returned from the assistant.
	Content string `json:"content"`
}

func NewMessage(from Sender, content string, isHidden bool) Message {
	return Message{
		From:     from,
		Content:  trimMessage(content),
		IsHidden: false,
	}
}

// Returns the openai Role of the message.
func (m Message) Role() string {
	switch m.From {
	case You:
		return openai.ChatMessageRoleUser
	case Assistant:
		return openai.ChatMessageRoleAssistant
	case System:
		fallthrough
	default:
		return openai.ChatMessageRoleUser
	}
}

// ChatCompletionMessage returns the message as an
// `openai.ChatCompletionMessage`.
func (m Message) ToChatCompletionMessage() openai.ChatCompletionMessage {
	return openai.ChatCompletionMessage{
		Role:    m.Role(),
		Content: trimMessage(m.Content),
	}
}

//------------------------------------------------------------------------------
// Conversation
//------------------------------------------------------------------------------
>>>>>>>> 1b81d3d (Simplify conversation data model a little bit):pkg/messages/messages.go

// Conversation represents a list of messages, in order.
type Conversation struct {
	Messages []Message
}

<<<<<<<< HEAD:pkg/messages/parsing.go
		isAction, action, remaining := getAction(line)
		if isAction {
			// Any built up text is a message. Add it and reset the current
			// message buffer.
			currentMessage = trimMessage(currentMessage)
			if currentMessage != "" {
				message := Message{
					From:     from,
					Content:  currentMessage,
					IsHidden: false,
				}
|||||||| parent of 1b81d3d (Simplify conversation data model a little bit):pkg/messages/messages.go
		isAction, action, remaining := getAction(line)
		if isAction {
			// Any built up text is a message. Add it and reset the current
			// message buffer.
			currentMessage = trimMessage(currentMessage)
			if currentMessage != "" {
				message := ChatMessage{
					From:     from,
					Content:  currentMessage,
					IsHidden: false,
				}
========
// NewConversation creates a new conversation.
func NewConversation() *Conversation {
	return &Conversation{
		Messages: []Message{},
	}
}
>>>>>>>> 1b81d3d (Simplify conversation data model a little bit):pkg/messages/messages.go

// AddMessage adds a message to the conversation.
func (c *Conversation) AddMessage(msg Message) {
	c.Messages = append(c.Messages, msg)
}

<<<<<<<< HEAD:pkg/messages/parsing.go
			// Now process the action
			switch action {
			case "file":
				if _, err := os.Stat(remaining); os.IsNotExist(err) {
					return messages, &MessageFileDoesNotExist{FilePath: remaining}
				}
|||||||| parent of 1b81d3d (Simplify conversation data model a little bit):pkg/messages/messages.go
			// Now process the action
			switch action {
			case "file":
				if _, err := os.Stat(remaining); os.IsNotExist(err) {
					return messages, &FileDoesNotExist{FilePath: remaining}
				}
========
// LastMessage returns the last message in the conversation.
func (c *Conversation) LastMessage() *Message {
	if len(c.Messages) == 0 {
		return nil
	}
>>>>>>>> 1b81d3d (Simplify conversation data model a little bit):pkg/messages/messages.go

<<<<<<<< HEAD:pkg/messages/parsing.go
				messages = append(messages, Message{
					From:     from,
					Content:  fmt.Sprintf("Attached file: %s", remaining),
					IsHidden: false,
				})
|||||||| parent of 1b81d3d (Simplify conversation data model a little bit):pkg/messages/messages.go
				messages = append(messages, ChatMessage{
					From:     from,
					Content:  fmt.Sprintf("Attached file: %s", remaining),
					IsHidden: false,
				})
========
	return &c.Messages[len(c.Messages)-1]
}
>>>>>>>> 1b81d3d (Simplify conversation data model a little bit):pkg/messages/messages.go

// ChatCompletionMessages returns the conversation's messages as a slice of
// OpenAI chat completion messages, appropriate to be used as an argument to
// `openai.CreateChatCompletion`.
func (c *Conversation) ChatCompletionMessages() []openai.ChatCompletionMessage {
	completionMessages := make([]openai.ChatCompletionMessage, len(c.Messages))

<<<<<<<< HEAD:pkg/messages/parsing.go
				for idx, part := range chunks {
					message := Message{
						From:     from,
						Content:  fmt.Sprintf("Attached file (%s) part %d:\n\n%s", remaining, idx, part),
						IsHidden: true,
					}
|||||||| parent of 1b81d3d (Simplify conversation data model a little bit):pkg/messages/messages.go
				for idx, part := range chunks {
					message := ChatMessage{
						From:     from,
						Content:  fmt.Sprintf("Attached file (%s) part %d:\n\n%s", remaining, idx, part),
						IsHidden: true,
					}
========
	for i, message := range c.Messages {
		completionMessages[i] = message.ToChatCompletionMessage()
	}
>>>>>>>> 1b81d3d (Simplify conversation data model a little bit):pkg/messages/messages.go

	return completionMessages
}

<<<<<<<< HEAD:pkg/messages/parsing.go
			case "exec":
				messages = append(messages, Message{
					From:     from,
					Content:  fmt.Sprintf("Executed command: %s", remaining),
					IsHidden: false,
				})
|||||||| parent of 1b81d3d (Simplify conversation data model a little bit):pkg/messages/messages.go
			case "exec":
				messages = append(messages, ChatMessage{
					From:     from,
					Content:  fmt.Sprintf("Executed command: %s", remaining),
					IsHidden: false,
				})
========
// ChatTranscript returns the conversation's messages as a formatted string
// representation of the chat.
func (c *Conversation) ChatTranscript() string {
	var buf strings.Builder
>>>>>>>> 1b81d3d (Simplify conversation data model a little bit):pkg/messages/messages.go

<<<<<<<< HEAD:pkg/messages/parsing.go
				chunks := splitExecOutputIntoDigestibleChunks(remaining)

				// We want to show the output of the command, so it's not hidden in the UI.
				for idx, part := range chunks {
					message := Message{
						From:     from,
						Content:  fmt.Sprintf("Attached command output (%s) part %d:\n\n%s", remaining, idx, part),
						IsHidden: false,
					}

					messages = append(messages, message)
				}

			default:
				fmt.Printf("unknown action: %s", action)
			}

			continue
|||||||| parent of 1b81d3d (Simplify conversation data model a little bit):pkg/messages/messages.go
				chunks := splitExecOutputIntoDigestibleChunks(remaining)

				// We want to show the output of the command, so it's not hidden in the UI.
				for idx, part := range chunks {
					message := ChatMessage{
						From:     from,
						Content:  fmt.Sprintf("Attached command output (%s) part %d:\n\n%s", remaining, idx, part),
						IsHidden: false,
					}

					messages = append(messages, message)
				}

			default:
				fmt.Printf("unknown action: %s", action)
			}

			continue
========
	for _, message := range c.Messages {
		if message.From != System {
			buf.WriteString(fmt.Sprintf("%s: %s\n\n", message.From, message.Content))
>>>>>>>> 1b81d3d (Simplify conversation data model a little bit):pkg/messages/messages.go
		}
	}

<<<<<<<< HEAD:pkg/messages/parsing.go
	// Add any remaining message content
	currentMessage = trimMessage(currentMessage)
	if currentMessage != "" {
		message := Message{
			From:     from,
			Content:  currentMessage,
			IsHidden: false,
		}

		messages = append(messages, message)
		currentMessage = ""
	}

	// Check for errors during scanning
	if err := scanner.Err(); err != nil {
		return messages, err
	}

	return messages, nil
}

// getAction checks if the line is an action (e.g. a slash command indicating a
// file or exec command) and returns the action type and the remaining content
// of the line.
func getAction(line string) (bool, string, string) {
	if strings.HasPrefix(line, "\\f ") {
		return true, "file", strings.TrimPrefix(line, "\\f ")
	}

	if strings.HasPrefix(line, "\\x ") {
		return true, "exec", strings.TrimPrefix(line, "\\x ")
	}

	return false, "", line
}

// Because OpenAI does not support all of the file types that we might care
// about in the line of battle, we send them as part of the conversation
// message. For larger files, this requires splitting the file into smaller
// chunks that can be sent as part of the conversation.
func splitFileIntoDigestibleChunks(filePath string) []string {
	file, err := os.Open(filePath)

	if err != nil {
		return []string{}
	}

	defer file.Close()

	return splitIntoDigestibleChunks(bufio.NewScanner(file))
}

// splitExecOutputIntoDigestibleChunks executes a command and splits the output
// into openai-sized chunks.
func splitExecOutputIntoDigestibleChunks(command string) []string {
	cmd := exec.Command("sh", "-c", command)
	output, err := cmd.CombinedOutput()

	// An error here is not fatal. It's part of the output.
	if err != nil {
		errorMsg := fmt.Sprintf("Failed to execute command: %v", err)
		return []string{errorMsg}
	}

	// Convert any ANSI escape codes to tview tags
	outputStr := tview.TranslateANSI(string(output))

	return splitIntoDigestibleChunks(bufio.NewScanner(strings.NewReader(outputStr)))
}

// splitIntoDigestibleChunks splits the input into chunks that are smaller than
// the OpenAI token limit.
func splitIntoDigestibleChunks(scanner *bufio.Scanner) []string {
	parts := []string{}

	scanner.Split(bufio.ScanRunes)

	var buffer strings.Builder
	currentSize := 0

	for scanner.Scan() {
		runeText := scanner.Text()
		runeSize := len([]byte(runeText))

		// If adding this rune exceeds the max chunk size, start a new chunk
		if currentSize+runeSize > MaxChunkSize {
			parts = append(parts, buffer.String())
			buffer.Reset()
			currentSize = 0
		}

		buffer.WriteString(runeText)
		currentSize += runeSize
	}

	// Add any remaining runes to the final chunk
	if buffer.Len() > 0 {
		parts = append(parts, buffer.String())
	}

	return parts
|||||||| parent of 1b81d3d (Simplify conversation data model a little bit):pkg/messages/messages.go
	// Add any remaining message content
	currentMessage = trimMessage(currentMessage)
	if currentMessage != "" {
		message := ChatMessage{
			From:     from,
			Content:  currentMessage,
			IsHidden: false,
		}

		messages = append(messages, message)
		currentMessage = ""
	}

	// Check for errors during scanning
	if err := scanner.Err(); err != nil {
		return messages, err
	}

	return messages, nil
}

func getAction(line string) (bool, string, string) {
	if strings.HasPrefix(line, "\\f ") {
		return true, "file", strings.TrimPrefix(line, "\\f ")
	}

	if strings.HasPrefix(line, "\\x ") {
		return true, "exec", strings.TrimPrefix(line, "\\x ")
	}

	return false, "", line
}

func trimMessage(msg string) string {
	msg = strings.TrimLeft(msg, " \r\n\t\f")
	msg = strings.TrimRight(msg, " \r\n\t\f")
	return msg
}

// Because OpenAI does not support all of the file types that we might care
// about in the line of battle, we send them as part of the conversation
// message. For larger files, this requires splitting the file into smaller
// chunks that can be sent as part of the conversation.
func splitFileIntoDigestibleChunks(filePath string) []string {
	file, err := os.Open(filePath)

	if err != nil {
		return []string{}
	}

	defer file.Close()

	return splitIntoDigestibleChunks(bufio.NewScanner(file))
}

func splitExecOutputIntoDigestibleChunks(command string) []string {
	cmd := exec.Command("sh", "-c", command)
	output, err := cmd.CombinedOutput()

	// An error here is not fatal. It's part of the output.
	if err != nil {
		errorMsg := fmt.Sprintf("Failed to execute command: %v", err)
		return []string{errorMsg}
	}

	// Convert any ANSI escape codes to tview tags
	outputStr := tview.TranslateANSI(string(output))

	return splitIntoDigestibleChunks(bufio.NewScanner(strings.NewReader(outputStr)))
}

func splitIntoDigestibleChunks(scanner *bufio.Scanner) []string {
	parts := []string{}

	scanner.Split(bufio.ScanRunes)

	var buffer strings.Builder
	currentSize := 0

	for scanner.Scan() {
		runeText := scanner.Text()
		runeSize := len([]byte(runeText))

		// If adding this rune exceeds the max chunk size, start a new chunk
		if currentSize+runeSize > MaxChunkSize {
			parts = append(parts, buffer.String())
			buffer.Reset()
			currentSize = 0
		}

		buffer.WriteString(runeText)
		currentSize += runeSize
	}

	// Add any remaining runes to the final chunk
	if buffer.Len() > 0 {
		parts = append(parts, buffer.String())
	}

	return parts
========
	return buf.String()
>>>>>>>> 1b81d3d (Simplify conversation data model a little bit):pkg/messages/messages.go
}
