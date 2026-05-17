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

func TestCleanMarkdownLinks(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "plain text unchanged",
			in:   "normal text without links",
			want: "normal text without links",
		},
		{
			name: "single markdown link",
			in:   `see [Opal Storage Specifications](https://example.com).`,
			want: `see Opal Storage Specifications.`,
		},
		{
			name: "multiple markdown links",
			in:   "read [one](https://one.com) and then [two](https://two.com)",
			want: "read one and then two",
		},
		{
			name: "markdown link with empty label",
			in:   "link [](https://example.com) empty",
			want: "link  empty",
		},
		{
			name: "markdown link with empty url",
			in:   "link [test]() empty",
			want: "link test empty",
		},
		{
			name: "incomplete missing closing bracket",
			in:   "test [incomplete link",
			want: "test [incomplete link",
		},
		{
			name: "incomplete missing opening parenthesis",
			in:   "test [label] without url",
			want: "test [label] without url",
		},
		{
			name: "incomplete missing closing parenthesis",
			in:   "test [label](https://example.com",
			want: "test [label](https://example.com",
		},
		{
			name: "keeps normal brackets",
			in:   "this is [just text] inside brackets",
			want: "this is [just text] inside brackets",
		},
		{
			name: "trims output",
			in:   "  [label](https://example.com)  ",
			want: "label",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cleanMarkdownLinks(tt.in)

			if got != tt.want {
				t.Fatalf("cleanMarkdownLinks(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
