package storage_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sysread/fnord/pkg/config"
	"github.com/sysread/fnord/pkg/storage"
)

func TestStorage(t *testing.T) {
	// Setup configuration and storage
	cfg := &config.Config{
		Home: t.TempDir(), // Use a temporary directory for the tests
	}

	store, err := storage.NewStorage(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, store)

	// Test Create
	content := "This is a test conversation."
	id, err := store.Create(content)
	assert.NoError(t, err)
	assert.NotEmpty(t, id)

	// Test Read
	readContent, err := store.Read(id)
	assert.NoError(t, err)
	assert.Equal(t, content, readContent)

	// Test Update
	newContent := "This is an updated conversation."
	err = store.Update(id, newContent)
	assert.NoError(t, err)

	// Test Read after Update
	updatedContent, err := store.Read(id)
	assert.NoError(t, err)
	assert.Equal(t, newContent, updatedContent)

	// Test Search
	searchResults, err := store.Search("updated", 10)
	assert.NoError(t, err)
	assert.Len(t, searchResults, 1)
	assert.Equal(t, id, searchResults[0].ID)
	assert.Equal(t, newContent, searchResults[0].Content)

	// Test Delete
	err = store.Delete(id)
	assert.NoError(t, err)

	// Test Read after Delete
	deletedContent, err := store.Read(id)
	assert.Error(t, err)
	assert.Empty(t, deletedContent)
}
