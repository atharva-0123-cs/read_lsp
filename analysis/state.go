package analysis

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/atharva-0123-cs/read_lsp/diagnostics"
	"github.com/atharva-0123-cs/read_lsp/lsp"
)

type document struct {
	text string
}

type State struct {
	mu        sync.RWMutex
	documents map[string]document
}

func NewState() *State {
	return &State{documents: map[string]document{}}
}

func (s *State) OpenDocument(uri, text string) []lsp.Diagnostic {
	s.PutDocument(uri, text)
	return diagnostics.Analyze(text)
}

func (s *State) PutDocument(uri, text string) {
	s.mu.Lock()
	s.documents[uri] = document{text: text}
	s.mu.Unlock()
}

func (s *State) UpdateDocument(uri, text string) []lsp.Diagnostic {
	s.ReplaceDocument(uri, text)
	return diagnostics.Analyze(text)
}

func (s *State) ReplaceDocument(uri, text string) {
	s.mu.Lock()
	s.documents[uri] = document{text: text}
	s.mu.Unlock()
}

func (s *State) CloseDocument(uri string) {
	s.mu.Lock()
	delete(s.documents, uri)
	s.mu.Unlock()
}

func (s *State) Diagnostics(uri string) []lsp.Diagnostic {
	doc, ok := s.document(uri)
	if !ok {
		return nil
	}
	return diagnostics.Analyze(doc.text)
}

func (s *State) DocumentText(uri string) (string, bool) {
	doc, ok := s.document(uri)
	return doc.text, ok
}

func (s *State) document(uri string) (document, bool) {
	s.mu.RLock()
	doc, ok := s.documents[uri]
	s.mu.RUnlock()
	return doc, ok
}

func (s *State) Hover(
	id json.RawMessage,
	uri string,
	position lsp.Position,
) lsp.HoverResponse {
	value := "No Markdown element at cursor."
	if text, ok := s.DocumentText(uri); ok {
		if hovered, found := hoverAt(text, position); found {
			value = hovered
		}
	}

	return lsp.HoverResponse{
		Response: lsp.NewResponse(id),
		Result: lsp.HoverResult{
			Contents: lsp.MarkupContent{
				Kind:  "markdown",
				Value: value,
			},
		},
	}
}

func (s *State) Definition(
	id json.RawMessage,
	uri string,
	position lsp.Position,
) lsp.DefinitionResponse {
	var location *lsp.Location
	if text, ok := s.DocumentText(uri); ok {
		location = definitionLocation(text, uri, position)
	}

	return lsp.DefinitionResponse{
		Response: lsp.NewResponse(id),
		Result:   location,
	}
}

func (s *State) TextDocumentCodeAction(
	id json.RawMessage,
	uri string,
	requestRange lsp.Range,
) lsp.TextDocumentCodeActionResponse {
	actions := []lsp.CodeAction{}
	text, ok := s.DocumentText(uri)
	if !ok {
		return lsp.TextDocumentCodeActionResponse{
			Response: lsp.NewResponse(id),
			Result:   actions,
		}
	}

	source := diagnostics.Parse(text)
	for _, diagnostic := range diagnostics.Analyze(text) {
		if !rangesOverlap(diagnostic.Range, requestRange) {
			continue
		}
		if action, found := codeActionFor(source, text, uri, diagnostic); found {
			actions = append(actions, action)
		}
	}

	return lsp.TextDocumentCodeActionResponse{
		Response: lsp.NewResponse(id),
		Result:   actions,
	}
}

func (s *State) TextDocumentCompletion(
	id json.RawMessage,
	uri string,
	position lsp.Position,
) lsp.CompletionResponse {
	items := []lsp.CompletionItem{}
	if text, ok := s.DocumentText(uri); ok {
		items = completionItems(text, position)
	}

	return lsp.CompletionResponse{
		Response: lsp.NewResponse(id),
		Result:   items,
	}
}

