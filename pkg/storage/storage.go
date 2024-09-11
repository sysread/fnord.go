// Storage provides an interface for storing and retrieving conversation data
package storage

import (
	"fmt"
	"path/filepath"
	"strings"

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

// DB is the database connection
var DB *chromem.DB

// Path is the path to the storage directory for the selected box
var Path string

// Init initializes the storage system
func Init(config *config.Config) error {
	var err error

	if DB != nil {
		return nil
	}

	Path = filepath.Join(config.Home, "vector_store")

	DB, err = chromem.NewPersistentDB(Path, true)
	if err != nil {
		return err
	}

	// Initialize the conversations collection
	err = InitializeConversationsCollection(config)
	if err != nil {
		return err
	}

	// Initialize the facts collection
	err = InitializeFactsCollection(config)
	if err != nil {
		return err
	}

	// Initialize the project files collection
	if config.ProjectPath != "" {
		err = InitializeProjectFilesCollection(config)
		if err != nil {
			return err
		}
	}

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
