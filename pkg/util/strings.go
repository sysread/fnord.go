package util

import (
	"bufio"
	"strings"
)

// TrimMessage trims leading and trailing whitespace from a message's content.
func TrimMessage(content string) string {
	content = strings.TrimLeft(content, " \r\n\t\f")
	content = strings.TrimRight(content, " \r\n\t\f")
	return content
}

// Chunkify splits a string into chunks of a given size. The final chunk may be
// smaller than the specified size.
func Chunkify(scanner *bufio.Scanner, chunkSize int) []string {
	parts := []string{}

	scanner.Split(bufio.ScanRunes)

	var buffer strings.Builder
	currentSize := 0

	for scanner.Scan() {
		runeText := scanner.Text()
		runeSize := len([]byte(runeText))

		// If adding this rune exceeds the max chunk size, start a new chunk
		if currentSize+runeSize > chunkSize {
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
