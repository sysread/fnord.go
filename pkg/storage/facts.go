package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/philippgille/chromem-go"

	"github.com/sysread/fnord/pkg/config"
)

// Facts represents the collection of stored facts.
var Facts *chromem.Collection

// InitializeFactsCollection initializes the facts collection in the chromem database.
func InitializeFactsCollection(config *config.Config) error {
	var err error
	collectionName := "facts:" + config.Box
	Facts, err = DB.GetOrCreateCollection(collectionName, nil, nil)
	return err
}

// ResetFactCollection removes all facts from the collection.
func ResetFactCollection() error {
	return Facts.Delete(context.Background(), nil, nil, "")
}

// CreateFact stores a new fact and returns its UUID.
func CreateFact(content string) (string, error) {
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

	err := Facts.AddDocuments(context.Background(), []chromem.Document{document}, 1)
	return id, err
}

// ReadFact retrieves a fact by UUID.
func ReadFact(id string) (string, error) {
	document, err := Facts.GetByID(context.Background(), id)
	if err != nil {
		return "", err
	}
	return document.Content, nil
}

// UpdateFact modifies the content of an existing fact.
func UpdateFact(id, content string) error {
	// Find the existing entry
	existingEntry, err := Facts.GetByID(context.Background(), id)
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

	// Add the updated entry
	return Facts.AddDocuments(context.Background(), []chromem.Document{doc}, 1)
}

// DeleteFact removes a fact by UUID.
func DeleteFact(id string) error {
	return Facts.Delete(context.Background(), nil, nil, id)
}

// SearchFact returns a list of facts that match the query.
func SearchFacts(query string, numResults int) ([]Result, error) {
	maxResults := Facts.Count()
	if numResults > maxResults {
		numResults = maxResults
	}

	if numResults == 0 {
		return []Result{}, nil
	}

	results, err := Facts.Query(context.Background(), query, numResults, nil, nil)
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

func (r *Result) FactString() string {
	id := r.ID
	content := r.Content
	created := r.Created
	updated := r.Updated
	return fmt.Sprintf("Fact with ID `%s` created on %s, last updated on %s:\n%s\n\n", id, created, updated, content)
}
