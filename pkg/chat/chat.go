package chat

import (
	"strings"

	"github.com/sysread/fnord/pkg/data"
	"github.com/sysread/fnord/pkg/debug"
	"github.com/sysread/fnord/pkg/gpt"
	"github.com/sysread/fnord/pkg/messages"
)

type Chat struct {
	gptClient    *gpt.OpenAIClient
	dataStore    *data.DataStore
	conversation *data.Conversation
}

func NewChat() *Chat {
	gptClient := gpt.NewOpenAIClient()
	dataStore := data.NewDataStore()
	conversation := dataStore.NewConversation()

	return &Chat{
		gptClient:    gptClient,
		dataStore:    dataStore,
		conversation: conversation,
	}
}

func (c *Chat) AddMessage(msg messages.ChatMessage) {
	c.conversation.AddMessage(msg)

	go func() {
		debug.Log("Saving conversation to disk")

		// If the message is from the assistant, update the conversation
		// summary. That way we are not generating both a new summary and
		// embedding on each and every message from the user. That is important
		// because the user's message input may be parsed into multiple
		// messages (e.g., for slash commands).
		if msg.From == messages.Assistant {
			// Update the summary of the conversation
			summary, _ := c.gptClient.GetSummary(c.conversation.Messages)
			debug.Log("Conversation summary: %s", summary)

			// Using the updated summary, generate a new embedding
			embedding, _ := c.gptClient.GetEmbedding(summary)

			// Store the updated summary and embedding in the struct
			c.conversation.SetSummary(summary, embedding)
		}

		// Save the conversation to disk
		c.conversation.Save()
	}()
}

func (c *Chat) RequestResponse(onChunkReceived func(string)) {
	done := make(chan bool)

	// Buffer to collect the streaming response
	var buf strings.Builder

	// Start the streaming response
	responseChan := c.gptClient.GetCompletionStream(c.conversation.Messages)

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
		msg := messages.NewMessage(messages.Assistant, buf.String())
		c.AddMessage(msg)

		done <- true
	}()

	<-done
}
