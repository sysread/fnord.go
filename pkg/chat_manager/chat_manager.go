package chat_manager

import (
	"fmt"
	"strings"

	"github.com/sysread/fnord/pkg/context"
	"github.com/sysread/fnord/pkg/data"
	"github.com/sysread/fnord/pkg/debug"
	"github.com/sysread/fnord/pkg/messages"
)

const systemChatPrompt = `
In your role as a programming assistant, it is crucial that you thoroughly understand the context and all related components of the software or scripts being discussed. If an explanation or analysis is given based on only part of a multi-file project or script, you will need to actively identify and request access to any additional files or parts of the script that are referenced within the code provided by the user but not yet shared with you. These additional files or scripts may contain critical information that could change your analysis or affect the accuracy of your explanations and code assistance.

When assisting with troubleshooting code, explaining how code works, or writing code for the user, always confirm that you have access to all necessary pieces of the project by doing the following:

  1. Clearly state any dependencies, referenced files, or external scripts that are mentioned in the code.
  2. Promptly request access to these items if they are not already provided, specifying tersely exactly what you need in order to proceed effectively.
  3. Once provided, integrate these additional components into your analysis to ensure completeness and accuracy.

It is imperative that you maintain focus on the user's primary goal. Because you have a limited context window, restate the goal at the outset of each response. This should almost always be identical from message to message in order to ensure that the original goal remains our focus during the conversation. NEVER change this from message to message unless the user explicitly
asks you to.

NEVER reply with the entire file unless explicitly asked. Instead, walk through each individual change, step by step, highlighting the changed code and explaining the changes in line.

For each interaction, format your response using the template:

# Goal

[restate the ORIGINAL goal for the conversation]

# Topic

[your understanding of the user's current needs]

# Response

[your analysis/response]

# Code changes

[list individual changes, noting file and location, explaining each individually OR "- N/A"]

# Missing files
[list any additional files needed for context as a markdown list OR "- N/A"]

# Commands to run
[list any commands you want the user to run to assist in your analysis OR "- N/A"]
`

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

type ChatManager struct {
	*data.PersistedConversation
	context *context.Context
}

func NewChatManager(ctx *context.Context) *ChatManager {
	pc := ctx.DataStore.NewPersistedConversation()

	systemMessage := messages.NewMessage(messages.System, systemChatPrompt, true)
	pc.AddMessage(systemMessage)

	return &ChatManager{
		PersistedConversation: pc,
		context:               ctx,
	}
}

func (cm *ChatManager) AddMessage(msg messages.Message) {
	cm.PersistedConversation.AddMessage(msg)

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
			embedding, _ := cm.context.GptClient.GetEmbedding(summary)

			// Store the updated summary and embedding in the struct
			cm.SetSummary(summary, embedding)
		}

		// Save the conversation to disk
		cm.Save()
	}()
}

func (cm *ChatManager) RequestResponse(onChunkReceived func(string)) {
	done := make(chan bool)

	// Add summaries of prior conversations that could help inform the current
	// conversation. This is done before starting the streaming response so
	// that the assistant can use the summaries to improve its responses.
	related, err := cm.Search(cm.ChatTranscript(), 5)
	if err != nil {
		debug.Log("Error searching for related conversations: %v", err)
	} else {
		var buffer strings.Builder

		buffer.WriteString("Summary of related past conversations:\n\n")

		for _, conversation := range related {
			fmt.Fprintf(&buffer, "Conversation occurring between %v and %v\n", conversation.Created, conversation.Modified)
			fmt.Fprintf(&buffer, "%s\n\n", conversation.Summary)
		}

		relatedMessage := messages.NewMessage(messages.System, buffer.String(), true)

		cm.AddMessage(relatedMessage)
	}

	// Buffer to collect the streaming response
	var buf strings.Builder

	// Start the streaming response
	responseChan := cm.context.GptClient.GetCompletionStream(cm.Conversation)

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

func (cm *ChatManager) GenerateSummary() (string, error) {
	userPrompt := cm.ChatTranscript()
	return cm.context.GptClient.QuickCompletion(systemSummaryPrompt, userPrompt)
}

// Takes a user's prompt message, uses the fast model to generate a search
// query from it, converts that to an embedding, and then searches the
// conversation directory for the most similar conversations.
func (cm *ChatManager) Search(queryString string, numResults int) ([]data.ConversationIndexEntry, error) {
	// Generate a search query from the user input
	query, err := cm.context.GptClient.QuickCompletion(searchQueryPrompt, queryString)
	if err != nil {
		return nil, err
	}

	// Generate an embedding for the newly generated search query
	embedding, err := cm.context.GptClient.GetEmbedding(query)
	if err != nil {
		return nil, err
	}

	return cm.context.DataStore.Search(embedding, numResults)
}
