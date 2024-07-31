package gpt

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

type Sender string

const (
	You       Sender = "You"
	Assistant Sender = "Assistant"

	// OpenAI has a token limit per request. When a file is too large to send
	// as part of a conversation message, we can split it up into smaller
	// chunks. 30k is a safe limit, lower than needed to avoid hitting the
	// token limit.
	MaxChunkSize = 30_000
)

type ChatMessage struct {
	From     Sender
	IsHidden bool
	Content  string
}

type Conversation []ChatMessage

func (c Conversation) ChatCompletionMessages() []openai.ChatCompletionMessage {
	messages := []openai.ChatCompletionMessage{}

	for _, message := range c {
		messages = append(messages, message.ChatCompletionMessage())
	}

	return messages
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

	currentMessage := ""
	for scanner.Scan() {
		line := scanner.Text()

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

				messages = append(messages, message)
				currentMessage = ""
			}

			// Now process the action
			switch action {
			case "file":
				// TODO if file doesn't exist, display file picker?
				messages = append(messages, ChatMessage{
					From:     from,
					Content:  fmt.Sprintf("Attached file: %s", remaining),
					IsHidden: false,
				})

				chunks := splitFileIntoDigestibleChunks(remaining)

				for idx, part := range chunks {
					message := ChatMessage{
						From:     from,
						Content:  fmt.Sprintf("Attached file (%s) part %d:\n\n%s", remaining, idx, part),
						IsHidden: true,
					}

					messages = append(messages, message)
				}

			case "exec":
				messages = append(messages, ChatMessage{
					From:     from,
					Content:  fmt.Sprintf("Executed command: %s", remaining),
					IsHidden: false,
				})

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
		}

		currentMessage += line + "\n"
	}

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

	// TODO display file picker instead?
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

	return splitIntoDigestibleChunks(bufio.NewScanner(strings.NewReader(string(output))))
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
}
