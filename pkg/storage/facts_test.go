package storage_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/sysread/fnord/pkg/config"
	"github.com/sysread/fnord/pkg/storage"
)

// Setup initializes the test configuration and database.
func setupTestConfig(t *testing.T) *config.Config {
	return &config.Config{
		Home: t.TempDir(), // Use a temporary directory for the tests
		Box:  "test_box",  // Use a distinct box for testing
	}
}

func setupTestStorage(t *testing.T, cfg *config.Config) {
	var err error
	t.Logf("CONFIG: %#v", cfg)

	err = storage.Init(cfg)
	assert.NoError(t, err)

	err = storage.InitializeFactsCollection(cfg)
	assert.NoError(t, err)

	err = storage.ResetFactCollection()
	assert.NoError(t, err)
}

func TestFactStorage(t *testing.T) {
	// Setup configuration and storage
	cfg := setupTestConfig(t)
	setupTestStorage(t, cfg)

	// Test CreateFact
	content := "This is a test fact."
	id, err := storage.CreateFact(content)
	assert.NoError(t, err)
	assert.NotEmpty(t, id)

	// Test ReadFact
	readContent, err := storage.ReadFact(id)
	assert.NoError(t, err)
	assert.Equal(t, content, readContent)

	// Test UpdateFact
	newContent := "This is an updated fact."
	err = storage.UpdateFact(id, newContent)
	assert.NoError(t, err)

	// Test Read after Update
	updatedContent, err := storage.ReadFact(id)
	assert.NoError(t, err)
	assert.Equal(t, newContent, updatedContent)

	// Test SearchFact
	searchResults, err := storage.SearchFacts("updated", 10)
	assert.NoError(t, err)
	assert.Len(t, searchResults, 1)
	assert.Equal(t, id, searchResults[0].ID)
	assert.Equal(t, newContent, searchResults[0].Content)

	// Test DeleteFact
	err = storage.DeleteFact(id)
	assert.NoError(t, err)

	// Test Read after Delete
	deletedContent, err := storage.ReadFact(id)
	assert.Error(t, err)
	assert.Empty(t, deletedContent)
}

func TestSearchFact(t *testing.T) {
	// Setup configuration and storage
	cfg := setupTestConfig(t)
	setupTestStorage(t, cfg)

	var err error

	contents := []string{
		"Fact one",
		"Another fact",
		"A third interesting fact",
		"Fact four",
	}

	ids := make([]string, len(contents))

	for i, content := range contents {
		ids[i], err = storage.CreateFact(content)
		assert.NoError(t, err)
	}

	searchResults, err := storage.SearchFacts("fact", 10)
	assert.NoError(t, err)
	assert.Len(t, searchResults, len(contents))

	searchResults, err = storage.SearchFacts("interesting", 1)
	assert.NoError(t, err)
	assert.Len(t, searchResults, 1)
	assert.Equal(t, "A third interesting fact", searchResults[0].Content)
}
