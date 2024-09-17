package chat_manager

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/sysread/fnord/pkg/debug"
	"github.com/sysread/fnord/pkg/fnord"
	"github.com/sysread/fnord/pkg/messages"
	"github.com/sysread/fnord/pkg/storage"
)

// ChatManager manages a conversation and provides methods for interacting with
// the conversation.
type ChatManager struct {
	*messages.Conversation
	fnord    *fnord.Fnord
	threadID string
}

// NewChatManager creates a new ChatManager instance.
func NewChatManager(fnord *fnord.Fnord) *ChatManager {
	cm := &ChatManager{
		Conversation: messages.NewConversation(),
		fnord:        fnord,
	}

	if fnord.Config.ProjectPath != "" {
		debug.Log("Informing the assistant that %s is the selected project", fnord.Config.ProjectPath)

		// Start the new thread off by explaining that the project is selected
		// and available via the `query_project_files` tool.
		msg := messages.NewMessage(messages.You,
			fmt.Sprintf("The project at `%s` is visible to you. Use the `query_project_files` tool as needed to search its contents.", fnord.Config.ProjectPath),
			false,
		)

		cm.AddMessage(msg)
	}

	return cm
}

// AddMessage adds a message to the conversation and persists the conversation.
func (cm *ChatManager) AddMessage(msg messages.Message) {
	cm.Conversation.AddMessage(msg)

	// Create the thread if it doesn't exist yet
	if cm.threadID == "" {
		threadID, err := cm.fnord.GptClient.CreateThread()
		if err != nil {
			panic(fmt.Sprintf("Error creating thread: %#v", err))
		}

		cm.threadID = threadID
	}

	// Add user messages to the thread. Assistant messages are added
	// automatically during the thread run.
	if msg.IsUserMessage() {
		err := cm.fnord.GptClient.AddMessage(cm.threadID, msg.Content)
		if err != nil {
			panic(fmt.Sprintf("Error adding message to thread: %#v", err))
		}
	}

	// Store the conversation transcript
	err := storage.UpdateConversation(cm.threadID, cm.ChatTranscript())
	if err != nil {
		panic(fmt.Sprintf("Error updating conversation: %#v", err))
	}
}

// RequestResponse sends the user's input to the assistant and processes the
// response.
func (cm *ChatManager) RequestResponse(onChunkReceived, onStatusReceived func(string)) {
	done := make(chan bool)

	// Buffer to collect the streaming response
	var buf strings.Builder

	// Channel to receive the streaming response
	responseChan := make(chan string)

	// Start a goroutine to collect the streaming response and send it to the
	// caller-supplied callback function.
	go func() {
		// Collect the streaming response
		for chunk := range responseChan {
			statusRe := regexp.MustCompile(`STATUS:\s*(.*)`)

			if statusRe.MatchString(chunk) {
				status := statusRe.FindStringSubmatch(chunk)[1]
				onStatusReceived(status)
			} else {
				// Append the chunk to the buffer
				buf.WriteString(chunk)

				// Send the chunk to the caller-supplied callback function
				onChunkReceived(chunk)
			}
		}

		// Finally, add the full response to the conversation. This will
		// trigger the conversation summary to be updated.
		msg := messages.NewMessage(messages.Assistant, buf.String(), false)
		cm.AddMessage(msg)

		done <- true
		close(done)
	}()

	// Start the streaming response producer
	go cm.fnord.GptClient.RunThread(cm.threadID, responseChan)

	<-done
}
