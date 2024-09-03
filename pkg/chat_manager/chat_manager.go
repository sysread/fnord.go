package chat_manager

import (
	"fmt"
	"strings"

	"github.com/sysread/fnord/pkg/data"
	"github.com/sysread/fnord/pkg/debug"
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
	*data.PersistedConversation
	fnord    *fnord.Fnord
	threadID string
}

// NewChatManager creates a new ChatManager instance.
func NewChatManager(fnord *fnord.Fnord) *ChatManager {
	pc := fnord.DataStore.NewPersistedConversation()

	return &ChatManager{
		PersistedConversation: pc,
		fnord:                 fnord,
	}
}

// AddMessage adds a message to the conversation and persists the conversation.
func (cm *ChatManager) AddMessage(msg messages.Message) {
	cm.PersistedConversation.AddMessage(msg)

	// Create the thread if it doesn't exist yet
	if cm.threadID == "" {
		threadID, err := cm.fnord.GptClient.CreateThread()
		if err != nil {
			debug.Log("Error creating thread: %v", err)
			return
		}

		cm.threadID = threadID
	}

	// Add user messages to the thread. Assistant messages are added
	// automatically during the thread run.
	if msg.IsUserMessage() {
		err := cm.fnord.GptClient.AddMessage(cm.threadID, msg.Content)
		if err != nil {
			debug.Log("Error adding message to thread: %v", err)
			return
		}
	}

	go func() {
		// If the message is from the assistant, update the conversation
		// summary. That way we are not generating both a new summary and
		// embedding on each and every message from the user. That is important
		// because the user's message input may be parsed into multiple
		// messages (e.g., for slash commands).
		if msg.From == messages.Assistant {
			// Update the summary of the conversation
			summary, _ := cm.GenerateSummary()

			// Using the updated summary, generate a new embedding
			embedding, _ := cm.fnord.GptClient.GetEmbedding(summary)

			// Store the updated summary and embedding in the struct
			cm.SetSummary(summary, embedding)
		}

		// Save the conversation to disk
		cm.Save()
	}()
}

// RequestResponse sends the user's input to the assistant and processes the
// response.
func (cm *ChatManager) RequestResponse(onChunkReceived func(string)) {
	done := make(chan bool)

	// Add summaries of prior conversations that could help inform the current
	// conversation. This is done before starting the streaming response so
	// that the assistant can use the summaries to improve its responses.
	related, err := cm.Search(cm.ChatTranscript(), 5)
	if err != nil {
		debug.Log("Error searching for related conversations: %v", err)
	} else {
		cm.AddMessage(messages.NewMessage(messages.You, "Summaries of related past conversations", true))
		for _, conversation := range related {
			content := fmt.Sprintf("Conversation occurring between %v and %v:\n\n%s", conversation.Created, conversation.Modified, conversation.Summary)
			cm.AddMessage(messages.NewMessage(messages.You, content, true))
		}
	}

	// Buffer to collect the streaming response
	var buf strings.Builder

	// Start the streaming response
	responseChan, err := cm.fnord.GptClient.RunThread(cm.threadID)
	if err != nil {
		debug.Log("Error starting response stream: %v", err)
		return
	}

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
	}()

	<-done
}

// Generates a summary of the conversation transcript using the fast model.
func (cm *ChatManager) GenerateSummary() (string, error) {
	userPrompt := cm.ChatTranscript()
	return cm.fnord.GptClient.GetCompletion(systemSummaryPrompt, userPrompt)
}

// Takes a user's prompt message, uses the fast model to generate a search
// query from it, converts that to an embedding, and then searches the
// conversation directory for the most similar conversations.
func (cm *ChatManager) Search(queryString string, numResults int) ([]data.ConversationIndexEntry, error) {
	// Generate a search query from the user input
	query, err := cm.fnord.GptClient.GetCompletion(searchQueryPrompt, queryString)
	if err != nil {
		return nil, err
	}

	// Generate an embedding for the newly generated search query
	embedding, err := cm.fnord.GptClient.GetEmbedding(query)
	if err != nil {
		return nil, err
	}

	return cm.fnord.DataStore.Search(embedding, numResults)
}
