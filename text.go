package main

import (
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
)

// normalizeText standardises line endings, trims whitespace, and ensures the
// result is valid UTF-8.
func normalizeText(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	text = strings.TrimSpace(text)
	return ensureUTF8(text)
}

// ensureUTF8 returns text unchanged when it is already valid UTF-8.
// Otherwise it attempts to decode from Windows-1252 or Mac Roman before
// falling back to replacing invalid bytes.
func ensureUTF8(text string) string {
	if utf8.ValidString(text) {
		return text
	}

	for _, dec := range []interface{ String(string) (string, error) }{
		charmap.Windows1252.NewDecoder(),
		charmap.Macintosh.NewDecoder(),
	} {
		if decoded, err := dec.String(text); err == nil && utf8.ValidString(decoded) {
			return strings.TrimSpace(decoded)
		}
	}

	logError("clipboard contains invalid UTF-8; replacing invalid bytes with U+FFFD")
	return strings.ToValidUTF8(text, "\uFFFD")
}

// stripMarkdownCodeBlocks removes fenced Markdown code blocks delimited by ```.
// It also supports fences with language names, such as ```go.
func stripMarkdownCodeBlocks(text string) string {
	lines := strings.Split(text, "\n")

	var out []string
	inCodeBlock := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "```") {
			inCodeBlock = !inCodeBlock
			continue
		}

		if inCodeBlock {
			continue
		}

		out = append(out, line)
	}

	return strings.TrimSpace(strings.Join(out, "\n"))
}
