package app

import (
	"testing"
)

func TestNormalizeText(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"empty", "", ""},
		{"plain", "hello world", "hello world"},
		{"trim spaces", "  hello  ", "hello"},
		{"CRLF", "line1\r\nline2", "line1\nline2"},
		{"CR only", "line1\rline2", "line1\nline2"},
		{"mixed endings", "a\r\nb\rc", "a\nb\nc"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := normalizeText(tc.input)
			if got != tc.want {
				t.Errorf("normalizeText(%q) = %q; want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestEnsureUTF8_ValidInput(t *testing.T) {
	input := "Héllo wörld 🌍"
	got := ensureUTF8(input)
	if got != input {
		t.Errorf("ensureUTF8 mangled valid UTF-8: got %q", got)
	}
}

func TestStripMarkdownCodeBlocks(t *testing.T) {
	input := `Intro text.

` + "```go" + `
fmt.Println("hello")
` + "```" + `

Outro text.`

	want := `Intro text.


Outro text.`

	got := stripMarkdownCodeBlocks(input)
	if got != want {
		t.Errorf("stripMarkdownCodeBlocks() = %q; want %q", got, want)
	}
}

func TestStripMarkdownCodeBlocks_MultipleBlocks(t *testing.T) {
	input := `Before

` + "```" + `
code one
` + "```" + `

Middle

` + "```js" + `
console.log("two")
` + "```" + `

After`

	want := `Before


Middle


After`

	got := stripMarkdownCodeBlocks(input)
	if got != want {
		t.Errorf("stripMarkdownCodeBlocks() = %q; want %q", got, want)
	}
}
