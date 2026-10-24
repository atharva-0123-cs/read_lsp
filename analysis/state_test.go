package analysis

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/atharva-0123-cs/read_lsp/lsp"
)

func TestHoverShowsHeading(t *testing.T) {
	state := NewState()
	uri := "file:///heading.md"
	state.OpenDocument(uri, "# Hello World\n")

	response := state.Hover(json.RawMessage(`1`), uri, lsp.Position{Line: 0, Character: 3})
	value := response.Result.Contents.Value

	for _, expected := range []string{"Markdown heading", "Level: `1`", "Anchor: `#hello-world`"} {
		if !strings.Contains(value, expected) {
			t.Fatalf("expected %q in hover, got:\n%s", expected, value)
		}
	}
}

func TestHoverShowsFencedCodeBlock(t *testing.T) {
	state := NewState()
	uri := "file:///fence.md"
	state.OpenDocument(uri, "```go\nfmt.Println(\"hello\")\n```\n")

	response := state.Hover(json.RawMessage(`1`), uri, lsp.Position{Line: 1, Character: 2})
	value := response.Result.Contents.Value

	if !strings.Contains(value, "Fenced code block") || !strings.Contains(value, "Info string: `go`") {
		t.Fatalf("unexpected fence hover:\n%s", value)
	}
}

func TestHoverShowsInlineLink(t *testing.T) {
	state := NewState()
	uri := "file:///link.md"
	state.OpenDocument(uri, "See [docs](https://example.com \"Title\").\n")

	response := state.Hover(json.RawMessage(`1`), uri, lsp.Position{Line: 0, Character: 6})
	value := response.Result.Contents.Value

	for _, expected := range []string{"Link", "Text: `docs`", "https://example.com", "Title"} {
		if !strings.Contains(value, expected) {
			t.Fatalf("expected %q in hover, got:\n%s", expected, value)
		}
	}
}

func TestHoverShowsReferenceDefinition(t *testing.T) {
	state := NewState()
	uri := "file:///reference.md"
	state.OpenDocument(uri, "[docs]: https://example.com \"Title\"\nSee [docs].\n")

	response := state.Hover(json.RawMessage(`1`), uri, lsp.Position{Line: 1, Character: 6})
	value := response.Result.Contents.Value

	for _, expected := range []string{"Reference label: `docs`", "Defined on line 1", "https://example.com"} {
		if !strings.Contains(value, expected) {
			t.Fatalf("expected %q in hover, got:\n%s", expected, value)
		}
	}
}

func TestHoverShowsInlineCode(t *testing.T) {
	state := NewState()
	uri := "file:///code.md"
	state.OpenDocument(uri, "Use `go test` here.\n")

	response := state.Hover(json.RawMessage(`1`), uri, lsp.Position{Line: 0, Character: 7})
	value := response.Result.Contents.Value

	if !strings.Contains(value, "Inline code") || !strings.Contains(value, "`go test`") {
		t.Fatalf("unexpected inline code hover:\n%s", value)
	}
}

func TestDefinitionResolvesReference(t *testing.T) {
	state := NewState()
	uri := "file:///reference.md"
	state.OpenDocument(uri, "[docs]: https://example.com\n\nSee [docs].\n")

	response := state.Definition(json.RawMessage(`1`), uri, lsp.Position{Line: 2, Character: 6})
	if response.Result == nil {
		t.Fatal("expected a definition location")
	}
	if response.Result.Range.Start.Line != 0 || response.Result.Range.Start.Character != 0 {
		t.Fatalf("unexpected definition location: %#v", response.Result)
	}
}

func TestDefinitionResolvesAnchor(t *testing.T) {
	state := NewState()
	uri := "file:///anchor.md"
	state.OpenDocument(uri, "# Target Heading\n\nSee [jump](#target-heading).\n")

	response := state.Definition(json.RawMessage(`1`), uri, lsp.Position{Line: 2, Character: 6})
	if response.Result == nil {
		t.Fatal("expected an anchor definition location")
	}
	if response.Result.Range.Start.Line != 0 {
		t.Fatalf("unexpected anchor location: %#v", response.Result)
	}
}

