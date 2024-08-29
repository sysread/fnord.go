package chat_manager

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/rivo/tview"

	"github.com/sysread/fnord/pkg/messages"
	"github.com/sysread/fnord/pkg/util"
)

// OpenAI has a token limit per request. When a file is too large to send as
// part of a conversation message, we can split it up into smaller chunks. 30k
// is a safe limit, lower than needed to avoid hitting the token limit.
const MaxChunkSize = 30_000

// MessageFileDoesNotExist is an error type that is returned when a file
// referenced in a slash command (e.g., `\f`) does not exist.
type MessageFileDoesNotExist struct {
	FilePath string
}

// Error returns the error message for a MessageFileDoesNotExist error.
func (e *MessageFileDoesNotExist) Error() string {
	return fmt.Sprintf("file does not exist: %s", e.FilePath)
}

func ParseMessage(from messages.Sender, content string) ([]messages.Message, error) {
	msgList := []messages.Message{}
	scanner := bufio.NewScanner(strings.NewReader(content))

	currentMessage := ""
	for scanner.Scan() {
		line := scanner.Text()

		isAction, action, remaining := getAction(line)
		if isAction {
			// Any built up text is a message. Add it and reset the current
			// message buffer.
			currentMessage = util.TrimMessage(currentMessage)
			if currentMessage != "" {
				msgList = append(msgList, messages.NewMessage(from, currentMessage, false))
				currentMessage = ""
			}

			// Now process the action
			switch action {
			case "file":
				if _, err := os.Stat(remaining); os.IsNotExist(err) {
					return msgList, &MessageFileDoesNotExist{FilePath: remaining}
				}

				msgList = append(msgList, messages.NewMessage(from, fmt.Sprintf("Attached file: %s", remaining), false))

				chunks := splitFileIntoDigestibleChunks(remaining)

				for idx, part := range chunks {
					content := fmt.Sprintf("Attached file (%s) part %d:\n\n%s", remaining, idx, part)
					msgList = append(msgList, messages.NewMessage(from, content, true))
				}

			case "exec":
				content := fmt.Sprintf("Executed command: %s", remaining)
				msgList = append(msgList, messages.NewMessage(from, content, false))

				chunks := splitExecOutputIntoDigestibleChunks(remaining)

				// We want to show the output of the command, so it's not hidden in the UI.
				for idx, part := range chunks {
					content := fmt.Sprintf("Attached command output (%s) part %d:\n\n%s", remaining, idx, part)
					msgList = append(msgList, messages.NewMessage(from, content, false))
				}

			default:
				fmt.Printf("unknown action: %s", action)
			}

			continue
		}

		currentMessage += line + "\n"
	}

	// Add any remaining message content
	currentMessage = util.TrimMessage(currentMessage)
	if currentMessage != "" {
		msgList = append(msgList, messages.NewMessage(from, currentMessage, false))
		currentMessage = ""
	}

	// Check for errors during scanning
	if err := scanner.Err(); err != nil {
		return msgList, err
	}

	return msgList, nil
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

	return util.Chunkify(bufio.NewScanner(file), MaxChunkSize)
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

	return util.Chunkify(bufio.NewScanner(strings.NewReader(outputStr)), MaxChunkSize)
}
