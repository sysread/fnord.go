// Storage provides an interface for storing and retrieving conversation data
package storage

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/philippgille/chromem-go"

	"github.com/sysread/fnord/pkg/config"
)

// Result represents a search result
type Result struct {
	ID      string
	Content string
	Created string
	Updated string
}

// Path is the path to the storage directory for the selected box
var Path string

// DB is the database connection
var DB *chromem.DB

// Conversations is the collection of conversation data
var Conversations *chromem.Collection

// Init initializes the storage system
func Init(config *config.Config) error {
	if DB != nil {
		return nil
	}

	Path = filepath.Join(config.BoxPath, "vector_store")

	var err error

	DB, err = chromem.NewPersistentDB(Path, true)
	if err != nil {
		return err
	}

	Conversations, err = DB.GetOrCreateCollection("conversations", nil, nil)
	if err != nil {
		return err
	}

	return nil
}

// Create stores a new entry and returns its UUID
func Create(content string) (string, error) {
	id := uuid.New().String()

	now := time.Now().Format(time.RFC3339)
	document := chromem.Document{
		ID:      id,
		Content: content,
		Metadata: map[string]string{
			"created": now,
			"updated": now,
		},
	}

	err := Conversations.AddDocument(context.Background(), document)
	if err != nil {
		return "", err
	}

	return id, nil
}

// Read retrieves content by UUID
func Read(id string) (string, error) {
	document, err := Conversations.GetByID(context.Background(), id)
	if err != nil {
		return "", err
	}

	return document.Content, nil
}

// Update modifies the content of an existing entry
func Update(id, content string) error {
	// Find the existing entry
	existingEntry, err := Conversations.GetByID(context.Background(), id)
	if err != nil {
		return err
	}

	// Preserve the original creation date and generate a new updated date
	now := time.Now().Format(time.RFC3339)
	created := existingEntry.Metadata["created"]
	if created == "" {
		created = now
	}

	// Then add the updated entry
	return Conversations.AddDocument(context.Background(), chromem.Document{
		ID:      id,
		Content: content,
		Metadata: map[string]string{
			"created": created,
			"updated": now,
		},
	})
}

// Delete removes an entry by UUID
func Delete(id string) error {
	return Conversations.Delete(context.Background(), nil, nil, id)
}

// Search returns a list of UUIDs that match the query
func Search(query string, numResults int) ([]Result, error) {
	maxResults := Conversations.Count()
	if numResults > maxResults {
		numResults = maxResults
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

// Returns a string representation of a search result.
func (r *Result) String() string {
	content := r.Content
	created := r.Created
	updated := r.Updated

	if updated == "" {
		return fmt.Sprintf("Conversation on %s:\n%s\n\n", created, content)
	}

	return fmt.Sprintf("Conversation from %s to %s:\n%s\n\n", created, updated, content)
}