func TestDefinitionResolvesAnchorFromDefinitionLine(t *testing.T) {
	state := NewState()
	uri := "file:///anchor.md"
	state.OpenDocument(uri, "# Target\n\n[ref]: #target\n")

	response := state.Definition(json.RawMessage(`1`), uri, lsp.Position{Line: 2, Character: 8})
	if response.Result == nil || response.Result.Range.Start.Line != 0 {
		t.Fatalf("unexpected definition location: %#v", response.Result)
	}
}

func TestDefinitionReturnsNullWhenNothingMatches(t *testing.T) {
	state := NewState()
	uri := "file:///empty.md"
	state.OpenDocument(uri, "Plain text.\n")

	response := state.Definition(json.RawMessage(`1`), uri, lsp.Position{Line: 0, Character: 2})
	if response.Result != nil {
		t.Fatalf("expected nil definition, got %#v", response.Result)
	}
}

func TestCodeActionAddsHeadingSpace(t *testing.T) {
	state := NewState()
	uri := "file:///heading.md"
	state.OpenDocument(uri, "#Bad\n")

	response := state.TextDocumentCodeAction(
		json.RawMessage(`1`),
		uri,
		lsp.Range{Start: lsp.Position{Line: 0, Character: 0}, End: lsp.Position{Line: 0, Character: 4}},
	)
	if len(response.Result) == 0 {
		t.Fatal("expected heading code action")
	}

	action := response.Result[0]
	if action.Edit == nil {
		t.Fatal("expected an edit")
	}
	edits := action.Edit.Changes[uri]
	if len(edits) != 1 || edits[0].NewText != " " {
		t.Fatalf("unexpected edits: %#v", edits)
	}
}

func TestCodeActionClosesFence(t *testing.T) {
	state := NewState()
	uri := "file:///fence.md"
	state.OpenDocument(uri, "# Example\n```go\ncode\n")

	response := state.TextDocumentCodeAction(
		json.RawMessage(`1`),
		uri,
		lsp.Range{Start: lsp.Position{Line: 1, Character: 0}, End: lsp.Position{Line: 1, Character: 3}},
	)
	if len(response.Result) == 0 {
		t.Fatal("expected fence code action")
	}
	edits := response.Result[0].Edit.Changes[uri]
	if len(edits) != 1 || !strings.HasSuffix(edits[0].NewText, "```") {
		t.Fatalf("unexpected fence edit: %#v", edits)
	}
}

func TestCodeActionAddsUndefinedReferenceDefinition(t *testing.T) {
	state := NewState()
	uri := "file:///reference.md"
	state.OpenDocument(uri, "See [missing][label].\n")

	response := state.TextDocumentCodeAction(
		json.RawMessage(`1`),
		uri,
		lsp.Range{Start: lsp.Position{Line: 0, Character: 0}, End: lsp.Position{Line: 0, Character: 20}},
	)
	if len(response.Result) == 0 {
		t.Fatal("expected reference code action")
	}
	edits := response.Result[0].Edit.Changes[uri]
	if len(edits) != 1 || !strings.Contains(edits[0].NewText, "[label]:") {
		t.Fatalf("unexpected reference edit: %#v", edits)
	}
}

func TestCompletionOffersHeadingSnippets(t *testing.T) {
	state := NewState()
	uri := "file:///completion.md"
	state.OpenDocument(uri, "#")

	response := state.TextDocumentCompletion(json.RawMessage(`1`), uri, lsp.Position{Line: 0, Character: 1})
	if len(response.Result) != 6 {
		t.Fatalf("expected six heading completions, got %#v", response.Result)
	}

	first := response.Result[0]
	if first.TextEdit == nil || first.TextEdit.NewText != "# ${1:Heading}" {
		t.Fatalf("unexpected completion: %#v", first)
	}
	if first.InsertTextFormat != 2 {
		t.Fatalf("expected snippet format, got %#v", first)
	}
}

func TestCompletionOffersBlockSnippetsOnEmptyLine(t *testing.T) {
	state := NewState()
	uri := "file:///completion.md"
	state.OpenDocument(uri, "")

	response := state.TextDocumentCompletion(json.RawMessage(`1`), uri, lsp.Position{Line: 0, Character: 0})
	if len(response.Result) < 10 {
		t.Fatalf("expected block completions, got %#v", response.Result)
	}
}

