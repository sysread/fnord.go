package util

import (
	"strings"
	"testing"

	"golang.org/x/net/html"
)

func TestExtractInnerText(t *testing.T) {
	tests := []struct {
		htmlInput    string
		expectedText string
	}{
		{
			htmlInput:    `<p>Hello World</p>`,
			expectedText: "Hello World",
		},
		{
			htmlInput:    `<div><p>Hello</p><p>World</p></div>`,
			expectedText: "Hello World",
		},
		{
			htmlInput:    `<div>Hello <span>World</span></div>`,
			expectedText: "Hello World",
		},
		{
			htmlInput:    `<div>Hello <script>console.log("World")</script></div>`,
			expectedText: "Hello",
		},
		{
			htmlInput:    `<div><style>body { color: red; }</style><p>Hello</p> World</div>`,
			expectedText: "Hello World",
		},
		{
			htmlInput:    `<div>  Hello    <span>World</span> </div>`,
			expectedText: "Hello World",
		},
	}

	for _, tt := range tests {
		doc, err := html.Parse(strings.NewReader(tt.htmlInput))
		if err != nil {
			t.Fatalf("Failed to parse HTML: %v", err)
		}

		actualText := extractInnerText(doc)
		if actualText != tt.expectedText {
			t.Errorf("extractInnerText(%q) = %q; want %q", tt.htmlInput, actualText, tt.expectedText)
		}
	}
}

