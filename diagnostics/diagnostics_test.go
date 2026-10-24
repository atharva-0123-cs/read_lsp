package diagnostics

import (
	"strings"
	"testing"

	"github.com/atharva-0123-cs/read_lsp/lsp"
)

func TestAnalyzeCleanMarkdown(t *testing.T) {
	text := strings.Join([]string{
		"# Example",
		"",
		"## Subtitle",
		"",
		"Paragraph with **bold**, *italic*, `code`, and [link](https://example.com \"Example\").",
		"",
		"- item",
		"> quote",
		"",
		"```go",
		"func main() {}",
		"```",
		"",
		"[ref]: https://example.com \"Title\"",
		"See [ref], [text][ref], and ![alt](image.png).",
	}, "\n")

	if got := Analyze(text); len(got) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", got)
	}
}

func TestAnalyzeUnclosedFence(t *testing.T) {
	got := Analyze("# Example\n```go\nfmt.Println(\"hello\")\n")
	requireCode(t, got, CodeUnclosedFence)
}

func TestAnalyzeHeadingMissingSpace(t *testing.T) {
	got := Analyze("#Heading\n")
	requireCode(t, got, CodeHeadingSpace)
}

func TestAnalyzeHeadingLevelTooDeep(t *testing.T) {
	got := Analyze("####### Heading\n")
	requireCode(t, got, CodeHeadingLevel)
}

func TestAnalyzeLinkTitleContainingParenthesis(t *testing.T) {
	got := Analyze("[x](https://example.com \"a)b\")\n")
	if len(got) != 0 {
		t.Fatalf("expected a valid titled link, got %#v", got)
	}
}

func TestAnalyzeUnclosedLink(t *testing.T) {
	got := Analyze("See [docs](https://example.com\n")
	requireCode(t, got, CodeUnclosedLink)
}

func TestAnalyzeInvalidLink(t *testing.T) {
	got := Analyze("See [docs](https://example.com \"broken\" trailing)\n")
	requireCode(t, got, CodeInvalidLink)
}

func TestAnalyzeUndefinedReference(t *testing.T) {
	got := Analyze("See [guide][missing] and [missing].\n")
	requireCode(t, got, CodeUndefinedReference)
}

func TestAnalyzeDuplicateReference(t *testing.T) {
	got := Analyze("[docs]: https://example.com\n[DOCS]: https://example.org\n")
	requireCode(t, got, CodeDuplicateReference)
}

func TestAnalyzeInvalidDefinition(t *testing.T) {
	got := Analyze("[docs]:\n")
	requireCode(t, got, CodeInvalidDefinition)
}

func TestAnalyzeUnclosedInlineCode(t *testing.T) {
	got := Analyze("Text with `unclosed code\n")
	requireCode(t, got, CodeInlineCode)
}

func TestAnalyzeUnclosedEmphasis(t *testing.T) {
	got := Analyze("Text with *unclosed emphasis\n")
	requireCode(t, got, CodeEmphasis)
}

func TestAnalyzeMissingAnchor(t *testing.T) {
	got := Analyze("# Present\n\nSee [missing](#missing-heading).\n")
	requireCode(t, got, CodeMissingAnchor)
}

func TestAnalyzeMissingAnchorInDefinition(t *testing.T) {
	got := Analyze("# Present\n\n[ref]: #missing-heading\n")
	requireCode(t, got, CodeMissingAnchor)
}

func TestAnalyzeValidAnchor(t *testing.T) {
	got := Analyze("# Present Heading\n\nSee [present](#present-heading).\n")
	if len(got) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", got)
	}
}

func TestAnalyzeIgnoresMarkdownInsideCode(t *testing.T) {
	got := Analyze("```\n#Heading\n[missing][label]\n`unclosed\n```\n")
	if len(got) != 0 {
		t.Fatalf("expected code content to be ignored, got %#v", got)
	}
}

func TestAnalyzeIgnoresLiteralAsterisksBetweenSpaces(t *testing.T) {
	got := Analyze("2 * 3 * 4\n")
	if len(got) != 0 {
		t.Fatalf("expected literal asterisks to be ignored, got %#v", got)
	}
}

func TestAnalyzeHandlesCRLF(t *testing.T) {
	got := Analyze("# Example\r\n\r\nText\r\n")
	if len(got) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", got)
	}
}

func TestAnalyzeAcceptsAutolinksAndHTML(t *testing.T) {
	text := strings.Join([]string{
		"Visit <https://example.com> or <name@example.com>.",
		"",
		"<span data-value=\"[not a link]\">*literal emphasis*</span>",
	}, "\n")

	if got := Analyze(text); len(got) != 0 {
		t.Fatalf("expected autolinks and HTML to be valid, got %#v", got)
	}
}

func TestAnalyzeUnclosedAutolink(t *testing.T) {
	got := Analyze("Visit <https://example.com\n")
	requireCode(t, got, CodeAutolink)
}

func TestAnalyzeReferenceLinkWithSpace(t *testing.T) {
	valid := Analyze("[docs]: https://example.com\n\nSee [documentation] [docs].\n")
	if len(valid) != 0 {
		t.Fatalf("expected spaced reference link to be valid, got %#v", valid)
	}

	invalid := Analyze("See [documentation] [missing].\n")
	requireCode(t, invalid, CodeUndefinedReference)
}

func TestAnalyzeOrderedListMarker(t *testing.T) {
	got := Analyze("1) item\n")
	requireCode(t, got, CodeListMarker)

	valid := Analyze("1. item\n")
	if len(valid) != 0 {
		t.Fatalf("expected period list marker to be valid, got %#v", valid)
	}
}

func TestAnalyzeContainerHeadingMissingSpace(t *testing.T) {
	got := Analyze("> #Bad\n")
	requireCode(t, got, CodeHeadingSpace)
}

func requireCode(t *testing.T, diagnostics []lsp.Diagnostic, code string) {
	t.Helper()

	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return
		}
	}

	t.Fatalf("expected diagnostic with code %q, got %#v", code, diagnostics)
}

func TestAnalyzeIgnoresMarkdownInsideBlockquoteFence(t *testing.T) {
	got := Analyze("> ```\n> #BadHeading\n> [missing][label]\n> ```\n")
	if len(got) != 0 {
		t.Fatalf("expected blockquote fence content to be ignored, got %#v", got)
	}
}

func TestAnalyzeMarkdownGuideBasicSyntax(t *testing.T) {
	text := strings.Join([]string{
		"# ATX Heading",
		"",
		"Setext Heading",
		"==============",
		"",
		"Paragraph with *italic*, **bold**, ***bold italic***, and `inline code`.",
		"",
		"> Blockquote",
		"> > Nested blockquote",
		">",
		"> - Quoted list item",
		"",
		"- Unordered item",
		"- [ ] Task item",
		"",
		"1. Ordered item",
		"2. Another item",
		"",
		"[Inline link](https://example.com \"Title\")",
		"[Full reference][ref] and [collapsed][] and [shortcut].",
		"<https://example.com> and <name@example.com>",
		"![Alt text](image.png)",
		"",
		"---",
		"",
		"```go",
		"# Not a heading",
		"```",
		"",
		"[ref]: https://example.com",
		"[collapsed]: https://example.com",
	}, "\n")

	if got := Analyze(text); len(got) != 0 {
		t.Fatalf("expected basic Markdown syntax to be valid, got %#v", got)
	}
}
