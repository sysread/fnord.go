package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/philippgille/chromem-go"

	"github.com/sysread/fnord/pkg/config"
	"github.com/sysread/fnord/pkg/debug"
)

// Facts represents the collection of stored facts.
var Facts *chromem.Collection

// InitializeFactsCollection initializes the facts collection in the chromem database.
func InitializeFactsCollection(config *config.Config) error {
	debug.Log("Initializing facts collection facts:%s", config.Box)
	var err error
	collectionName := "facts:" + config.Box
	Facts, err = DB.GetOrCreateCollection(collectionName, nil, nil)
	return err
}

// ResetFactCollection removes all facts from the collection.
func ResetFactCollection() error {
	debug.Log("Resetting facts collection")
	return Facts.Delete(context.Background(), nil, nil, "")
}

// CreateFact stores a new fact and returns its UUID.
func CreateFact(content string) (string, error) {
	debug.Log("Creating fact: '%s'", content)

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
	if err != nil {
		debug.Log("Error creating fact: %v", err)
		return "", err
	}

	debug.Log("Created fact with ID: %s", id)
	return id, nil
}

// ReadFact retrieves a fact by UUID.
func ReadFact(id string) (string, error) {
	debug.Log("Reading fact: %s", id)

	document, err := Facts.GetByID(context.Background(), id)
	if err != nil {
		debug.Log("Fact not found: %s", id)
		return "", err
	}

	debug.Log("Read fact with ID: %s", id)
	return document.Content, nil
}

// UpdateFact modifies the content of an existing fact.
func UpdateFact(id, content string) (string, error) {
	debug.Log("Updating fact %s to '%s'", id, content)

	// Find the existing entry
	existingEntry, err := Facts.GetByID(context.Background(), id)
	if err != nil {
		debug.Log("Fact not found; creating instead: %s", id)
		return CreateFact(content)
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
	_, err = id, Facts.AddDocuments(context.Background(), []chromem.Document{doc}, 1)
	if err != nil {
		debug.Log("Failed to update fact: %s", id)
		return "", err
	}

	debug.Log("Updated fact %s", id)
	return id, nil
}

// DeleteFact removes a fact by UUID.
func DeleteFact(id string) error {
	debug.Log("Deleting fact: %s", id)

	err := Facts.Delete(context.Background(), nil, nil, id)
	if err != nil {
		debug.Log("Error deleting fact: %s", id)
		return err
	}

	debug.Log("Deleted fact: %s", id)
	return nil
}

// SearchFact returns a list of facts that match the query.
func SearchFacts(query string, numResults int) ([]Result, error) {
	debug.Log("Searching facts for %d results using query '%s'", numResults, query)

	maxResults := Facts.Count()
	if numResults > maxResults {
		numResults = maxResults
	}

	if numResults == 0 {
		debug.Log("No indexed facts to search!")
		return []Result{}, nil
	}

	results, err := Facts.Query(context.Background(), query, numResults, nil, nil)
	if err != nil {
		debug.Log("Error querying facts: %v", err)
		return nil, err
	}

	var found []Result
	for _, doc := range results {
		debug.Log("Found fact: %s", doc.ID)
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
