package messages

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/rivo/tview"
)

// trimMessage trims leading and trailing whitespace from a message's content.
func trimMessage(content string) string {
	content = strings.TrimLeft(content, " \r\n\t\f")
	content = strings.TrimRight(content, " \r\n\t\f")
	return content
}

func ParseMessage(from Sender, content string) ([]Message, error) {
	messages := []Message{}
	scanner := bufio.NewScanner(strings.NewReader(content))

	currentMessage := ""
	for scanner.Scan() {
		line := scanner.Text()

		isAction, action, remaining := getAction(line)
		if isAction {
			// Any built up text is a message. Add it and reset the current
			// message buffer.
			currentMessage = trimMessage(currentMessage)
			if currentMessage != "" {
				message := Message{
					From:     from,
					Content:  currentMessage,
					IsHidden: false,
				}

				messages = append(messages, message)
				currentMessage = ""
			}

			// Now process the action
			switch action {
			case "file":
				if _, err := os.Stat(remaining); os.IsNotExist(err) {
					return messages, &MessageFileDoesNotExist{FilePath: remaining}
				}

				messages = append(messages, Message{
					From:     from,
					Content:  fmt.Sprintf("Attached file: %s", remaining),
					IsHidden: false,
				})

				chunks := splitFileIntoDigestibleChunks(remaining)

				for idx, part := range chunks {
					message := Message{
						From:     from,
						Content:  fmt.Sprintf("Attached file (%s) part %d:\n\n%s", remaining, idx, part),
						IsHidden: true,
					}

					messages = append(messages, message)
				}

			case "exec":
				messages = append(messages, Message{
					From:     from,
					Content:  fmt.Sprintf("Executed command: %s", remaining),
					IsHidden: false,
				})

				chunks := splitExecOutputIntoDigestibleChunks(remaining)

				// We want to show the output of the command, so it's not hidden in the UI.
				for idx, part := range chunks {
					message := Message{
						From:     from,
						Content:  fmt.Sprintf("Attached command output (%s) part %d:\n\n%s", remaining, idx, part),
						IsHidden: false,
					}

					messages = append(messages, message)
				}

			default:
				fmt.Printf("unknown action: %s", action)
			}

			continue
		}

		currentMessage += line + "\n"
	}

	// Add any remaining message content
	currentMessage = trimMessage(currentMessage)
	if currentMessage != "" {
		message := Message{
			From:     from,
			Content:  currentMessage,
			IsHidden: false,
		}

		messages = append(messages, message)
		currentMessage = ""
	}

	// Check for errors during scanning
	if err := scanner.Err(); err != nil {
		return messages, err
	}

	return messages, nil
}

// getAction checks if the line is an action (e.g. a slash command indicating a
// file or exec command) and returns the action type and the remaining content
// of the line.
func getAction(line string) (bool, string, string) {
	if strings.HasPrefix(line, "\\f ") {
		return true, "file", strings.TrimPrefix(line, "\\f ")
	}

	if strings.HasPrefix(line, "\\x ") {
		return true, "exec", strings.TrimPrefix(line, "\\x ")
	}

	return false, "", line
}

// Because OpenAI does not support all of the file types that we might care
// about in the line of battle, we send them as part of the conversation
// message. For larger files, this requires splitting the file into smaller
// chunks that can be sent as part of the conversation.
func splitFileIntoDigestibleChunks(filePath string) []string {
	file, err := os.Open(filePath)

	if err != nil {
		return []string{}
	}

	defer file.Close()

	return splitIntoDigestibleChunks(bufio.NewScanner(file))
}

// splitExecOutputIntoDigestibleChunks executes a command and splits the output
// into openai-sized chunks.
func splitExecOutputIntoDigestibleChunks(command string) []string {
	cmd := exec.Command("sh", "-c", command)
	output, err := cmd.CombinedOutput()

	// An error here is not fatal. It's part of the output.
	if err != nil {
		errorMsg := fmt.Sprintf("Failed to execute command: %v", err)
		return []string{errorMsg}
	}

	// Convert any ANSI escape codes to tview tags
	outputStr := tview.TranslateANSI(string(output))

	return splitIntoDigestibleChunks(bufio.NewScanner(strings.NewReader(outputStr)))
}

// splitIntoDigestibleChunks splits the input into chunks that are smaller than
// the OpenAI token limit.
func splitIntoDigestibleChunks(scanner *bufio.Scanner) []string {
	parts := []string{}

	scanner.Split(bufio.ScanRunes)

	var buffer strings.Builder
	currentSize := 0

	for scanner.Scan() {
		runeText := scanner.Text()
		runeSize := len([]byte(runeText))

		// If adding this rune exceeds the max chunk size, start a new chunk
		if currentSize+runeSize > MaxChunkSize {
			parts = append(parts, buffer.String())
			buffer.Reset()
			currentSize = 0
		}

		buffer.WriteString(runeText)
		currentSize += runeSize
	}

	// Add any remaining runes to the final chunk
	if buffer.Len() > 0 {
		parts = append(parts, buffer.String())
	}

	return parts
}