func TestCompletionOffersLinksAndImages(t *testing.T) {
	state := NewState()
	uri := "file:///completion.md"
	state.OpenDocument(uri, "[\n![")

	linkResponse := state.TextDocumentCompletion(json.RawMessage(`1`), uri, lsp.Position{Line: 0, Character: 1})
	if len(linkResponse.Result) != 2 {
		t.Fatalf("expected link completions, got %#v", linkResponse.Result)
	}

	imageResponse := state.TextDocumentCompletion(json.RawMessage(`2`), uri, lsp.Position{Line: 1, Character: 2})
	if len(imageResponse.Result) != 2 {
		t.Fatalf("expected image completions, got %#v", imageResponse.Result)
	}
}

func TestCompletionOffersOrderedListSnippet(t *testing.T) {
	state := NewState()
	uri := "file:///completion.md"
	state.OpenDocument(uri, "1.")

	response := state.TextDocumentCompletion(json.RawMessage(`1`), uri, lsp.Position{Line: 0, Character: 2})
	if len(response.Result) != 1 || response.Result[0].Label != "Numbered list item" {
		t.Fatalf("unexpected ordered list completions: %#v", response.Result)
	}
}

func TestCompletionIsEmptyInPlainText(t *testing.T) {
	state := NewState()
	uri := "file:///completion.md"
	state.OpenDocument(uri, "plain text")

	response := state.TextDocumentCompletion(json.RawMessage(`1`), uri, lsp.Position{Line: 0, Character: 5})
	if len(response.Result) != 0 {
		t.Fatalf("expected no completions, got %#v", response.Result)
	}
}

func TestStateConcurrentDocumentAccess(t *testing.T) {
	state := NewState()
	var wait sync.WaitGroup

	for worker := 0; worker < 32; worker++ {
		wait.Add(1)
		go func(worker int) {
			defer wait.Done()
			for iteration := 0; iteration < 200; iteration++ {
				uri := fmt.Sprintf("file:///document-%d.md", (worker+iteration)%8)
				state.OpenDocument(uri, "# Heading\n")
				state.UpdateDocument(uri, "# Updated\n")
				_ = state.Diagnostics(uri)
				_ = state.Hover(json.RawMessage(`1`), uri, lsp.Position{Line: 0, Character: 2})
				_ = state.Definition(json.RawMessage(`1`), uri, lsp.Position{Line: 0, Character: 2})
				state.CloseDocument(uri)
			}
		}(worker)
	}

	wait.Wait()
}

func TestHoverShowsAutolinkAndHTML(t *testing.T) {
	state := NewState()
	uri := "file:///autolink.md"
	state.OpenDocument(uri, "Visit <https://example.com> and <span>HTML</span>.\n")

	autolink := state.Hover(json.RawMessage(`1`), uri, lsp.Position{Line: 0, Character: 8})
	if !strings.Contains(autolink.Result.Contents.Value, "Autolink") ||
		!strings.Contains(autolink.Result.Contents.Value, "https://example.com") {
		t.Fatalf("unexpected autolink hover:\n%s", autolink.Result.Contents.Value)
	}

	html := state.Hover(json.RawMessage(`2`), uri, lsp.Position{Line: 0, Character: 36})
	if !strings.Contains(html.Result.Contents.Value, "Inline HTML") ||
		!strings.Contains(html.Result.Contents.Value, "<span>") {
		t.Fatalf("unexpected HTML hover:\n%s", html.Result.Contents.Value)
	}
}

func TestHoverShowsSpacedReferenceLink(t *testing.T) {
	state := NewState()
	uri := "file:///reference.md"
	state.OpenDocument(uri, "[docs]: https://example.com\n\nSee [documentation] [docs].\n")

	response := state.Hover(json.RawMessage(`1`), uri, lsp.Position{Line: 2, Character: 8})
	value := response.Result.Contents.Value

	for _, expected := range []string{"Reference label: `docs`", "https://example.com"} {
		if !strings.Contains(value, expected) {
			t.Fatalf("expected %q in hover, got:\n%s", expected, value)
		}
	}
}

