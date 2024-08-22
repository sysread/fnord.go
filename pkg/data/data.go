/*
# SYNOPSIS

This package provides access to the persistent file store used to save and load conversation data.

The store path is determined by the FNORD_HOME environment variable, defaulting to $HOME/.config/fnord.

# ORGANIZATION

The store is organized as follows:

	$FNORD_HOME/
		conversations/
			index.jsonl
			$uuid1/
				embeddings.json
				messages.jsonl
			$uuid2/
				embeddings.json
				messages.jsonl
			...

Individual conversations are comprised of two files:

  - `messages.json`: A JSON array of ChatMessage objects.
  - `embeddings.json`: A JSON array of embeddings for each message in the conversation.

## index.jsonl

Each line in the index file is a JSON object representing a conversation:

```json

	{
		"uuid":		"uuid1",
		"created":	"2021-01-01T00:00:00Z",
		"modified":	"2021-01-01T00:00:00Z",
		"summary":	"A summary of the conversation",
	}

```

## messages.jsonl

Each line in the messagesfile is a JSON object representing a message:

```json

	{
		// You | Assistant
		"from": "You",

		// Whether the message should be visible in the chat UI
		"is_hidden": false,

		// The message content
		"content": "Hello, how are you?",
	}

```

## embeddings.json

The `embeddings.json` file stores a single JSON object representing the embedding for the summary of the entire conversation.

The file is organized as follows:

```json

	{
		// The embedding vector as an array of floats
		"embedding": [0.123, -0.456, ...],

		// SHA-256 hash of the conversation content
		"hash": "DEADBEEF"

		// The time when the embedding was generated
		"timestamp": "2024-08-16T00:00:00Z"
	}

```
*/
package data

import (
	"bufio"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/encoding/json"

	"github.com/sysread/fnord/pkg/debug"
	"github.com/sysread/fnord/pkg/gpt"
	"github.com/sysread/fnord/pkg/messages"
)

type DataStore struct {
	gptClient          gpt.Client
	HomeDir            string
	ConversationsDir   string
	ConversationsIndex string
}

type Conversation struct {
	DataStore *DataStore
	Created   time.Time
	Modified  time.Time
	UUID      string
	Hash      [32]byte
	Summary   string
	Embedding []float32
	Messages  []messages.ChatMessage
}

type ConversationIndexEntry struct {
	Created  time.Time `json:"start"`
	Modified time.Time `json:"end"`
	UUID     string    `json:"uuid"`
	Summary  string    `json:"summary"`
}

type EmbeddingEntry struct {
	UUID      string    `json:"uuid"`
	Hash      [32]byte  `json:"hash"`
	Embedding []float32 `json:"embedding"`
}

type SearchResult struct {
	Score        float32
	Conversation ConversationIndexEntry
}

func NewDataStore() *DataStore {
	// Retrieve $FNORD_HOME from the environment, defaulting to
	// $HOME/.config/fnord.
	baseDir := os.Getenv("FNORD_HOME")
	if baseDir == "" {
		baseDir = filepath.Join(os.Getenv("HOME"), ".config", "fnord")
	}

	// Ensure the base directory exists.
	if err := os.MkdirAll(baseDir, 0700); err != nil {
		panic(err)
	}

	conversationsDir := filepath.Join(baseDir, "conversations")

	// Ensure $FNORD_HOME/conversations exists.
	if err := os.MkdirAll(conversationsDir, 0700); err != nil {
		panic(err)
	}

	return &DataStore{
		gptClient:          gpt.NewOpenAIClient(),
		HomeDir:            baseDir,
		ConversationsDir:   conversationsDir,
		ConversationsIndex: filepath.Join(conversationsDir, "index.json"),
	}
}

func (ds *DataStore) conversationDirPath(uuid string) string {
	return filepath.Join(ds.ConversationsDir, uuid)
}

func (ds *DataStore) conversationFilePath(uuid string) string {
	return filepath.Join(ds.conversationDirPath(uuid), "messages.jsonl")
}

func (ds *DataStore) embeddingFilePath(uuid string) string {
	return filepath.Join(ds.conversationDirPath(uuid), "embeddings.json")
}

func (ds *DataStore) NewConversation() *Conversation {
	return &Conversation{
		DataStore: ds,
		Created:   time.Now(),
		Modified:  time.Now(),
		UUID:      uuid.NewString(),
		Messages:  make([]messages.ChatMessage, 0, 50),
	}
}