func hoverAt(text string, position lsp.Position) (string, bool) {
	source := diagnostics.Parse(text)
	if position.Line < 0 || position.Line >= len(source.Lines) {
		return "", false
	}

	if fence, ok := source.FenceAt(position.Line); ok {
		return hoverFence(fence), true
	}
	if heading, ok := source.HeadingAt(position.Line); ok {
		return hoverHeading(heading), true
	}
	if definition, ok := source.DefinitionAt(position.Line); ok {
		return hoverDefinition(source, definition), true
	}
	if code, ok := inlineCodeAt(source.Lines[position.Line], position.Character); ok {
		return "**Inline code**\n\n`" + code + "`", true
	}
	if autolink, ok := diagnostics.AutolinkAt(source.Lines[position.Line], position.Character); ok {
		return hoverAutolink(autolink), true
	}
	if tag, ok := diagnostics.HTMLTagAt(source.Lines[position.Line], position.Character); ok {
		return "**Inline HTML**\n\n`" + tag + "`", true
	}
	if link, ok := linkAt(source, position.Line, position.Character); ok {
		return hoverLink(source, link), true
	}

	return "", false
}

func hoverHeading(heading diagnostics.Heading) string {
	parts := []string{
		"**Markdown heading**",
		fmt.Sprintf("Level: `%d`", heading.Level),
		fmt.Sprintf("Text: `%s`", heading.Text),
	}
	if heading.Setext {
		parts = append(parts, "Syntax: Setext")
	} else {
		parts = append(parts, "Syntax: ATX")
	}
	if heading.Slug != "" {
		parts = append(parts, fmt.Sprintf("Anchor: `#%s`", heading.Slug))
	}
	return strings.Join(parts, "\n\n")
}

func hoverFence(fence diagnostics.Fence) string {
	status := "closed"
	if !fence.Closed {
		status = "unclosed"
	}
	info := fence.Info
	if info == "" {
		info = "none"
	}
	return strings.Join([]string{
		"**Fenced code block**",
		fmt.Sprintf("Marker: `%s`", strings.Repeat(string(fence.Marker), fence.Length)),
		fmt.Sprintf("Info string: `%s`", info),
		fmt.Sprintf("Status: %s", status),
	}, "\n\n")
}

func hoverDefinition(source *diagnostics.Document, definition diagnostics.Definition) string {
	parts := []string{
		"**Link reference definition**",
		fmt.Sprintf("Label: `%s`", definition.Label),
	}
	if definition.Destination != "" {
		parts = append(parts, fmt.Sprintf("Destination: `%s`", definition.Destination))
	}
	if definition.Title != "" {
		parts = append(parts, fmt.Sprintf("Title: `%s`", definition.Title))
	}
	if heading, ok := anchorHeading(source, definition.Destination); ok {
		parts = append(parts, fmt.Sprintf("Target heading: `%s` (line %d)", heading.Text, heading.Line+1))
	}
	return strings.Join(parts, "\n\n")
}

func hoverAutolink(link diagnostics.Autolink) string {
	kind := "Autolink"
	if link.Email {
		kind = "Email autolink"
	}
	return strings.Join([]string{
		"**" + kind + "**",
		"Text: `" + link.Text + "`",
		"Destination: `" + link.Destination + "`",
	}, "\n\n")
}

func hoverLink(source *diagnostics.Document, link linkInfo) string {
	kind := "Link"
	if link.Image {
		kind = "Image"
	}

	parts := []string{"**" + kind + "**"}
	if link.Text != "" {
		parts = append(parts, fmt.Sprintf("Text: `%s`", link.Text))
	}
	if link.Reference {
		parts = append(parts, fmt.Sprintf("Reference label: `%s`", link.Label))
	}

	destination := link.Destination
	if link.Reference && link.HasDefinition {
		destination = link.Definition.Destination
		parts = append(parts, fmt.Sprintf("Defined on line %d", link.Definition.Line+1))
	}
	if destination != "" {
		parts = append(parts, fmt.Sprintf("Destination: `%s`", destination))
	}

	title := link.Title
	if link.Reference && link.HasDefinition {
		title = link.Definition.Title
	}
	if title != "" {
		parts = append(parts, fmt.Sprintf("Title: `%s`", title))
	}
	if heading, ok := anchorHeading(source, destination); ok {
		parts = append(parts, fmt.Sprintf("Target heading: `%s` (line %d)", heading.Text, heading.Line+1))
	}

	return strings.Join(parts, "\n\n")
}

