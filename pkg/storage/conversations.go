package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/philippgille/chromem-go"

	"github.com/sysread/fnord/pkg/config"
	"github.com/sysread/fnord/pkg/debug"
)

// Conversations is the chromem collection of conversation data
var Conversations *chromem.Collection

// InitializeConversationsCollection initializes the conversations collection in the chromem database.
func InitializeConversationsCollection(config *config.Config) error {
	debug.Log("Initializing conversations collection conversation:%s", config.Box)
	var err error
	collectionName := "conversations:" + config.Box
	Conversations, err = DB.GetOrCreateCollection(collectionName, nil, nil)
	return err
}

// CreateConversation stores a new conversation and returns an error if the operation fails.
func CreateConversation(threadID string, content string) error {
	debug.Log("Creating conversation %s", threadID)

	now := time.Now().Format(time.RFC3339)
	document := chromem.Document{
		ID:      threadID,
		Content: content,
		Metadata: map[string]string{
			"created": now,
			"updated": now,
		},
	}

	return Conversations.AddDocuments(context.Background(), []chromem.Document{document}, 1)
}

// ReadConversation retrieves a conversation by thread ID.
func ReadConversation(threadID string) (string, error) {
	debug.Log("Reading conversation %s", threadID)

	document, err := Conversations.GetByID(context.Background(), threadID)
	if err != nil {
		debug.Log("Conversation not found: %s", threadID)
		return "", err
	}

	return document.Content, nil
}

// UpdateConversation modifies the content of an existing conversation.
func UpdateConversation(threadID, content string) error {
	debug.Log("Updating conversation %s", threadID)

	// Find the existing entry
	existingEntry, err := Conversations.GetByID(context.Background(), threadID)
	if err != nil {
		debug.Log("Conversation not found; creating instead: %s", threadID)
		return CreateConversation(threadID, content)
	}

	// Update the content
	existingEntry.Content = content

	// Set the updated date
	existingEntry.Metadata["updated"] = time.Now().Format(time.RFC3339)

	// Preserve the original creation date, or set it to the updated date if it
	// was missing.
	if existingEntry.Metadata["created"] == "" {
		existingEntry.Metadata["created"] = existingEntry.Metadata["updated"]
	}

	// Save the updated entry
	err = Conversations.AddDocuments(context.Background(), []chromem.Document{existingEntry}, 1)
	if err != nil {
		debug.Log("Failed to update conversation: %s", threadID)
		return err
	}

	debug.Log("Updated conversation %s", threadID)

	return nil
}

// DeleteConversation removes a conversation by thread ID.
func DeleteConversation(threadID string) error {
	debug.Log("Deleting conversation %s", threadID)

	err := Conversations.Delete(context.Background(), nil, nil, threadID)
	if err != nil {
		debug.Log("Failed to delete conversation: %s", threadID)
		return err
	}

	return nil
}

// SearchConversations queries the conversation collection for a given query
// string and returns a slice of search results.
func SearchConversations(query string, numResults int) ([]Result, error) {
	debug.Log("Searching conversations for %d results using query '%s'", numResults, query)
	maxResults := Conversations.Count()
	if numResults > maxResults {
		numResults = maxResults
	}

	if numResults == 0 {
		debug.Log("No indexed conversations to search!")
		return []Result{}, nil
	}

	results, err := Conversations.Query(context.Background(), query, numResults, nil, nil)
	if err != nil {
		debug.Log("Error querying conversations: %v", err)
		return nil, err
	}

	var found []Result
	for _, doc := range results {
		debug.Log("Found conversation: %s", doc.ID)
		found = append(found, Result{
			ID:      doc.ID,
			Content: doc.Content,
			Created: doc.Metadata["created"],
			Updated: doc.Metadata["updated"],
		})
	}

	return found, nil
}

// Returns a string representation of a search result.
func (r *Result) ConversationString() string {
	content := r.Content
	created := r.Created
	updated := r.Updated

	if updated == "" {
		return fmt.Sprintf("Conversation on %s:\n%s\n\n", created, content)
	}

	return fmt.Sprintf("Conversation from %s to %s:\n%s\n\n", created, updated, content)
}
