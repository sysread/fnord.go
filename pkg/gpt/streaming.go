package gpt

import (
	"bufio"
	"encoding/json"
	"fmt"
	"strings"
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
					responseChan <- fmt.Sprintf("STATUS: %s", getToolStatusLine(toolCall.Function.Name))
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

				// Clear tool call outputs so that they do not get re-submitted
				// if the assistant requests further tool outputs.
				s.toolCallOutputs = nil

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
	var toolOutputString string
	var err error

	switch tool {
	case "query_conversations":
		toolOutputString, err = queryConversations(argsJSON)

	case "query_project_files":
		toolOutputString, err = queryProjectFiles(argsJSON)

	case "curl":
		toolOutputString, err = curl(argsJSON)

	case "save_fact":
		toolOutputString, err = saveFact(argsJSON)

	case "delete_fact":
		toolOutputString, err = deleteFact(argsJSON)

	case "update_fact":
		toolOutputString, err = updateFact(argsJSON)

	case "search_facts":
		toolOutputString, err = searchFacts(argsJSON)

	default:
		s.fail("unhandled function call: %s", tool)
	}

	if err != nil {
		s.fail("%s: %s", tool, err)
	}

	s.toolCallOutputs = append(s.toolCallOutputs, toolOutput{
		ToolCallID: toolCallID,
		Output:     toolOutputString,
	})
}
