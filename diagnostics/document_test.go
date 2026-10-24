package diagnostics

import "testing"

func TestParseHeadingsAndSlugs(t *testing.T) {
	document := Parse("# Hello, World!\n\n## Hello, World!\n\nTitle\n=====\n\nSubtitle\n---\n")

	if len(document.Headings) != 4 {
		t.Fatalf("expected 4 headings, got %#v", document.Headings)
	}

	expected := []struct {
		level int
		text  string
		slug  string
	}{
		{1, "Hello, World!", "hello-world"},
		{2, "Hello, World!", "hello-world-1"},
		{1, "Title", "title"},
		{2, "Subtitle", "subtitle"},
	}

	for index, want := range expected {
		got := document.Headings[index]
		if got.Level != want.level || got.Text != want.text || got.Slug != want.slug {
			t.Fatalf("heading %d: expected %#v, got %#v", index, want, got)
		}
	}
}

func TestParseIgnoresFencedCode(t *testing.T) {
	text := "# Real\n```go\n# Not a heading\n[key]: https://example.com\n```\n"
	document := Parse(text)

	if len(document.Headings) != 1 || document.Headings[0].Text != "Real" {
		t.Fatalf("expected only the real heading, got %#v", document.Headings)
	}
	if len(document.Definitions) != 0 {
		t.Fatalf("expected definitions inside fences to be ignored, got %#v", document.Definitions)
	}
	if len(document.Fences) != 1 || !document.Fences[0].Closed {
		t.Fatalf("expected one closed fence, got %#v", document.Fences)
	}
	if _, ok := document.HeadingAt(2); ok {
		t.Fatal("expected fenced heading line to be ignored")
	}
}

func TestParseDefinitions(t *testing.T) {
	text := "[docs]: https://example.com \"Documentation\"\n[empty]:\n"
	document := Parse(text)

	if len(document.Definitions) != 2 {
		t.Fatalf("expected two definitions, got %#v", document.Definitions)
	}

	definition, ok := document.DefinitionFor(" DOCS ")
	if !ok {
		t.Fatal("expected normalized label lookup")
	}
	if definition.Destination != "https://example.com" || definition.Title != "Documentation" {
		t.Fatalf("unexpected definition: %#v", definition)
	}
	if document.Definitions[1].Valid {
		t.Fatal("expected empty definition to be invalid")
	}
}

func TestParseSetextUnderlineIsNotHeadingText(t *testing.T) {
	document := Parse("Title\n===\n---\n")

	if len(document.Headings) != 1 {
		t.Fatalf("expected one heading, got %#v", document.Headings)
	}
	if document.Headings[0].Level != 1 || document.Headings[0].Text != "Title" {
		t.Fatalf("unexpected heading: %#v", document.Headings[0])
	}
	if !document.IsThematicBreak(2) {
		t.Fatal("expected line 2 to remain a thematic break")
	}
}

func TestParseFenceAndThematicBreak(t *testing.T) {
	text := "~~~\ncode\n~~~\n\n---\n"
	document := Parse(text)

	if len(document.Fences) != 1 || document.Fences[0].Info != "" || !document.Fences[0].Closed {
		t.Fatalf("unexpected fences: %#v", document.Fences)
	}
	if !document.IsThematicBreak(4) {
		t.Fatal("expected line 4 to be a thematic break")
	}
}

func TestParseDestinationTitle(t *testing.T) {
	tests := []struct {
		input       string
		destination string
		title       string
		valid       bool
	}{
		{`https://example.com "Example"`, "https://example.com", "Example", true},
		{`<https://example.com/a(b)> 'Title'`, "https://example.com/a(b)", "Title", true},
		{`https://example.com`, "https://example.com", "", true},
		{``, "", "", false},
		{`https://example.com "unterminated`, "https://example.com", "", false},
	}

	for _, test := range tests {
		destination, title, valid := ParseDestinationTitle(test.input)
		if destination != test.destination || title != test.title || valid != test.valid {
			t.Fatalf("ParseDestinationTitle(%q) = %q, %q, %v", test.input, destination, title, valid)
		}
	}
}

func TestParseContainerHeadingsAndDefinitions(t *testing.T) {
	text := "> # Quoted Heading\n> Title\n> =====\n> - # Listed Heading\n> [ref]: #quoted-heading\n    # Not A Heading\n"
	document := Parse(text)

	if len(document.Headings) != 3 {
		t.Fatalf("expected three container headings, got %#v", document.Headings)
	}
	if document.Headings[0].Text != "Quoted Heading" || document.Headings[0].StartCol != 2 {
		t.Fatalf("unexpected blockquote heading: %#v", document.Headings[0])
	}
	if document.Headings[1].Text != "Title" || !document.Headings[1].Setext || document.Headings[1].StartCol != 2 {
		t.Fatalf("unexpected blockquote Setext heading: %#v", document.Headings[1])
	}
	if document.Headings[2].Text != "Listed Heading" || document.Headings[2].StartCol != 4 {
		t.Fatalf("unexpected list heading: %#v", document.Headings[2])
	}
	if len(document.Definitions) != 1 || document.Definitions[0].StartCol != 2 {
		t.Fatalf("unexpected container definition: %#v", document.Definitions)
	}
}

func TestSlugify(t *testing.T) {
	if got := Slugify("Markdown Guide: Basic Syntax"); got != "markdown-guide-basic-syntax" {
		t.Fatalf("unexpected slug %q", got)
	}
}

func TestParseFenceInsideBlockquote(t *testing.T) {
	text := "> ```go\n> # Not A Heading\n> ```\n\n# Real Heading\n"
	document := Parse(text)

	if len(document.Fences) != 1 || !document.Fences[0].Closed {
		t.Fatalf("expected one closed blockquote fence, got %#v", document.Fences)
	}
	if document.Fences[0].StartLine != 0 || document.Fences[0].EndLine != 2 {
		t.Fatalf("unexpected blockquote fence range: %#v", document.Fences[0])
	}
	if len(document.Headings) != 1 || document.Headings[0].Text != "Real Heading" {
		t.Fatalf("expected only the real heading, got %#v", document.Headings)
	}
}

func TestTopLevelFenceIsNotClosedByQuotedFence(t *testing.T) {
	document := Parse("```\n> ```\ncode\n```\n")

	if len(document.Fences) != 1 || !document.Fences[0].Closed || document.Fences[0].EndLine != 3 {
		t.Fatalf("unexpected fence: %#v", document.Fences)
	}
}

func TestParseThematicBreaksWithListLikeMarkers(t *testing.T) {
	document := Parse("* * *\n\n- - -\n\n> _ _ _\n")

	for _, line := range []int{0, 2, 4} {
		if !document.IsThematicBreak(line) {
			t.Fatalf("expected line %d to be a thematic break", line)
		}
	}
}