func anchorHeading(source *diagnostics.Document, destination string) (diagnostics.Heading, bool) {
	if !strings.HasPrefix(destination, "#") {
		return diagnostics.Heading{}, false
	}
	return source.HeadingForAnchor(destination)
}

type linkInfo struct {
	Image         bool
	Text          string
	Label         string
	Destination   string
	Title         string
	Reference     bool
	HasDefinition bool
	Definition    diagnostics.Definition
	Start         int
	End           int
}

func linkAt(source *diagnostics.Document, lineNumber, character int) (linkInfo, bool) {
	line := source.Lines[lineNumber]
	character = clamp(character, 0, len(line))
	for _, link := range linksInLine(source, lineNumber) {
		if character >= link.Start && character < link.End {
			return link, true
		}
	}
	return linkInfo{}, false
}

func linksInLine(source *diagnostics.Document, lineNumber int) []linkInfo {
	line := source.Lines[lineNumber]
	result := []linkInfo{}

	for index := 0; index < len(line); {
		if line[index] == '\\' {
			index += 2
			continue
		}
		if line[index] == '`' {
			run := backtickRun(line, index)
			closeAt := closingBacktickRun(line, index+run, run)
			if closeAt < 0 {
				index += run
			} else {
				index = closeAt + run
			}
			continue
		}
		if line[index] == '<' {
			if end := closingAngle(line, index+1); end >= 0 {
				index = end + 1
				continue
			}
		}

		start := index
		bracketStart := -1
		image := false
		switch {
		case line[index] == '!' && index+1 < len(line) && line[index+1] == '[':
			bracketStart = index + 1
			image = true
		case line[index] == '[':
			bracketStart = index
		default:
			index++
			continue
		}

		closeBracket := matchingBracket(line, bracketStart)
		if closeBracket < 0 {
			index = bracketStart + 1
			continue
		}

		info := linkInfo{
			Image: image,
			Text:  line[bracketStart+1 : closeBracket],
			Start: start,
		}
		next := closeBracket + 1

		if next < len(line) && line[next] == '(' {
			closeParen := matchingParen(line, next)
			if closeParen < 0 {
				index = next + 1
				continue
			}
			destination, title, _ := diagnostics.ParseDestinationTitle(line[next+1 : closeParen])
			info.Destination = destination
			info.Title = title
			info.End = closeParen + 1
		} else if referenceBracket := spacedReferenceBracket(line, next); referenceBracket >= 0 {
			closeReference := closingReference(line, referenceBracket)
			if closeReference < 0 {
				index = referenceBracket + 1
				continue
			}
			label := strings.TrimSpace(line[referenceBracket+1 : closeReference])
			if label == "" {
				label = strings.TrimSpace(info.Text)
			}
			info.Reference = true
			info.Label = label
			info.End = closeReference + 1
			if definition, ok := source.DefinitionFor(label); ok {
				info.HasDefinition = true
				info.Definition = definition
			}
		} else {
			label := strings.TrimSpace(info.Text)
			definition, ok := source.DefinitionFor(label)
			if !ok {
				index = closeBracket + 1
				continue
			}
			info.Reference = true
			info.Label = label
			info.HasDefinition = true
			info.Definition = definition
			info.End = closeBracket + 1
		}

		result = append(result, info)
		index = info.End
	}

	return result
}

func resolvedDestination(link linkInfo) string {
	if link.Reference && link.HasDefinition {
		return link.Definition.Destination
	}
	return link.Destination
}

