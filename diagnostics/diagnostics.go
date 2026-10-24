package diagnostics

import (
	"sort"
	"strconv"
	"strings"

	"github.com/atharva-0123-cs/read_lsp/lsp"
)

const Source = "read_lsp"

const (
	SeverityError   = 1
	SeverityWarning = 2
	SeverityInfo    = 3
	SeverityHint    = 4
)

const (
	CodeHeadingLevel       = "markdown.heading-level"
	CodeHeadingSpace       = "markdown.heading-space"
	CodeUnclosedFence      = "markdown.unclosed-fence"
	CodeInlineCode         = "markdown.inline-code"
	CodeEmphasis           = "markdown.emphasis"
	CodeUnclosedLink       = "markdown.unclosed-link"
	CodeInvalidLink        = "markdown.invalid-link"
	CodeUndefinedReference = "markdown.undefined-reference"
	CodeDuplicateReference = "markdown.duplicate-reference"
	CodeInvalidDefinition  = "markdown.invalid-definition"
	CodeMissingAnchor      = "markdown.missing-anchor"
	CodeAutolink           = "markdown.autolink"
	CodeListMarker         = "markdown.list-marker"
)

// Analyze parses Markdown text and returns sorted syntax diagnostics.
func Analyze(text string) []lsp.Diagnostic {
	document := Parse(text)
	result := []lsp.Diagnostic{}

	result = append(result, headingDiagnostics(document)...)
	result = append(result, fenceDiagnostics(document)...)
	result = append(result, definitionDiagnostics(document)...)
	result = append(result, listDiagnostics(document)...)

	references := []referenceUse{}
	for line := range document.Lines {
		if _, insideFence := document.FenceAt(line); insideFence {
			continue
		}
		if document.IsDefinitionLine(line) || document.IsSetextUnderline(line) || document.IsThematicBreak(line) {
			continue
		}

		lineDiagnostics, lineReferences := inlineDiagnostics(document, line)
		result = append(result, lineDiagnostics...)
		references = append(references, lineReferences...)
	}

	for _, reference := range references {
		if _, defined := document.DefinitionFor(reference.Label); defined {
			continue
		}
		result = append(result, newDiagnostic(
			reference.Line, reference.Start, reference.End, SeverityWarning,
			CodeUndefinedReference,
			"Undefined link reference label '"+reference.Label+"'; add a definition such as ["+reference.Label+"]: https://example.com",
		))
	}

	sortDiagnostics(result)
	return result
}

// headingDiagnostics checks heading level and spacing.
func headingDiagnostics(document *Document) []lsp.Diagnostic {
	result := []lsp.Diagnostic{}

	for _, heading := range document.Headings {
		if heading.Setext {
			continue
		}

		if heading.Level > 6 {
			result = append(result, newDiagnostic(
				heading.Line, heading.StartCol, heading.MarkerEnd, SeverityError,
				CodeHeadingLevel,
				"Markdown headings support at most six levels",
			))
		}

		if !heading.SpaceAfter {
			end := heading.MarkerEnd + 1
			if end > len(document.Lines[heading.Line]) {
				end = heading.MarkerEnd
			}
			result = append(result, newDiagnostic(
				heading.Line, heading.MarkerEnd, end, SeverityWarning,
				CodeHeadingSpace,
				"Add a space etween the '#' characters and the heading text",
			))
		}
	}

	return result
}

// fenceDiagnostics reports fenced code blocks that are not closed.
func fenceDiagnostics(document *Document) []lsp.Diagnostic {
	result := []lsp.Diagnostic{}

	for _, fence := range document.Fences {
		if fence.Closed {
			continue
		}
		result = append(result, newDiagnostic(
			fence.StartLine, fence.StartCol, fence.StartCol+fence.Length, SeverityError,
			CodeUnclosedFence,
			"Unclosed fenced code block; add a closing fence with at least "+strconv.Itoa(fence.Length)+" '"+string(fence.Marker)+"' characters",
		))
	}

	return result
}

// listDiagnostics checks ordered list marker compatibility.
func listDiagnostics(document *Document) []lsp.Diagnostic {
	result := []lsp.Diagnostic{}

	for line := range document.Lines {
		if _, insideFence := document.FenceAt(line); insideFence {
			continue
		}
		content, offset, _ := blockquotePrefix(document.Lines[line])
		indent, marker, ok := ListMarker(content)
		if !ok || !strings.HasSuffix(marker, ")") {
			continue
		}
		result = append(result, newDiagnostic(
			line, offset+indent+len(marker)-1, offset+indent+len(marker), SeverityWarning,
			CodeListMarker,
			"Use a period after the number in an ordered list marker for broader compatibility",
		))
	}

	return result
}

// definitionDiagnostics checks link reference definitions.
func definitionDiagnostics(document *Document) []lsp.Diagnostic {
	result := []lsp.Diagnostic{}
	seen := map[string]Definition{}

	for _, definition := range document.Definitions {
		if !definition.Valid {
			result = append(result, newDiagnostic(
				definition.Line, definition.StartCol, definition.EndCol, SeverityWarning,
				CodeInvalidDefinition,
				"Invalid link reference definition; expected a destination and optional title",
			))
		}
		if definition.Valid && strings.HasPrefix(definition.Destination, "#") && len(definition.Destination) > 1 &&
			!document.AnchorExists(definition.Destination) {
			result = append(result, newDiagnostic(
				definition.Line, definition.StartCol, definition.EndCol, SeverityInfo,
				CodeMissingAnchor,
				"No heading matches the anchor '"+definition.Destination+"'",
			))
		}

		label := NormalizeLabel(definition.Label)
		if first, exists := seen[label]; exists {
			result = append(result, newDiagnostic(
				definition.Line, definition.StartCol, definition.EndCol, SeverityWarning,
				CodeDuplicateReference,
				"Duplicate link reference definition '"+definition.Label+"'; first defined on line "+strconv.Itoa(first.Line+1),
			))
			continue
		}
		seen[label] = definition
	}

	return result
}

// newDiagnostic constructs a diagnostic with a single-line range.
func newDiagnostic(line, start, end, severity int, code, message string) lsp.Diagnostic {
	if start < 0 {
		start = 0
	}
	if end < start {
		end = start
	}

	return lsp.Diagnostic{
		Range:    lineRange(line, start, end),
		Severity: severity,
		Code:     code,
		Source:   Source,
		Message:  message,
	}
}

// lineRange constructs a range on one line.
func lineRange(line, start, end int) lsp.Range {
	return lsp.Range{
		Start: lsp.Position{Line: line, Character: start},
		End:   lsp.Position{Line: line, Character: end},
	}
}

// sortDiagnostics orders diagnostics by line and character.
func sortDiagnostics(diagnostics []lsp.Diagnostic) {
	sort.SliceStable(diagnostics, func(left, right int) bool {
		leftPosition := diagnostics[left].Range.Start
		rightPosition := diagnostics[right].Range.Start
		if leftPosition.Line != rightPosition.Line {
			return leftPosition.Line < rightPosition.Line
		}
		return leftPosition.Character < rightPosition.Character
	})
}
