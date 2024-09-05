// Storage provides an interface for storing and retrieving conversation data
package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/philippgille/chromem-go"

	"github.com/sysread/fnord/pkg/config"
	"github.com/sysread/fnord/pkg/debug"
)

// Result represents a search result
type Result struct {
	ID      string
	Content string
	Created string
	Updated string
}

// DB is the database connection
var DB *chromem.DB

// ConversationsPath is the path to the storage directory for the selected box
var ConversationsPath string

// Conversations is the chromem collection of conversation data
var Conversations *chromem.Collection

// Project path is the path to a project directory, selected by the user via
// the --project flag, to be indexed by the service.
var ProjectPath string

// ProjectFiles is the chromem collection of files in the project directory
var ProjectFiles *chromem.Collection

// Init initializes the storage system
func Init(config *config.Config) error {
	if DB != nil {
		return nil
	}

	ConversationsPath = filepath.Join(config.BoxPath, "vector_store")

	var err error

	DB, err = chromem.NewPersistentDB(ConversationsPath, true)
	if err != nil {
		return err
	}

	Conversations, err = DB.GetOrCreateCollection("conversations", nil, nil)
	if err != nil {
		return err
	}

	if config.ProjectPath != "" {
		gitPath := filepath.Join(config.ProjectPath, ".git")
		if _, err := os.Stat(gitPath); err != nil {
			return fmt.Errorf("project path %s is not a git repository", config.ProjectPath)
		}

		ProjectPath = config.ProjectPath
		collectionName := fmt.Sprintf("project_files:%s", ProjectPath)
		ProjectFiles, err = DB.GetOrCreateCollection(collectionName, nil, nil)
		if err != nil {
			debug.Log("Error creating project_files collection: %v", err)
		} else {
			go startIndexer()
		}
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

	err := Conversations.AddDocuments(context.Background(), []chromem.Document{document}, 2)
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

	doc := chromem.Document{
		ID:      id,
		Content: content,
		Metadata: map[string]string{
			"created": created,
			"updated": now,
		},
	}

	// Then add the updated entry
	return Conversations.AddDocuments(context.Background(), []chromem.Document{doc}, 2)
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

func SearchProject(query string, numResults int) ([]Result, error) {
	if ProjectFiles == nil {
		return []Result{}, nil
	}

	maxResults := ProjectFiles.Count()
	if numResults > maxResults {
		numResults = maxResults
	}

	results, err := ProjectFiles.Query(context.Background(), query, numResults, nil, nil)
	if err != nil {
		return nil, err
	}

	var found []Result
	for _, doc := range results {
		found = append(found, Result{
			ID:      doc.ID,
			Content: doc.Content,
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

func (r *Result) ProjectFileString() string {
	path := r.ID
	content := r.Content
	return fmt.Sprintf("Project file: %s\n%s\n\n", path, content)
}