func TestDocumentSymbolsNestHeadings(t *testing.T) {
	state := NewState()
	uri := "file:///symbols.md"
	state.OpenDocument(uri, "# Top\n\n> ## Child\n\n### Grandchild\n\n# Second\n")

	response := state.DocumentSymbols(json.RawMessage(`1`), uri)
	if len(response.Result) != 2 {
		t.Fatalf("expected two root symbols, got %#v", response.Result)
	}
	if response.Result[0].Name != "Top" || len(response.Result[0].Children) != 1 {
		t.Fatalf("unexpected first symbol: %#v", response.Result[0])
	}
	child := response.Result[0].Children[0]
	if child.Name != "Child" || len(child.Children) != 1 || child.Children[0].Name != "Grandchild" {
		t.Fatalf("unexpected nested symbols: %#v", child)
	}
	if response.Result[1].Name != "Second" {
		t.Fatalf("unexpected second symbol: %#v", response.Result[1])
	}
}

func TestFoldingRangesCoverHeadingsAndFences(t *testing.T) {
	state := NewState()
	uri := "file:///folding.md"
	state.OpenDocument(uri, "# Top\n\ntext\n\n```go\ncode\n```\n")

	response := state.FoldingRanges(json.RawMessage(`1`), uri)
	if len(response.Result) < 2 {
		t.Fatalf("expected heading and fence folds, got %#v", response.Result)
	}

	foundFence := false
	for _, current := range response.Result {
		if current.StartLine == 4 && current.EndLine == 6 {
			foundFence = true
		}
	}
	if !foundFence {
		t.Fatalf("expected fenced code fold, got %#v", response.Result)
	}
}

func TestReferencesForLabel(t *testing.T) {
	state := NewState()
	uri := "file:///references.md"
	state.OpenDocument(uri, "[docs]: https://example.com\n\nSee [docs] and [text][docs].\n")

	response := state.References(json.RawMessage(`1`), uri, lsp.Position{Line: 2, Character: 5}, true)
	if len(response.Result) != 3 {
		t.Fatalf("expected definition and two uses, got %#v", response.Result)
	}

	withoutDeclaration := state.References(json.RawMessage(`2`), uri, lsp.Position{Line: 2, Character: 5}, false)
	if len(withoutDeclaration.Result) != 2 {
		t.Fatalf("expected two uses without declaration, got %#v", withoutDeclaration.Result)
	}
}

func TestReferencesForAnchorFromHeading(t *testing.T) {
	state := NewState()
	uri := "file:///anchors.md"
	state.OpenDocument(uri, "# Target Heading\n\nSee [jump](#target-heading) and [ref][target].\n\n[target]: #target-heading\n")

	response := state.References(json.RawMessage(`1`), uri, lsp.Position{Line: 0, Character: 3}, true)
	if len(response.Result) != 4 {
		t.Fatalf("expected heading, two links, and anchor definition, got %#v", response.Result)
	}

	withoutDeclaration := state.References(json.RawMessage(`2`), uri, lsp.Position{Line: 0, Character: 3}, false)
	if len(withoutDeclaration.Result) != 3 {
		t.Fatalf("expected two links and anchor definition, got %#v", withoutDeclaration.Result)
	}
}

func TestCodeActionFixesOrderedListMarker(t *testing.T) {
	state := NewState()
	uri := "file:///list.md"
	state.OpenDocument(uri, "1) item\n")

	response := state.TextDocumentCodeAction(
		json.RawMessage(`1`),
		uri,
		lsp.Range{Start: lsp.Position{Line: 0, Character: 1}, End: lsp.Position{Line: 0, Character: 2}},
	)
	if len(response.Result) == 0 {
		t.Fatal("expected list marker code action")
	}
	edits := response.Result[0].Edit.Changes[uri]
	if len(edits) != 1 || edits[0].NewText != "." || edits[0].Range.Start.Character != 1 {
		t.Fatalf("unexpected list marker edit: %#v", edits)
	}
}