func (ds *DataStore) LoadConversation(info ConversationIndexEntry) (*Conversation, error) {
	filePath := ds.conversationFilePath(info.UUID)
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	msgs := make([]messages.ChatMessage, 0, 50)
	for scanner.Scan() {
		var data messages.ChatMessage

		if err := json.Unmarshal(scanner.Bytes(), &data); err != nil {
			return nil, err
		}

		msgs = append(msgs, data)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	conversation := &Conversation{
		DataStore: ds,
		Created:   info.Created,
		Modified:  info.Modified,
		UUID:      info.UUID,
		Messages:  msgs,
	}

	return conversation, nil
}

func (ds *DataStore) ListConversations() ([]ConversationIndexEntry, error) {
	file, err := os.Open(ds.ConversationsIndex)

	if err != nil {
		return nil, err
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)
	conversations := make([]ConversationIndexEntry, 0, 200)
	for scanner.Scan() {
		var data ConversationIndexEntry

		if err := json.Unmarshal(scanner.Bytes(), &data); err != nil {
			return nil, err
		}

		conversations = append(conversations, data)
	}

	if err := scanner.Err(); err != nil {
		return conversations, err
	}

	return conversations, nil
}

func (ds *DataStore) Search(query string, numResults int) ([]ConversationIndexEntry, error) {
	// Generate an embedding for the query
	queryEmbedding, err := ds.gptClient.GetEmbedding(query)
	if err != nil {
		return nil, err
	}

	// Walk the conversation directory, loading each conversation
	conversations, err := ds.ListConversations()
	if err != nil {
		return nil, err
	}

	matches := make([]SearchResult, 0, len(conversations))
	for _, conversation := range conversations {
		// Read in the conversation embedding file
		file, err := os.Open(ds.embeddingFilePath(conversation.UUID))
		if err != nil {
			return nil, err
		}
		defer file.Close()

		// Decode the JSON data
		var data EmbeddingEntry
		if err := json.NewDecoder(file).Decode(&data); err != nil {
			return nil, err
		}

		// Calculate the cosine similarity between the query embedding and the
		// conversation embedding.
		similarity := cosineSimilarity(queryEmbedding, data.Embedding)

		matches = append(matches, SearchResult{
			Score:        similarity,
			Conversation: conversation,
		})
	}

	// Sort the matches by similarity
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Score > matches[j].Score
	})

	// Return the top `numResults` matches
	results := make([]ConversationIndexEntry, 0, numResults)
	for i := 0; i < numResults && i < len(matches); i++ {
		results = append(results, matches[i].Conversation)
	}

	return results, nil
}

// -----------------------------------------------------------------------------
// Conversation
// -----------------------------------------------------------------------------

// AddMessage adds a message to the conversation
func (c *Conversation) AddMessage(message messages.ChatMessage) {
	c.Messages = append(c.Messages, message)
	c.Modified = time.Now()
}

// SetSummary updates the conversation's summary and embedding. Also regenerates
// the hash of the conversation summary.
func (c *Conversation) SetSummary(summary string, embedding []float32) {
	c.Summary = summary
	c.Embedding = embedding
	c.Hash = sha256.Sum256([]byte(c.Summary))
}

// HasStaleEmbedding returns true if the conversation's embedding is out of
// sync with its summary.
func (c *Conversation) HasStaleEmbedding() bool {
	hash := sha256.Sum256([]byte(c.Summary))
	return c.Hash != hash
}

// Transcript returns a string representation of the conversation.
func (c *Conversation) Transcript() string {
	var buf strings.Builder

	for _, message := range c.Messages {
		buf.WriteString(fmt.Sprintf("%s:\n\n%s\n\n", message.From, message.Content))
	}

	return buf.String()
}

// Save updates the summary and embedding, then saves the
// conversation to disk. The `Conversation`'s `Summary` and `Modified` fields
// will be updated in place.
func (c *Conversation) Save() {
	// Update the index file ($FNORD_HOME/conversations/index.jsonl)
	if err := c.saveConversationIndexEntry(); err != nil {
		debug.Log("Failed to save conversation index entry: %v", err)
	}

	// Update the embeddings file ($FNORD_HOME/conversations/$uuid/embeddings.json)
	if err := c.saveEmbedding(); err != nil {
		debug.Log("Failed to save conversation embedding: %v", err)
	}

	// Update the conversation file ($FNORD_HOME/conversations/$uuid/messages.jsonl)
	if err := c.saveConversation(); err != nil {
		debug.Log("Failed to save conversation: %v", err)
	}
}

