package data

import (
	"math"
	"os"
	"sort"

	"github.com/segmentio/encoding/json"
)

type searchResult struct {
	Score        float32
	Conversation ConversationIndexEntry
}

func (ds *DataStore) Search(queryEmbedding []float32, numResults int) ([]ConversationIndexEntry, error) {
	// Walk the conversation directory, loading each conversation
	conversations, err := ds.ListConversations()
	if err != nil {
		return nil, err
	}

	matches := make([]searchResult, 0, len(conversations))
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

		matches = append(matches, searchResult{
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

// Calculates the cosine similarity between two vectors.
func cosineSimilarity(a []float32, b []float32) float32 {
    if len(a) == 0 || len(b) == 0 {
        return 0
    }

    var dotProduct float32
    var magnitudeA float32
    var magnitudeB float32

    for i := range a {
        dotProduct += a[i] * b[i]
        magnitudeA += a[i] * a[i]
        magnitudeB += b[i] * b[i]
    }

    magnitudeA = float32(math.Sqrt(float64(magnitudeA)))
    magnitudeB = float32(math.Sqrt(float64(magnitudeB)))

    if magnitudeA == 0 || magnitudeB == 0 {
        return 0
    }

    return dotProduct / (magnitudeA * magnitudeB)
}
