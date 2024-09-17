package gpt

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/sysread/fnord/pkg/debug"
	"github.com/sysread/fnord/pkg/storage"
	"github.com/sysread/fnord/pkg/util"
)

func getToolStatusLine(toolName string) string {
	switch toolName {
	case "query_conversations":
		return "Checking past conversations..."
	case "query_project_files":
		return "Searching project files..."
	case "curl":
		return "Downloading content from the web..."
	case "save_fact":
		return "Saving a new fact..."
	case "update_fact":
		return "Updating a saved fact..."
	case "delete_fact":
		return "Deleting a saved fact..."
	case "search_facts":
		return "Searching saved facts..."
	default:
		return "Executing " + toolName + "..."
	}
}

func queryConversations(argsJSON string) (string, error) {
	debug.Log("[gpt] [query_conversations] %s", argsJSON)

	var query struct {
		QueryText string `json:"query_text"`
	}

	if err := json.Unmarshal([]byte(argsJSON), &query); err != nil {
		debug.Log("[gpt] [query_conversations] error unmarshalling args: %s", err)
		return "", fmt.Errorf("query_vector_db: error unmarshalling args: %s", err)
	}

	results, err := storage.SearchConversations(query.QueryText, 10)
	if err != nil {
		debug.Log("[gpt] [query_conversations] error searching storage: %s", err)
		return "", fmt.Errorf("query_vector_db: error searching storage: %s", err)
	}

	var output strings.Builder
	for _, result := range results {
		output.WriteString(result.ConversationString())
	}

	debug.Log("[gpt] [query_conversations] returning %d results", len(results))
	return output.String(), nil
}

func queryProjectFiles(argsJSON string) (string, error) {
	debug.Log("[gpt] [query_project_files] %s", argsJSON)

	var query struct {
		QueryText string `json:"query_text"`
	}

	if err := json.Unmarshal([]byte(argsJSON), &query); err != nil {
		debug.Log("[gpt] [query_project_files] error unmarshalling args: %s", err)
		return "", fmt.Errorf("query_project_files: error unmarshalling args: %s", err)
	}

	results, err := storage.SearchProject(query.QueryText, 10)
	if err != nil {
		debug.Log("[gpt] [query_project_files] error searching project: %s", err)
		return "", fmt.Errorf("query_project_files: error searching project: %s", err)
	}

	var output strings.Builder
	for _, result := range results {
		output.WriteString(result.ProjectFileString())
	}

	debug.Log("[gpt] [query_project_files] returning %d results", len(results))
	return output.String(), nil
}

func curl(argsJSON string) (string, error) {
	debug.Log("[gpt] [curl] %s", argsJSON)

	var args struct {
		URLs []string `json:"urls"`
	}

	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		debug.Log("[gpt] [curl] error unmarshalling args: %s", err)
		return "", fmt.Errorf("curl: error unmarshalling args: %s", err)
	}

	// Retrieve the contents of each URL. We'll spin each off into a
	// goroutine and wait for all of them to finish before continuing.
	var outputs = make(map[string]string)
	var condvar sync.WaitGroup

	for _, url := range args.URLs {
		condvar.Add(1)
		outputs[url] = "<not yet downloaded>"

		go func(url string) {
			defer condvar.Done()

			output, err := util.HttpGetText(url)
			if err != nil {
				debug.Log("[gpt] [curl] error making HTTP request: %s", err)
				outputs[url] = fmt.Sprintf("Error making HTTP request: %s", err)
			}

			outputs[url] = output
		}(url)
	}

	condvar.Wait()

	// Construct the output string
	buf := strings.Builder{}
	for url, output := range outputs {
		buf.WriteString(fmt.Sprintf("Contents of %s:\n\n%s\n", url, output))
		buf.WriteString("-----\n\n")
	}

	debug.Log("[gpt] [curl] returning %d bytes", buf.Len())
	return buf.String(), nil
}

func saveFact(argsJSON string) (string, error) {
	debug.Log("[gpt] [save_fact] %s", argsJSON)

	var info struct {
		Content string `json:"content"`
	}

	if err := json.Unmarshal([]byte(argsJSON), &info); err != nil {
		debug.Log("[gpt] [save_fact] error unmarshalling args: %s", err)
		return "", fmt.Errorf("save_fact: error unmarshalling args: %s", err)
	}

	id, err := storage.CreateFact(info.Content)
	if err != nil {
		debug.Log("[gpt] [save_fact] error saving fact: %s", err)
		return "", fmt.Errorf("save_fact: error saving fact: %s", err)
	}

	debug.Log("[gpt] [save_fact] saved fact with ID %s", id)
	return fmt.Sprintf("Saved fact with ID %s", id), nil
}

func updateFact(argsJSON string) (string, error) {
	debug.Log("[gpt] [update_fact] %s", argsJSON)

	var info struct {
		ID      string `json:"id"`
		Content string `json:"content"`
	}

	if err := json.Unmarshal([]byte(argsJSON), &info); err != nil {
		debug.Log("[gpt] [update_fact] error unmarshalling args: %s", err)
		return "", fmt.Errorf("update_fact: error unmarshalling args: %s", err)
	}

	if _, err := storage.UpdateFact(info.ID, info.Content); err != nil {
		debug.Log("[gpt] [update_fact] error updating fact: %s", err)
		return "", fmt.Errorf("update_fact: error updating fact: %s", err)
	}

	debug.Log("[gpt] [update_fact] updated fact with ID %s", info.ID)
	return fmt.Sprintf("Updated fact with ID %s", info.ID), nil
}

func deleteFact(argsJSON string) (string, error) {
	debug.Log("[gpt] [delete_fact] %s", argsJSON)

	var info struct {
		ID string `json:"id"`
	}

	if err := json.Unmarshal([]byte(argsJSON), &info); err != nil {
		debug.Log("[gpt] [delete_fact] error unmarshalling args: %s", err)
		return "", fmt.Errorf("delete_fact: error unmarshalling args: %s", err)
	}

	if err := storage.DeleteFact(info.ID); err != nil {
		debug.Log("[gpt] [delete_fact] error deleting fact: %s", err)
		return "", fmt.Errorf("delete_fact: error deleting fact: %s", err)
	}

	debug.Log("[gpt] [delete_fact] deleted fact with ID %s", info.ID)
	return fmt.Sprintf("Deleted fact with ID %s", info.ID), nil
}

func searchFacts(argsJSON string) (string, error) {
	debug.Log("[gpt] [search_facts] %s", argsJSON)

	var query struct {
		QueryText string `json:"query_text"`
	}

	if err := json.Unmarshal([]byte(argsJSON), &query); err != nil {
		debug.Log("[gpt] [search_facts] error unmarshalling args: %s", err)
		return "", fmt.Errorf("query_facts: error unmarshalling args: %s", err)
	}

	results, err := storage.SearchFacts(query.QueryText, 10)
	if err != nil {
		debug.Log("[gpt] [search_facts] error searching saved facts: %s", err)
		return "", fmt.Errorf("query_facts: error searching saved facts: %s", err)
	}

	var output strings.Builder
	for _, result := range results {
		output.WriteString(result.FactString())
	}

	debug.Log("[gpt] [search_facts] returning %d results", len(results))
	return output.String(), nil
}
