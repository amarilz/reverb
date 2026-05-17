package app

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
	text = ensureUTF8(text)
	text = cleanMarkdownLinks(text)
	return strings.TrimSpace(text)
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

func cleanMarkdownLinks(text string) string {
	var b strings.Builder
	b.Grow(len(text))

	for i := 0; i < len(text); i++ {
		if text[i] != '[' {
			b.WriteByte(text[i])
			continue
		}

		labelStart := i + 1
		labelEnd := strings.IndexByte(text[labelStart:], ']')
		if labelEnd < 0 {
			b.WriteByte(text[i])
			continue
		}
		labelEnd += labelStart

		if labelEnd+1 >= len(text) || text[labelEnd+1] != '(' {
			b.WriteByte(text[i])
			continue
		}

		urlStart := labelEnd + 2
		urlEnd := strings.IndexByte(text[urlStart:], ')')
		if urlEnd < 0 {
			b.WriteByte(text[i])
			continue
		}
		urlEnd += urlStart

		b.WriteString(text[labelStart:labelEnd])
		i = urlEnd
	}

	return strings.TrimSpace(b.String())
}
