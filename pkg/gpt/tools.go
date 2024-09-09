package gpt

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/sysread/fnord/pkg/storage"
	"github.com/sysread/fnord/pkg/util"
)

func queryVectorDB(argsJSON string) (string, error) {
	var query struct {
		QueryText string `json:"query_text"`
	}

	if err := json.Unmarshal([]byte(argsJSON), &query); err != nil {
		return "", fmt.Errorf("query_vector_db: error unmarshalling args: %s", err)
	}

	results, err := storage.Search(query.QueryText, 10)
	if err != nil {
		return "", fmt.Errorf("query_vector_db: error searching storage: %s", err)
	}

	var output strings.Builder
	for _, result := range results {
		output.WriteString(result.String())
	}

	return output.String(), nil
}

func queryProjectFiles(argsJSON string) (string, error) {
	var query struct {
		QueryText string `json:"query_text"`
	}

	if err := json.Unmarshal([]byte(argsJSON), &query); err != nil {
		return "", fmt.Errorf("query_project_files: error unmarshalling args: %s", err)
	}

	results, err := storage.SearchProject(query.QueryText, 10)
	if err != nil {
		return "", fmt.Errorf("query_project_files: error searching project: %s", err)
	}

	var output strings.Builder
	for _, result := range results {
		output.WriteString(result.ProjectFileString())
	}

	return output.String(), nil
}

func curl(argsJSON string) (string, error) {
	var args struct {
		URLs []string `json:"urls"`
	}

	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
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

	return buf.String(), nil
}

func saveFact(argsJSON string) (string, error) {
	var info struct {
		Content string `json:"content"`
	}

	if err := json.Unmarshal([]byte(argsJSON), &info); err != nil {
		return "", fmt.Errorf("save_fact: error unmarshalling args: %s", err)
	}

	id, err := storage.CreateFact(info.Content)
	if err != nil {
		return "", fmt.Errorf("save_fact: error saving fact: %s", err)
	}

	return fmt.Sprintf("Saved fact with ID %s", id), nil
}

func updateFact(argsJSON string) (string, error) {
	var info struct {
		ID      string `json:"id"`
		Content string `json:"content"`
	}

	if err := json.Unmarshal([]byte(argsJSON), &info); err != nil {
		return "", fmt.Errorf("update_fact: error unmarshalling args: %s", err)
	}

	if err := storage.UpdateFact(info.ID, info.Content); err != nil {
		return "", fmt.Errorf("update_fact: error updating fact: %s", err)
	}

	return fmt.Sprintf("Updated fact with ID %s", info.ID), nil
}

func deleteFact(argsJSON string) (string, error) {
	var info struct {
		ID string `json:"id"`
	}

	if err := json.Unmarshal([]byte(argsJSON), &info); err != nil {
		return "", fmt.Errorf("delete_fact: error unmarshalling args: %s", err)
	}

	if err := storage.DeleteFact(info.ID); err != nil {
		return "", fmt.Errorf("delete_fact: error deleting fact: %s", err)
	}

	return fmt.Sprintf("Deleted fact with ID %s", info.ID), nil
}

func searchFacts(argsJSON string) (string, error) {
	var query struct {
		QueryText string `json:"query_text"`
	}

	if err := json.Unmarshal([]byte(argsJSON), &query); err != nil {
		return "", fmt.Errorf("query_facts: error unmarshalling args: %s", err)
	}

	results, err := storage.SearchFacts(query.QueryText, 10)
	if err != nil {
		return "", fmt.Errorf("query_facts: error searching saved facts: %s", err)
	}

	var output strings.Builder
	for _, result := range results {
		output.WriteString(result.FactString())
	}

	return output.String(), nil
}