func inlineCodeAt(line string, character int) (string, bool) {
	character = clamp(character, 0, len(line))
	for index := 0; index < len(line); {
		if line[index] == '\\' {
			index += 2
			continue
		}
		if line[index] != '`' {
			index++
			continue
		}

		run := backtickRun(line, index)
		closeAt := closingBacktickRun(line, index+run, run)
		if closeAt < 0 {
			return "", false
		}
		end := closeAt + run
		if character >= index && character <= end {
			return strings.TrimSpace(line[index+run : closeAt]), true
		}
		index = end
	}
	return "", false
}

func definitionLocation(text, uri string, position lsp.Position) *lsp.Location {
	source := diagnostics.Parse(text)
	if position.Line < 0 || position.Line >= len(source.Lines) {
		return nil
	}

	if definition, ok := source.DefinitionAt(position.Line); ok {
		return anchorLocation(source, uri, definition.Destination)
	}

	link, ok := linkAt(source, position.Line, position.Character)
	if !ok {
		return nil
	}
	if link.Reference && link.HasDefinition {
		definition := link.Definition
		return &lsp.Location{
			URI:   uri,
			Range: markdownRange(definition.Line, definition.StartCol, definition.EndCol),
		}
	}
	return anchorLocation(source, uri, link.Destination)
}

func anchorLocation(source *diagnostics.Document, uri, destination string) *lsp.Location {
	heading, ok := anchorHeading(source, destination)
	if !ok {
		return nil
	}
	return &lsp.Location{
		URI:   uri,
		Range: markdownRange(heading.Line, heading.StartCol, heading.EndCol),
	}
}

func codeActionFor(source *diagnostics.Document, text, uri string, diagnostic lsp.Diagnostic) (lsp.CodeAction, bool) {
	switch diagnostic.Code {
	case diagnostics.CodeHeadingSpace:
		return editAction(
			"Add a space after the heading marker",
			uri,
			lsp.Range{Start: diagnostic.Range.Start, End: diagnostic.Range.Start},
			" ",
		), true

	case diagnostics.CodeHeadingLevel:
		return editAction(
			"Limit the heading to six levels",
			uri,
			diagnostic.Range,
			"######",
		), true

	case diagnostics.CodeListMarker:
		return editAction(
			"Use a period in the ordered list marker",
			uri,
			diagnostic.Range,
			".",
		), true

	case diagnostics.CodeUnclosedFence:
		fence, ok := source.FenceAt(diagnostic.Range.Start.Line)
		if !ok {
			return lsp.CodeAction{}, false
		}
		closing := strings.Repeat(string(fence.Marker), fence.Length)
		if len(text) > 0 && !strings.HasSuffix(text, "\n") {
			closing = "\n" + closing
		}
		return editAction(
			"Close the fenced code block",
			uri,
			lsp.Range{Start: endPosition(text), End: endPosition(text)},
			closing,
		), true

	case diagnostics.CodeDuplicateReference, diagnostics.CodeInvalidDefinition:
		title := "Remove the duplicate link reference definition"
		if diagnostic.Code == diagnostics.CodeInvalidDefinition {
			title = "Remove the invalid link reference definition"
		}
		return editAction(
			title,
			uri,
			lineDeletionRange(source.Lines, diagnostic.Range.Start.Line),
			"",
		), true

	case diagnostics.CodeUndefinedReference:
		label := textInRange(source.Lines, diagnostic.Range)
		if label == "" {
			return lsp.CodeAction{}, false
		}
		insert := "\n\n[" + label + "]: https://example.com\n"
		if strings.HasSuffix(text, "\n\n") {
			insert = "[" + label + "]: https://example.com\n"
		} else if strings.HasSuffix(text, "\n") {
			insert = "\n[" + label + "]: https://example.com\n"
		}
		return editAction(
			"Add a link reference definition for '"+label+"'",
			uri,
			lsp.Range{Start: endPosition(text), End: endPosition(text)},
			insert,
		), true

	case diagnostics.CodeInlineCode:
		line := source.Lines[diagnostic.Range.Start.Line]
		start := clamp(diagnostic.Range.Start.Character, 0, len(line))
		end := clamp(diagnostic.Range.End.Character, start, len(line))
		insert := line[start:end]
		return editAction(
			"Close the inline code span",
			uri,
			lsp.Range{Start: lsp.Position{Line: diagnostic.Range.Start.Line, Character: len(line)}, End: lsp.Position{Line: diagnostic.Range.Start.Line, Character: len(line)}},
			insert,
		), true

	case diagnostics.CodeUnclosedLink:
		line := source.Lines[diagnostic.Range.Start.Line]
		closing := ")"
		title := "Close the link target"
		if strings.Contains(diagnostic.Message, "reference label") {
			closing = "]"
			title = "Close the reference label"
		}
		return editAction(
			title,
			uri,
			lsp.Range{Start: lsp.Position{Line: diagnostic.Range.Start.Line, Character: len(line)}, End: lsp.Position{Line: diagnostic.Range.Start.Line, Character: len(line)}},
			closing,
		), true

	case diagnostics.CodeAutolink:
		line := source.Lines[diagnostic.Range.Start.Line]
		return editAction(
			"Close the autolink",
			uri,
			lsp.Range{Start: lsp.Position{Line: diagnostic.Range.Start.Line, Character: len(line)}, End: lsp.Position{Line: diagnostic.Range.Start.Line, Character: len(line)}},
			">",
		), true

	case diagnostics.CodeEmphasis:
		line := source.Lines[diagnostic.Range.Start.Line]
		start := clamp(diagnostic.Range.Start.Character, 0, len(line))
		end := clamp(diagnostic.Range.End.Character, start, len(line))
		if start == end {
			return lsp.CodeAction{}, false
		}
		marker := line[start : start+1]
		insert := strings.Repeat(marker, end-start)
		return editAction(
			"Close the emphasis span",
			uri,
			lsp.Range{Start: lsp.Position{Line: diagnostic.Range.Start.Line, Character: len(line)}, End: lsp.Position{Line: diagnostic.Range.Start.Line, Character: len(line)}},
			insert,
		), true

	default:
		return lsp.CodeAction{}, false
	}
}

