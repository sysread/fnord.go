package util

import (
	"bytes"
	"io"
	"net/http"
	"strings"

	"golang.org/x/net/html"
)

func HttpGet(url string) (io.ReadCloser, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	return resp.Body, nil
}

func HttpGetHTML(url string) (string, error) {
	body, err := HttpGet(url)
	if err != nil {
		return "", err
	}

	defer body.Close()

	html, err := io.ReadAll(body)
	if err != nil {
		return "", err
	}

	return string(html), nil
}

func HttpGetText(url string) (string, error) {
	body, err := HttpGet(url)
	if err != nil {
		return "", err
	}

	defer body.Close()

	// Parse the HTML body directly from the response body
	doc, err := html.Parse(body)
	if err != nil {
		return "", err
	}

	// Extract the innerText
	return extractInnerText(doc), nil
}

// extractInnerText approximates the behavior of document.innerText by
// extracting visible text from an HTML node.
func extractInnerText(n *html.Node) string {
	var buf bytes.Buffer

	var f func(*html.Node)

	f = func(n *html.Node) {
		switch n.Type {
		case html.ElementNode:
			if n.Data == "script" || n.Data == "style" {
				return
			}

		case html.TextNode:
			text := strings.TrimSpace(n.Data)
			if len(text) > 0 {
				if buf.Len() > 0 {
					buf.WriteString(" ")
				}

				buf.WriteString(text + " ")
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}

	f(n)

	return strings.Join(strings.Fields(buf.String()), " ")
}
