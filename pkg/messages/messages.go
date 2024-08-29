package messages

import (
	"fmt"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

// OpenAI has a token limit per request. When a file is too large to send as
// part of a conversation message, we can split it up into smaller chunks. 30k
// is a safe limit, lower than needed to avoid hitting the token limit.
const MaxChunkSize = 30_000

//------------------------------------------------------------------------------
// MessageFileDoesNotExist
//------------------------------------------------------------------------------

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

// Conversation represents a list of messages, in order.
type Conversation struct {
	Messages []Message
}

// NewConversation creates a new conversation.
func NewConversation() *Conversation {
	return &Conversation{
		Messages: []Message{},
	}
}

// AddMessage adds a message to the conversation.
func (c *Conversation) AddMessage(msg Message) {
	c.Messages = append(c.Messages, msg)
}

// LastMessage returns the last message in the conversation.
func (c *Conversation) LastMessage() *Message {
	if len(c.Messages) == 0 {
		return nil
	}

	return &c.Messages[len(c.Messages)-1]
}

// ChatCompletionMessages returns the conversation's messages as a slice of
// OpenAI chat completion messages, appropriate to be used as an argument to
// `openai.CreateChatCompletion`.
func (c *Conversation) ChatCompletionMessages() []openai.ChatCompletionMessage {
	completionMessages := make([]openai.ChatCompletionMessage, len(c.Messages))

	for i, message := range c.Messages {
		completionMessages[i] = message.ToChatCompletionMessage()
	}

	return completionMessages
}

// ChatTranscript returns the conversation's messages as a formatted string
// representation of the chat.
func (c *Conversation) ChatTranscript() string {
	var buf strings.Builder

	for _, message := range c.Messages {
		if message.From != System {
			buf.WriteString(fmt.Sprintf("%s: %s\n\n", message.From, message.Content))
		}
	}

	return buf.String()
}