func editAction(title, uri string, rangeValue lsp.Range, newText string) lsp.CodeAction {
	return lsp.CodeAction{
		Title: title,
		Edit: &lsp.WorkspaceEdit{
			Changes: map[string][]lsp.TextEdit{
				uri: {{Range: rangeValue, NewText: newText}},
			},
		},
	}
}

func completionItems(text string, position lsp.Position) []lsp.CompletionItem {
	lines := strings.Split(text, "\n")
	if position.Line < 0 || position.Line >= len(lines) {
		return []lsp.CompletionItem{}
	}

	line := lines[position.Line]
	character := clamp(position.Character, 0, len(line))
	prefix := line[:character]
	trimmed := strings.TrimLeft(prefix, " \t")
	indent := prefix[:len(prefix)-len(trimmed)]
	marker := strings.TrimSpace(trimmed)
	replace := lsp.Range{
		Start: lsp.Position{Line: position.Line, Character: 0},
		End:   lsp.Position{Line: position.Line, Character: character},
	}

	switch {
	case marker == "":
		return blockCompletionItems(indent, replace)
	case strings.Trim(marker, "#") == "":
		return headingCompletionItems(indent, replace)
	case marker == ">":
		return []lsp.CompletionItem{
			snippetItem("Blockquote", "Markdown blockquote", indent+"> ${1:quote}", replace),
		}
	case marker == "-" || marker == "*" || marker == "+":
		return []lsp.CompletionItem{
			snippetItem("Bulleted list item", "Markdown bulleted list", indent+marker+" ${1:item}", replace),
			snippetItem("Task list item", "Markdown task list", indent+"- [ ] ${1:task}", replace),
		}
	case isOrderedListMarker(marker):
		return []lsp.CompletionItem{
			snippetItem("Numbered list item", "Markdown numbered list", indent+"1. ${1:item}", replace),
		}
	case strings.HasPrefix(marker, "["):
		return []lsp.CompletionItem{
			snippetItem("Inline link", "Markdown inline link", indent+"[${1:link text}](${2:https://example.com})", replace),
			snippetItem("Reference link", "Markdown reference link", indent+"[${1:link text}][${2:label}]", replace),
		}
	case strings.HasPrefix(marker, "!["):
		return []lsp.CompletionItem{
			snippetItem("Image", "Markdown image", indent+"![${1:alt text}](${2:https://example.com/image.png})", replace),
			snippetItem("Reference image", "Markdown reference image", indent+"![${1:alt text}][${2:label}]", replace),
		}
	case strings.HasPrefix(marker, "<"):
		return []lsp.CompletionItem{
			snippetItem("Autolink", "Markdown autolink", indent+"<${1:https://example.com}>", replace),
			snippetItem("Email autolink", "Markdown email autolink", indent+"<${1:name@example.com}>", replace),
		}
	default:
		return []lsp.CompletionItem{}
	}
}

