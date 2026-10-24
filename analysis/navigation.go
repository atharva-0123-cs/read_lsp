package analysis

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"

	"github.com/atharva-0123-cs/read_lsp/diagnostics"
	"github.com/atharva-0123-cs/read_lsp/lsp"
)

func (s *State) DocumentSymbols(id json.RawMessage, uri string) lsp.DocumentSymbolResponse {
	result := []lsp.DocumentSymbol{}
	if text, ok := s.DocumentText(uri); ok {
		result = documentSymbols(diagnostics.Parse(text))
	}
	return lsp.DocumentSymbolResponse{
		Response: lsp.NewResponse(id),
		Result:   result,
	}
}

func (s *State) FoldingRanges(id json.RawMessage, uri string) lsp.FoldingRangeResponse {
	result := []lsp.FoldingRange{}
	if text, ok := s.DocumentText(uri); ok {
		result = foldingRanges(diagnostics.Parse(text))
	}
	return lsp.FoldingRangeResponse{
		Response: lsp.NewResponse(id),
		Result:   result,
	}
}

func (s *State) References(
	id json.RawMessage,
	uri string,
	position lsp.Position,
	includeDeclaration bool,
) lsp.ReferenceResponse {
	result := []lsp.Location{}
	if text, ok := s.DocumentText(uri); ok {
		result = referenceLocations(text, uri, position, includeDeclaration)
	}
	return lsp.ReferenceResponse{
		Response: lsp.NewResponse(id),
		Result:   result,
	}
}

type symbolNode struct {
	symbol   lsp.DocumentSymbol
	level    int
	children []*symbolNode
}

func documentSymbols(source *diagnostics.Document) []lsp.DocumentSymbol {
	roots := []*symbolNode{}
	stack := []*symbolNode{}

	for index, heading := range source.Headings {
		if heading.Level > 6 {
			continue
		}
		node := &symbolNode{
			symbol: lsp.DocumentSymbol{
				Name:           heading.Text,
				Detail:         "Heading " + strconv.Itoa(heading.Level),
				Kind:           lsp.SymbolKindString,
				Range:          headingRange(source, index),
				SelectionRange: markdownRange(heading.Line, heading.StartCol, heading.EndCol),
			},
			level: heading.Level,
		}

		for len(stack) > 0 && stack[len(stack)-1].level >= node.level {
			stack = stack[:len(stack)-1]
		}
		if len(stack) == 0 {
			roots = append(roots, node)
		} else {
			parent := stack[len(stack)-1]
			parent.children = append(parent.children, node)
		}
		stack = append(stack, node)
	}

	converted := make([]lsp.DocumentSymbol, 0, len(roots))
	for _, root := range roots {
		converted = append(converted, convertSymbol(root))
	}
	return converted
}

func convertSymbol(node *symbolNode) lsp.DocumentSymbol {
	symbol := node.symbol
	symbol.Children = make([]lsp.DocumentSymbol, 0, len(node.children))
	for _, child := range node.children {
		symbol.Children = append(symbol.Children, convertSymbol(child))
	}
	return symbol
}

func headingRange(source *diagnostics.Document, index int) lsp.Range {
	heading := source.Headings[index]
	endLine := headingSectionEnd(source, index)
	endCharacter := 0
	if endLine >= 0 && endLine < len(source.Lines) {
		endCharacter = len(source.Lines[endLine])
	}
	return lsp.Range{
		Start: lsp.Position{Line: heading.Line, Character: heading.StartCol},
		End:   lsp.Position{Line: endLine, Character: endCharacter},
	}
}

func headingSectionEnd(source *diagnostics.Document, index int) int {
	heading := source.Headings[index]
	end := len(source.Lines) - 1
	for next := index + 1; next < len(source.Headings); next++ {
		other := source.Headings[next]
		if other.Level > 6 || other.Level > heading.Level {
			continue
		}
		end = other.Line - 1
		break
	}
	if heading.Setext && end < heading.Line+1 && heading.Line+1 < len(source.Lines) {
		end = heading.Line + 1
	}
	for end > heading.Line && end < len(source.Lines) && strings.TrimSpace(source.Lines[end]) == "" {
		end--
	}
	if end < heading.Line {
		end = heading.Line
	}
	return end
}