// saveConversationIndexEntry updates the index file with the current
// conversation data. If the conversation does not yet exist in the index, a new
// entry is created. If the conversation already exists in the index, the entry
// is replaced with the updated data.
//
// Note that before calling this function, the `updateSummary` function should
// be called to ensure that the `Summary` and `Modified` fields are up to date.
func (c *Conversation) saveConversationIndexEntry() error {
	conversationsIndex := c.DataStore.ConversationsIndex

	// Open the source file for reading
	sourceFile, sourceErr := os.Open(conversationsIndex)
	// It's ok if it does not yet exist. We'll create it below.
	if sourceErr != nil {
		defer sourceFile.Close()
	}

	// Open a temp file for writing
	tempFile, tempErr := os.CreateTemp(c.DataStore.HomeDir, "index-*.json")
	if tempErr != nil {
		return tempErr
	}
	defer tempFile.Close()

	// If the index file does not yet exist, there is no work to do. If the
	// index file exists, copy each line into the temp file. If the UUID of the
	// current line matches the UUID of the entry, write the updated entry
	// instead.
	if sourceErr == nil {
		scanner := bufio.NewScanner(sourceFile)

		// Write the new entry to the top of the file
		entry := ConversationIndexEntry{
			Created:  c.Created,
			Modified: c.Modified,
			UUID:     c.UUID,
			Summary:  c.Summary,
		}

		// Overwrite the updated entry
		entryJson, entryErr := json.Marshal(entry)
		if entryErr != nil {
			return entryErr
		}

		tempFile.Write(entryJson)

		// Then, copy the rest of the file, skipping the old entry
		uuidRegex := regexp.MustCompile(`"uuid":\s*"` + c.UUID + `"`)
		for scanner.Scan() {
			line := scanner.Bytes()
			if !uuidRegex.Match(line) {
				tempFile.Write(line)
			}
		}
	}

	// If there is already a .bak, remove it.
	if _, bakErr := os.Stat(conversationsIndex + ".bak"); bakErr == nil {
		os.Remove(conversationsIndex + ".bak")
	}

	// Back up the original file
	os.Rename(conversationsIndex, conversationsIndex+".bak")

	// Rename the temp file to the original file
	os.Rename(tempFile.Name(), conversationsIndex)

	return nil
}

func (c *Conversation) saveEmbedding() error {
	conversationDirPath := c.DataStore.conversationDirPath(c.UUID)
	originalFilePath := c.DataStore.embeddingFilePath(c.UUID)

	// Ensure that the conversation directory exists
	if err := os.MkdirAll(conversationDirPath, 0700); err != nil {
		return err
	}

	// Open a temp file for writing
	tempFile, tempErr := os.CreateTemp(conversationDirPath, "embedding-*.json")
	if tempErr != nil {
		return tempErr
	}
	defer tempFile.Close()

	// Write out the embedding to the temp file
	embeddingEntry := EmbeddingEntry{
		UUID:      c.UUID,
		Hash:      c.Hash,
		Embedding: c.Embedding,
	}

	jsonData, jsonErr := json.Marshal(embeddingEntry)
	if jsonErr != nil {
		return jsonErr
	}
	tempFile.Write(jsonData)

	// If there is already a .bak, remove it.
	if _, bakErr := os.Stat(originalFilePath + ".bak"); bakErr == nil {
		os.Remove(originalFilePath + ".bak")
	}

	// Back up the original file
	os.Rename(originalFilePath, originalFilePath+".bak")

	// Rename the temp file to the original file
	os.Rename(tempFile.Name(), originalFilePath)

	return nil
}

// saveConversation saves the conversation to disk. The `Conversation`'s
// `Messages` field will be written to the conversation file. This method does
// NOT update the `Modified` field.
func (c *Conversation) saveConversation() error {
	conversationDirPath := c.DataStore.conversationDirPath(c.UUID)
	originalFilePath := c.DataStore.conversationFilePath(c.UUID)

	// Ensure that the conversation directory exists
	if err := os.MkdirAll(conversationDirPath, 0700); err != nil {
		return err
	}

	// Open a temp file for writing
	tempFile, tempErr := os.CreateTemp(conversationDirPath, "index-*.json")
	if tempErr != nil {
		return tempErr
	}
	defer tempFile.Close()

	// For an individual conversation, we're just going to overwrite the entire
	// file with the current message data.
	for _, message := range c.Messages {
		jsonData, jsonErr := json.Marshal(message)
		if jsonErr != nil {
			return jsonErr
		}
		tempFile.Write(jsonData)
	}

	// If there is already a .bak, remove it.
	if _, bakErr := os.Stat(originalFilePath + ".bak"); bakErr == nil {
		os.Remove(originalFilePath + ".bak")
	}

	// Back up the original file
	os.Rename(originalFilePath, originalFilePath+".bak")

	// Rename the temp file to the original file
	os.Rename(tempFile.Name(), originalFilePath)

	return nil
}

// -----------------------------------------------------------------------------
// Mathy stuff
// -----------------------------------------------------------------------------

func cosineSimilarity(a []float32, b []float32) float32 {
	var dotProduct float32
	var magnitudeA float32
	var magnitudeB float32

	for i := range a {
		dotProduct += a[i] * b[i]
		magnitudeA += a[i] * a[i]
		magnitudeB += b[i] * b[i]
	}

	return dotProduct / (magnitudeA * magnitudeB)
}
