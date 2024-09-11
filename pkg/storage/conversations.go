package storage

import (
	"context"
	"time"

	"github.com/philippgille/chromem-go"

	"github.com/sysread/fnord/pkg/config"
)

// Conversations is the chromem collection of conversation data
var Conversations *chromem.Collection

// InitializeConversationsCollection initializes the conversations collection in the chromem database.
func InitializeConversationsCollection(config *config.Config) error {
	var err error
	collectionName := "conversations:" + config.Box
	Conversations, err = DB.GetOrCreateCollection(collectionName, nil, nil)
	return err
}

// CreateConversation stores a new conversation and returns an error if the operation fails.
func CreateConversation(threadID string, content string) error {
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
	document, err := Conversations.GetByID(context.Background(), threadID)
	if err != nil {
		return "", err
	}

	return document.Content, nil
}

// UpdateConversation modifies the content of an existing conversation.
func UpdateConversation(threadID, content string) error {
	// Find the existing entry
	existingEntry, err := Conversations.GetByID(context.Background(), threadID)
	if err != nil {
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
	return Conversations.AddDocuments(context.Background(), []chromem.Document{existingEntry}, 1)
}

// DeleteConversation removes a conversation by thread ID.
func DeleteConversation(threadID string) error {
	return Conversations.Delete(context.Background(), nil, nil, threadID)
}

// SearchConversations queries the conversation collection for a given query
// string and returns a slice of search results.
func SearchConversations(query string, numResults int) ([]Result, error) {
	maxResults := Conversations.Count()
	if numResults > maxResults {
		numResults = maxResults
	}

	if numResults == 0 {
		return []Result{}, nil
	}

	results, err := Conversations.Query(context.Background(), query, numResults, nil, nil)
	if err != nil {
		return nil, err
	}

	var found []Result
	for _, doc := range results {
		found = append(found, Result{
			ID:      doc.ID,
			Content: doc.Content,
			Created: doc.Metadata["created"],
			Updated: doc.Metadata["updated"],
		})
	}

	return found, nil
}