func blockCompletionItems(indent string, replace lsp.Range) []lsp.CompletionItem {
	items := headingCompletionItems(indent, replace)
	items = append(items,
		snippetItem("Inline link", "Markdown inline link", indent+"[${1:link text}](${2:https://example.com})", replace),
		snippetItem("Image", "Markdown image", indent+"![${1:alt text}](${2:https://example.com/image.png})", replace),
		snippetItem("Bold text", "Markdown bold text", indent+"**${1:bold text}**", replace),
		snippetItem("Italic text", "Markdown italic text", indent+"*${1:italic text}*", replace),
		snippetItem("Bulleted list item", "Markdown bulleted list", indent+"- ${1:item}", replace),
		snippetItem("Numbered list item", "Markdown numbered list", indent+"1. ${1:item}", replace),
		snippetItem("Task list item", "Markdown task list", indent+"- [ ] ${1:task}", replace),
		snippetItem("Blockquote", "Markdown blockquote", indent+"> ${1:quote}", replace),
		snippetItem("Fenced code block", "Markdown fenced code block", "```\n${1:code}\n```", replace),
		snippetItem("Link reference definition", "Markdown link reference definition", indent+"[${1:label}]: ${2:https://example.com}", replace),
		snippetItem("Autolink", "Markdown autolink", indent+"<${1:https://example.com}>", replace),
		snippetItem("Email autolink", "Markdown email autolink", indent+"<${1:name@example.com}>", replace),
	)
	return items
}

func headingCompletionItems(indent string, replace lsp.Range) []lsp.CompletionItem {
	items := make([]lsp.CompletionItem, 0, 6)
	for level := 1; level <= 6; level++ {
		items = append(items, snippetItem(
			fmt.Sprintf("Heading %d", level),
			"Markdown ATX heading",
			indent+strings.Repeat("#", level)+" ${1:Heading}",
			replace,
		))
	}
	return items
}

func snippetItem(label, detail, insert string, replace lsp.Range) lsp.CompletionItem {
	return lsp.CompletionItem{
		Label:            label,
		Detail:           detail,
		Documentation:    detail,
		InsertTextFormat: 2,
		TextEdit: &lsp.TextEdit{
			Range:   replace,
			NewText: insert,
		},
	}
}

func rangesOverlap(left, right lsp.Range) bool {
	return !positionBefore(left.End, right.Start) && !positionBefore(right.End, left.Start)
}

func positionBefore(left, right lsp.Position) bool {
	if left.Line != right.Line {
		return left.Line < right.Line
	}
	return left.Character < right.Character
}

func markdownRange(line, start, end int) lsp.Range {
	return lsp.Range{
		Start: lsp.Position{Line: line, Character: start},
		End:   lsp.Position{Line: line, Character: end},
	}
}

