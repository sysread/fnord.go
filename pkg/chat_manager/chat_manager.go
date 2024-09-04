package chat_manager

import (
	"fmt"
	"strings"

	"github.com/sysread/fnord/pkg/fnord"
	"github.com/sysread/fnord/pkg/messages"
)

const systemSummaryPrompt = `
Your job is to summarize a conversation.
It is essential that you identify all significant facts in the conversation transcript.
You will assemble a nested an outline of this conversation in markdown format.
If there is file content present, be sure to include the file path and an individual summary of the file content as a distinct set of nested list items.
If there is command output, include the command, a the relevance of its output, and then VERY tersely summarize how it relates to the conversation.
If a problem was solved in the conversation, DEFINITELY include the details of the problem and solution, as well as the steps taken to troubleshoot.
Respond ONLY with a summary of the discussion, followed by your outline of ALL facts identified in the conversation.
`

const searchQueryPrompt = `
Take the user input and respond ONLY with a very short query string to use RAG to identify matching entries.
`

// ChatManager manages a conversation and provides methods for interacting with
// the conversation.
type ChatManager struct {
	*messages.Conversation
	fnord    *fnord.Fnord
	threadID string
	vectorID string
}

// NewChatManager creates a new ChatManager instance.
func NewChatManager(fnord *fnord.Fnord) *ChatManager {
	return &ChatManager{
		Conversation: messages.NewConversation(),
		fnord:        fnord,
	}
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

	// If the conversation has no vector ID, create, otherwise, update the
	// conversation on disk.
	if cm.vectorID == "" {
		vectorID, err := cm.fnord.Storage.Create(cm.ChatTranscript())
		if err != nil {
			panic(fmt.Sprintf("Error creating conversation: %#v", err))
		}

		cm.vectorID = vectorID
	} else {
		err := cm.fnord.Storage.Update(cm.vectorID, cm.ChatTranscript())
		if err != nil {
			panic(fmt.Sprintf("Error updating conversation: %#v", err))
		}
	}
}

// RequestResponse sends the user's input to the assistant and processes the
// response.
func (cm *ChatManager) RequestResponse(onChunkReceived func(string)) {
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

			// Append the chunk to the buffer
			buf.WriteString(chunk)

			// Send the chunk to the caller-supplied callback function
			onChunkReceived(chunk)
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

// Generates a summary of the conversation transcript using the fast model.
func (cm *ChatManager) GenerateSummary() (string, error) {
	userPrompt := cm.ChatTranscript()
	return cm.fnord.GptClient.GetCompletion(systemSummaryPrompt, userPrompt)
}
