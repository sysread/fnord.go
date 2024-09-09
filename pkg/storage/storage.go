// Storage provides an interface for storing and retrieving conversation data
package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/philippgille/chromem-go"
	gitignore "github.com/sabhiram/go-gitignore"

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

// Path is the path to the storage directory for the selected box
var Path string

// Conversations is the chromem collection of conversation data
var Conversations *chromem.Collection

// Project path is the path to a project directory, selected by the user via
// the --project flag, to be indexed by the service. This is optional. If
// unset, the service will not index a project directory.
var ProjectPath string

// ProjectFiles is the chromem collection of files in the project directory
var ProjectFiles *chromem.Collection

// ProjectGitIgnored is the gitignore parser for the project directory
var ProjectGitIgnored *gitignore.GitIgnore

// Init initializes the storage system
func Init(config *config.Config) error {
	if DB != nil {
		return nil
	}

	Path = filepath.Join(config.Home, "vector_store")

	var err error

	DB, err = chromem.NewPersistentDB(Path, true)
	if err != nil {
		return err
	}

	// Initialize the conversations collection
	conversationsCollectionName := "conversations:" + config.Box
	Conversations, err = DB.GetOrCreateCollection(conversationsCollectionName, nil, nil)
	if err != nil {
		return err
	}

	// Initialize the project files collection
	if config.ProjectPath != "" {
		gitPath := filepath.Join(config.ProjectPath, ".git")
		if _, err := os.Stat(gitPath); err != nil {
			return fmt.Errorf("project path %s is not a git repository", config.ProjectPath)
		}

		ProjectPath = config.ProjectPath

		collectionName := fmt.Sprintf("project_files:%s", ProjectPath)
		ProjectFiles, err = DB.GetOrCreateCollection(collectionName, nil, nil)
		if err != nil {
			debug.Log("Error creating %s collection: %v", collectionName, err)
		} else {
			// Load the .gitignore file
			ProjectGitIgnored, err = gitignore.CompileIgnoreFile(filepath.Join(ProjectPath, ".gitignore"))
			if err != nil {
				panic(fmt.Errorf("error loading .gitignore: %v", err))
			}

			go startIndexer()
		}
	}

	// Initialize the facts collection
	InitializeFactsCollection(config)

	return nil
}

// Function to list all boxes' collections
func GetBoxes() ([]string, error) {
	collections := DB.ListCollections()
	var boxes []string

	for name := range collections {
		// We exclude project files' collections based on their naming pattern
		if strings.HasPrefix(name, "conversations:") {
			name = strings.TrimPrefix(name, "conversations:")
			boxes = append(boxes, name)
		}
	}

	return boxes, nil
}

// Function to list all projects' collections
func GetProjects() ([]string, error) {
	collections := DB.ListCollections()
	var projects []string

	for name := range collections {
		// We exclude project files' collections based on their naming pattern
		if strings.HasPrefix(name, "project_files:") {
			name = strings.TrimPrefix(name, "project_files:")
			projects = append(projects, name)
		}
	}

	return projects, nil
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
