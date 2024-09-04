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

// Storage provides an interface for storing and retrieving conversation data
type Storage struct {
	db            *chromem.DB
	conversations *chromem.Collection
	path          string
}

// Result represents a search result
type Result struct {
	ID      string
	Content string
	Created string
	Updated string
}

// NewStorage initializes the storage system
func NewStorage(config *config.Config) (*Storage, error) {
	vectordb_path := filepath.Join(config.Home, "vector_store")

	db, err := chromem.NewPersistentDB(vectordb_path, true)
	if err != nil {
		return nil, err
	}

	conversations, err := db.GetOrCreateCollection("conversations", nil, nil)
	if err != nil {
		return nil, err
	}

	store := &Storage{
		db:            db,
		conversations: conversations,
		path:          vectordb_path,
	}

	return store, nil
}

// Create stores a new entry and returns its UUID
func (s *Storage) Create(content string) (string, error) {
	id := uuid.New().String()

	collection, err := s.db.GetOrCreateCollection("conversations", nil, nil)
	if err != nil {
		return "", err
	}

	now := time.Now().Format(time.DateOnly)
	document := chromem.Document{
		ID:       id,
		Content:  content,
		Metadata: map[string]string{
			"created": now,
			"updated": now,
		},
	}

	err = collection.AddDocument(context.Background(), document)
	if err != nil {
		return "", err
	}

	return id, nil
}

// Read retrieves content by UUID
func (s *Storage) Read(id string) (string, error) {
	document, err := s.conversations.GetByID(context.Background(), id)
	if err != nil {
		return "", err
	}

	return document.Content, nil
}

// Update modifies the content of an existing entry
func (s *Storage) Update(id, content string) error {
	// Find the existing entry
	existingEntry, err := s.conversations.GetByID(context.Background(), id)
	if err != nil {
		return err
	}

	// Delete it from the store
	err = s.conversations.Delete(context.Background(), nil, nil, id)
	if err != nil {
		return err
	}

	// Then add the updated entry
	return s.conversations.AddDocument(context.Background(), chromem.Document{
		ID:       id,
		Content:  content,
		Metadata: map[string]string{
			"created": existingEntry.Metadata["created"],
			"updated": time.Now().Format(time.DateOnly),
		},
	})
}

// Delete removes an entry by UUID
func (s *Storage) Delete(id string) error {
	return s.conversations.Delete(context.Background(), nil, nil, id)
}

// Search returns a list of UUIDs that match the query
func (s *Storage) Search(query string, numResults int) ([]Result, error) {
	maxResults := s.conversations.Count()
	if numResults > maxResults {
		numResults = maxResults
	}

	results, err := s.conversations.Query(context.Background(), query, numResults, nil, nil)
	if err != nil {
		return nil, err
	}

	var conversations []Result
	for _, doc := range results {
		conversations = append(conversations, Result{
			ID:      doc.ID,
			Content: doc.Content,
			Created: doc.Metadata["created"],
			Updated: doc.Metadata["updated"],
		})
	}

	return conversations, nil
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
