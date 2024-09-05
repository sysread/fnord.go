package gpt

import (
	"bufio"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sysread/fnord/pkg/storage"
	"github.com/sysread/fnord/pkg/util"
)

type streamer struct {
	done            bool
	msgOutputChan   chan<- string
	toolCallOutputs []toolOutput
}

type threadMessageDelta struct {
	Delta struct {
		Content []struct {
			Text struct {
				Value string `json:"value"`
			} `json:"text"`
		} `json:"content"`
	} `json:"delta"`
}

type threadRequiredAction struct {
	RunID          string `json:"id"`
	RequiredAction struct {
		SubmitToolOutputs struct {
			ToolCalls []struct {
				ID       string `json:"id"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"submit_tool_outputs"`
	} `json:"required_action"`
}

type toolOutput struct {
	ToolCallID string `json:"tool_call_id"`
	Output     string `json:"output"`
}

func (c *OpenAIClient) RunThread(threadID string, responseChan chan<- string) {
	s := &streamer{
		done:          false,
		msgOutputChan: responseChan,
	}

	run, err := c.CreateRun(threadID)
	if err != nil {
		s.fail("Error creating run: %s", err)
		return
	}

	var event string

LINE:
	for scanner := bufio.NewScanner(run); scanner.Scan(); {
		line := scanner.Text()

		// New event
		if strings.HasPrefix(line, "event: ") {
			event = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			continue LINE
		}

		// Data recevied
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))

			switch event {
			case "done":
				if data == "[DONE]" {
					s.finish()
					break LINE
				}

			case "thread.message.delta":
				var delta threadMessageDelta

				if err := json.Unmarshal([]byte(data), &delta); err != nil {
					s.fail("Error unmarshalling thread message delta: %s", err)
					return
				}

				for _, message := range delta.Delta.Content {
					s.send(message.Text.Value)
				}

			case "thread.run.requires_action":
				var action threadRequiredAction

				if err := json.Unmarshal([]byte(data), &action); err != nil {
					s.fail("Error unmarshalling thread required action: %s", err)
					return
				}

				// Collect tool call outputs
				for _, toolCall := range action.RequiredAction.SubmitToolOutputs.ToolCalls {
					s.addToolCallOutput(
						toolCall.ID,
						toolCall.Function.Name,
						toolCall.Function.Arguments,
					)
				}

				// Submit tool call outputs
				newRun, err := c.submitToolOutputs(threadID, action.RunID, s.toolCallOutputs)
				if err != nil {
					s.fail("Error submitting tool outputs: %s", err)
					return
				}

				// Replace `run` with the response body reader returned by
				// `SubmitToolOutputs`, which will continue streaming the
				// response. Then, jump back to the beginning so that our
				// scanner is reinitialized with the new reader.
				run = newRun
				goto LINE
			}
		}
	}

	s.finish()
}

func (s *streamer) finish() {
	if s.done {
		return
	}

	s.done = true
	close(s.msgOutputChan)
}

func (s *streamer) send(msg string) {
	if s.done {
		return
	}

	s.msgOutputChan <- msg
}

func (s *streamer) fail(msg string, args ...interface{}) {
	if s.done {
		return
	}

	errorMsg := fmt.Sprintf(msg, args...)
	s.send(errorMsg)
	s.finish()
}

func (s *streamer) addToolCallOutput(toolCallID, tool, argsJSON string) {
	switch tool {
	case "query_vector_db":
		var query struct {
			QueryText string `json:"query_text"`
		}

		if err := json.Unmarshal([]byte(argsJSON), &query); err != nil {
			s.fail("Error unmarshalling query_vector_db args: %s", err)
		}

		results, err := storage.Search(query.QueryText, 10)
		if err != nil {
			s.fail("Error searching storage: %s", err)
		}

		var output strings.Builder
		for _, result := range results {
			output.WriteString(result.String())
		}

		s.toolCallOutputs = append(s.toolCallOutputs, toolOutput{
			ToolCallID: toolCallID,
			Output:     output.String(),
		})

	case "query_project_files":
		var query struct {
			QueryText string `json:"query_text"`
		}

		if err := json.Unmarshal([]byte(argsJSON), &query); err != nil {
			s.fail("Error unmarshalling query_project_files args: %s", err)
		}

		results, err := storage.SearchProject(query.QueryText, 10)
		if err != nil {
			s.fail("Error searching storage: %s", err)
		}

		var output strings.Builder
		for _, result := range results {
			output.WriteString(result.ProjectFileString())
		}

		s.toolCallOutputs = append(s.toolCallOutputs, toolOutput{
			ToolCallID: toolCallID,
			Output:     output.String(),
		})

	case "curl":
		var args struct {
			URL string `json:"url"`
		}

		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			s.fail("Error unmarshalling curl args: %s", err)
		}

		output, err := util.HttpGet(args.URL)
		if err != nil {
			s.fail("Error making HTTP request: %s", err)
		}

		s.toolCallOutputs = append(s.toolCallOutputs, toolOutput{
			ToolCallID: toolCallID,
			Output:     fmt.Sprintf("Contents of %s:\n%s", args.URL, output),
		})

	default:
		s.fail("unhandled function call: %s", tool)
	}
}
