package common

type Sender string

const (
	You       Sender = "You"
	Assistant Sender = "Assistant"
)

type ChatMessage struct {
	From    Sender
	Content string
}