func foldingRanges(source *diagnostics.Document) []lsp.FoldingRange {
	result := []lsp.FoldingRange{}

	for index, heading := range source.Headings {
		if heading.Level > 6 {
			continue
		}
		end := headingSectionEnd(source, index)
		if end <= heading.Line {
			continue
		}
		if heading.Setext && end == heading.Line+1 {
			continue
		}
		result = append(result, lsp.FoldingRange{
			StartLine: heading.Line,
			EndLine:   end,
			Kind:      "region",
		})
	}

	for _, fence := range source.Fences {
		if fence.EndLine <= fence.StartLine {
			continue
		}
		result = append(result, lsp.FoldingRange{
			StartLine: fence.StartLine,
			EndLine:   fence.EndLine,
			Kind:      "comment",
		})
	}

	sort.SliceStable(result, func(left, right int) bool {
		if result[left].StartLine != result[right].StartLine {
			return result[left].StartLine < result[right].StartLine
		}
		return result[left].EndLine < result[right].EndLine
	})
	return result
}

type lineLink struct {
	line int
	link linkInfo
}

type referenceTarget struct {
	kind   string
	label  string
	anchor string
}

func referenceLocations(text, uri string, position lsp.Position, includeDeclaration bool) []lsp.Location {
	source := diagnostics.Parse(text)
	if position.Line < 0 || position.Line >= len(source.Lines) {
		return []lsp.Location{}
	}

	target, ok := referenceTargetAt(source, position)
	if !ok {
		return []lsp.Location{}
	}

	locations := []lsp.Location{}
	seen := map[string]bool{}
	add := func(line, start, end int) {
		key := strconv.Itoa(line) + ":" + strconv.Itoa(start) + ":" + strconv.Itoa(end)
		if seen[key] {
			return
		}
		seen[key] = true
		locations = append(locations, lsp.Location{
			URI:   uri,
			Range: markdownRange(line, start, end),
		})
	}

	links := allLinks(source)
	if target.kind == "label" {
		normalized := diagnostics.NormalizeLabel(target.label)
		if includeDeclaration {
			for _, definition := range source.Definitions {
				if diagnostics.NormalizeLabel(definition.Label) == normalized {
					add(definition.Line, definition.StartCol, definition.EndCol)
				}
			}
		}
		for _, current := range links {
			if current.link.Reference && diagnostics.NormalizeLabel(current.link.Label) == normalized {
				add(current.line, current.link.Start, current.link.End)
			}
		}
	} else {
		normalized := diagnostics.NormalizeAnchor(target.anchor)
		if includeDeclaration {
			for _, heading := range source.Headings {
				if heading.Level <= 6 && heading.Slug == normalized {
					add(heading.Line, heading.StartCol, heading.EndCol)
				}
			}
		}
		for _, definition := range source.Definitions {
			if strings.HasPrefix(definition.Destination, "#") &&
				diagnostics.NormalizeAnchor(definition.Destination) == normalized {
				add(definition.Line, definition.StartCol, definition.EndCol)
			}
		}
		for _, current := range links {
			destination := resolvedDestination(current.link)
			if strings.HasPrefix(destination, "#") &&
				diagnostics.NormalizeAnchor(destination) == normalized {
				add(current.line, current.link.Start, current.link.End)
			}
		}
	}

	sort.SliceStable(locations, func(left, right int) bool {
		leftRange := locations[left].Range
		rightRange := locations[right].Range
		if leftRange.Start.Line != rightRange.Start.Line {
			return leftRange.Start.Line < rightRange.Start.Line
		}
		if leftRange.Start.Character != rightRange.Start.Character {
			return leftRange.Start.Character < rightRange.Start.Character
		}
		return leftRange.End.Character < rightRange.End.Character
	})
	return locations
}

func referenceTargetAt(source *diagnostics.Document, position lsp.Position) (referenceTarget, bool) {
	if definition, ok := source.DefinitionAt(position.Line); ok {
		if strings.HasPrefix(definition.Destination, "#") && position.Character > definition.EndCol {
			return referenceTarget{kind: "anchor", anchor: definition.Destination}, true
		}
		return referenceTarget{kind: "label", label: definition.Label}, true
	}

	if link, ok := linkAt(source, position.Line, position.Character); ok {
		if link.Reference && link.Label != "" {
			return referenceTarget{kind: "label", label: link.Label}, true
		}
		destination := resolvedDestination(link)
		if strings.HasPrefix(destination, "#") {
			return referenceTarget{kind: "anchor", anchor: destination}, true
		}
		return referenceTarget{}, false
	}

	if heading, ok := source.HeadingAt(position.Line); ok && heading.Level <= 6 {
		return referenceTarget{kind: "anchor", anchor: "#" + heading.Slug}, true
	}

	return referenceTarget{}, false
}

func allLinks(source *diagnostics.Document) []lineLink {
	result := []lineLink{}
	for line := range source.Lines {
		if _, insideFence := source.FenceAt(line); insideFence {
			continue
		}
		if source.IsDefinitionLine(line) || source.IsSetextUnderline(line) || source.IsThematicBreak(line) {
			continue
		}
		for _, link := range linksInLine(source, line) {
			result = append(result, lineLink{line: line, link: link})
		}
	}
	return result
}