func endPosition(text string) lsp.Position {
	lines := strings.Split(text, "\n")
	return lsp.Position{
		Line:      len(lines) - 1,
		Character: len(lines[len(lines)-1]),
	}
}

func lineDeletionRange(lines []string, line int) lsp.Range {
	if line < 0 || line >= len(lines) {
		return lsp.Range{}
	}
	if line+1 < len(lines) {
		return lsp.Range{
			Start: lsp.Position{Line: line, Character: 0},
			End:   lsp.Position{Line: line + 1, Character: 0},
		}
	}
	return lsp.Range{
		Start: lsp.Position{Line: line, Character: 0},
		End:   lsp.Position{Line: line, Character: len(lines[line])},
	}
}

func textInRange(lines []string, rangeValue lsp.Range) string {
	if rangeValue.Start.Line < 0 || rangeValue.Start.Line >= len(lines) {
		return ""
	}
	line := lines[rangeValue.Start.Line]
	start := clamp(rangeValue.Start.Character, 0, len(line))
	end := clamp(rangeValue.End.Character, start, len(line))
	return line[start:end]
}

func matchingBracket(line string, open int) int {
	depth := 0
	escaped := false
	for index := open; index < len(line); index++ {
		if escaped {
			escaped = false
			continue
		}
		switch line[index] {
		case '\\':
			escaped = true
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				return index
			}
		}
	}
	return -1
}

func matchingParen(line string, open int) int {
	depth := 1
	escaped := false
	angle := false
	var quote byte
	for index := open + 1; index < len(line); index++ {
		current := line[index]
		if escaped {
			escaped = false
			continue
		}
		if current == '\\' {
			escaped = true
			continue
		}
		if index == open+1 && current == '<' {
			angle = true
			continue
		}
		if angle {
			if current == '>' {
				angle = false
			}
			continue
		}
		if quote != 0 {
			if current == quote {
				quote = 0
			}
			continue
		}
		if (current == '"' || current == '\'') && (index == open+1 || line[index-1] == ' ' || line[index-1] == '\t') {
			quote = current
			continue
		}
		switch current {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return index
			}
		}
	}
	return -1
}

func spacedReferenceBracket(line string, next int) int {
	if next < len(line) && line[next] == '[' {
		return next
	}
	cursor := next
	for cursor < len(line) && (line[cursor] == ' ' || line[cursor] == '\t') {
		cursor++
	}
	if cursor > next && cursor < len(line) && line[cursor] == '[' {
		return cursor
	}
	return -1
}

func closingReference(line string, open int) int {
	escaped := false
	for index := open + 1; index < len(line); index++ {
		if escaped {
			escaped = false
			continue
		}
		if line[index] == '\\' {
			escaped = true
			continue
		}
		if line[index] == ']' {
			return index
		}
	}
	return -1
}

func closingAngle(line string, start int) int {
	escaped := false
	for index := start; index < len(line); index++ {
		if escaped {
			escaped = false
			continue
		}
		if line[index] == '\\' {
			escaped = true
			continue
		}
		if line[index] == '>' {
			return index
		}
	}
	return -1
}

func backtickRun(line string, start int) int {
	length := 0
	for start+length < len(line) && line[start+length] == '`' {
		length++
	}
	return length
}

func closingBacktickRun(line string, start, length int) int {
	for index := start; index < len(line); {
		if line[index] == '\\' {
			index += 2
			continue
		}
		if line[index] != '`' {
			index++
			continue
		}
		run := backtickRun(line, index)
		if run == length {
			return index
		}
		index += run
	}
	return -1
}

func isOrderedListMarker(value string) bool {
	digits := 0
	for digits < len(value) && value[digits] >= '0' && value[digits] <= '9' {
		digits++
	}
	if digits == 0 || digits != len(value)-1 {
		return false
	}
	return value[digits] == '.' || value[digits] == ')'
}

func clamp(value, minimum, maximum int) int {
	if value < minimum {
		return minimum
	}
	if value > maximum {
		return maximum
	}
	return value
}
