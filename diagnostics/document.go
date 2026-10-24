package diagnostics

import (
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

var (
	definitionPattern = regexp.MustCompile(`^ {0,3}\[([^\]]+)\]:[ \t]*(.*)$`)
	setextPattern     = regexp.MustCompile(`^ {0,3}(=+|-+)[ \t]*$`)
)

// Document is the parsed Markdown model for one source text.
type Document struct {
	Lines       []string
	Headings    []Heading
	Definitions []Definition
	Fences      []Fence

	headingAt    map[int]int
	definitionAt map[int]int
	fenceAt      map[int]int
	setextAt     map[int]bool
	thematicAt   map[int]bool

	definitionsByLabel map[string]int
}

// Heading describes an ATX or Setext heading.
type Heading struct {
	Level      int
	Text       string
	Line       int
	StartCol   int
	EndCol     int
	MarkerEnd  int
	SpaceAfter bool
	Setext     bool
	Slug       string
}

// Definition describes a link reference definition.
type Definition struct {
	Label       string
	Destination string
	Title       string
	Line        int
	StartCol    int
	EndCol      int
	Valid       bool
}

// Fence describes a fenced code block.
type Fence struct {
	StartLine int
	EndLine   int
	Marker    byte
	Length    int
	Info      string
	StartCol  int
	Closed    bool
}

// fenceOpen is the parsed opening marker of a fenced code block.
type fenceOpen struct {
	marker   byte
	length   int
	info     string
	startCol int
}

// atxHeading is the parsed form of an ATX heading line.
type atxHeading struct {
	level      int
	text       string
	startCol   int
	markerEnd  int
	spaceAfter bool
}

// Parse builds a Markdown document model from source text.
func Parse(text string) *Document {
	lines := splitLines(text)
	doc := &Document{
		Lines:              lines,
		headingAt:          map[int]int{},
		definitionAt:       map[int]int{},
		fenceAt:            map[int]int{},
		setextAt:           map[int]bool{},
		thematicAt:         map[int]bool{},
		definitionsByLabel: map[string]int{},
	}
	slugCounts := map[string]int{}

	for index := 0; index < len(lines); {
		line := lines[index]

		fenceLine := line
		fenceOffset := 0
		quoteDepth := 0
		if content, offset, depth := blockquotePrefix(line); depth > 0 {
			fenceLine = content
			fenceOffset = offset
			quoteDepth = depth
		}

		if open, ok := parseFenceOpen(fenceLine); ok {
			open.startCol += fenceOffset
			fenceIndex := len(doc.Fences)
			fence := Fence{
				StartLine: index,
				EndLine:   len(lines) - 1,
				Marker:    open.marker,
				Length:    open.length,
				Info:      open.info,
				StartCol:  open.startCol,
			}
			doc.fenceAt[index] = fenceIndex

			closed := false
			for next := index + 1; next < len(lines); next++ {
				doc.fenceAt[next] = fenceIndex
				closeLine := lines[next]
				if quoteDepth > 0 {
					content, _, depth := blockquotePrefix(closeLine)
					if depth != quoteDepth {
						continue
					}
					closeLine = content
				}
				if isFenceClose(closeLine, open) {
					fence.EndLine = next
					fence.Closed = true
					closed = true
					index = next + 1
					break
				}
			}

			doc.Fences = append(doc.Fences, fence)
			if !closed {
				index = len(lines)
			}
			continue
		}

		content, offset, _ := containerPrefix(line)

		if definition, ok := parseDefinition(content); ok {
			definition.Line = index
			definition.StartCol += offset
			definition.EndCol += offset
			definitionIndex := len(doc.Definitions)
			doc.Definitions = append(doc.Definitions, definition)
			doc.definitionAt[index] = definitionIndex
			label := NormalizeLabel(definition.Label)
			if _, exists := doc.definitionsByLabel[label]; !exists {
				doc.definitionsByLabel[label] = definitionIndex
			}
			index++
			continue
		}

		if level, ok := setextLevel(content); ok && index > 0 {
			textLine := lines[index-1]
			textContent, textOffset, textInList := containerPrefix(textLine)
			if doc.canBeSetextText(index-1, textContent, textInList) {
				doc.addHeading(Heading{
					Level:    level,
					Text:     strings.TrimSpace(textContent),
					Line:     index - 1,
					StartCol: textOffset + firstNonSpace(textContent),
					EndCol:   textOffset + len(strings.TrimRight(textContent, " \t")),
					Setext:   true,
				}, slugCounts)
				doc.setextAt[index] = true
				index++
				continue
			}
		}
		if atx, ok := parseATXHeading(content); ok {
			doc.addHeading(Heading{
				Level:      atx.level,
				Text:       atx.text,
				Line:       index,
				StartCol:   offset + atx.startCol,
				EndCol:     offset + len(strings.TrimRight(content, " \t")),
				MarkerEnd:  offset + atx.markerEnd,
				SpaceAfter: atx.spaceAfter,
			}, slugCounts)
			index++
			continue
		}

		quoteContent, _, _ := blockquotePrefix(line)
		if isThematicBreak(line) || isThematicBreak(quoteContent) {
			doc.thematicAt[index] = true
		}
		index++
	}

	return doc
}

// addHeading appends a heading and assigns a unique slug.
func (d *Document) addHeading(heading Heading, slugCounts map[string]int) {
	base := Slugify(heading.Text)
	slug := base
	if count := slugCounts[base]; count > 0 {
		slug = base + "-" + strconv.Itoa(count)
	}
	slugCounts[base]++
	heading.Slug = slug

	d.headingAt[heading.Line] = len(d.Headings)
	d.Headings = append(d.Headings, heading)
}

// canBeSetextText reports whether a line can be Setext heading text.
func (d *Document) canBeSetextText(index int, content string, inList bool) bool {
	if index < 0 || index >= len(d.Lines) {
		return false
	}
	if _, ok := d.fenceAt[index]; ok {
		return false
	}
	if _, ok := d.definitionAt[index]; ok {
		return false
	}
	if inList {
		return false
	}

	trimmed := strings.TrimSpace(content)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return false
	}
	if _, ok := setextLevel(content); ok {
		return false
	}
	if strings.HasPrefix(content, "    ") || strings.HasPrefix(content, "\t") {
		return false
	}
	if isThematicBreak(content) {
		return false
	}
	if _, _, ok := ListMarker(content); ok {
		return false
	}
	return true
}

// FenceAt returns the fenced code block containing a line.
func (d *Document) FenceAt(line int) (Fence, bool) {
	index, ok := d.fenceAt[line]
	if !ok {
		return Fence{}, false
	}
	return d.Fences[index], true
}

// HeadingAt returns the heading defined on a line.
func (d *Document) HeadingAt(line int) (Heading, bool) {
	index, ok := d.headingAt[line]
	if !ok {
		return Heading{}, false
	}
	return d.Headings[index], true
}

// DefinitionAt returns the link reference definition on a line.
func (d *Document) DefinitionAt(line int) (Definition, bool) {
	index, ok := d.definitionAt[line]
	if !ok {
		return Definition{}, false
	}
	return d.Definitions[index], true
}

// DefinitionFor finds a definition by normalized label.
func (d *Document) DefinitionFor(label string) (Definition, bool) {
	index, ok := d.definitionsByLabel[NormalizeLabel(label)]
	if !ok {
		return Definition{}, false
	}
	return d.Definitions[index], true
}

// IsSetextUnderline reports whether a line is a Setext underline.
func (d *Document) IsSetextUnderline(line int) bool {
	return d.setextAt[line]
}

// IsThematicBreak reports whether a line is a thematic break.
func (d *Document) IsThematicBreak(line int) bool {
	return d.thematicAt[line]
}

// IsDefinitionLine reports whether a line is a link reference definition.
func (d *Document) IsDefinitionLine(line int) bool {
	_, ok := d.definitionAt[line]
	return ok
}

// HeadingForAnchor finds a heading by anchor slug.
func (d *Document) HeadingForAnchor(anchor string) (Heading, bool) {
	normalized := NormalizeAnchor(anchor)
	for _, heading := range d.Headings {
		if heading.Level <= 6 && heading.Slug == normalized {
			return heading, true
		}
	}
	return Heading{}, false
}

// AnchorExists reports whether a heading slug exists.
func (d *Document) AnchorExists(anchor string) bool {
	_, ok := d.HeadingForAnchor(anchor)
	return ok
}

// NormalizeLabel lowercases and collapses whitespace in a reference label.
func NormalizeLabel(label string) string {
	return strings.ToLower(strings.Join(strings.Fields(label), " "))
}

// NormalizeAnchor decodes and lowercases an anchor target.
func NormalizeAnchor(anchor string) string {
	normalized := strings.TrimSpace(strings.TrimPrefix(anchor, "#"))
	if decoded, err := url.PathUnescape(normalized); err == nil {
		normalized = decoded
	}
	return strings.ToLower(normalized)
}

// Slugify converts heading text into a Markdown anchor slug.
func Slugify(text string) string {
	var builder strings.Builder
	previousSpace := false

	for _, current := range strings.ToLower(text) {
		switch {
		case unicode.IsLetter(current) || unicode.IsDigit(current) || current == '-' || current == '_':
			builder.WriteRune(current)
			previousSpace = false
		case unicode.IsSpace(current) || current == '\t':
			if !previousSpace && builder.Len() > 0 {
				builder.WriteByte('-')
				previousSpace = true
			}
		}
	}

	return strings.Trim(builder.String(), "-")
}

// blockquotePrefix strips blockquote markers and returns the quote depth.
func blockquotePrefix(line string) (content string, offset int, depth int) {
	for {
		cursor := offset
		spaces := 0
		for cursor < len(line) && spaces < 3 && line[cursor] == ' ' {
			cursor++
			spaces++
		}
		if cursor >= len(line) || line[cursor] != '>' {
			break
		}
		cursor++
		if cursor < len(line) && (line[cursor] == ' ' || line[cursor] == '\t') {
			cursor++
		}
		offset = cursor
		depth++
	}
	return line[offset:], offset, depth
}

// containerPrefix strips blockquote and list markers from a line.
func containerPrefix(line string) (content string, offset int, inList bool) {
	remainder, offset, _ := blockquotePrefix(line)

	indent, marker, ok := ListMarker(remainder)
	if !ok {
		return remainder, offset, false
	}

	afterMarker := offset + indent + len(marker)
	for afterMarker < len(line) && (line[afterMarker] == ' ' || line[afterMarker] == '\t') {
		afterMarker++
	}
	return line[afterMarker:], afterMarker, true
}

// ListMarker parses an ordered or unordered list marker.
func ListMarker(line string) (indent int, marker string, ok bool) {
	index := 0
	for index < len(line) && index < 3 && line[index] == ' ' {
		index++
	}
	if index >= len(line) {
		return 0, "", false
	}

	start := index
	if line[index] == '-' || line[index] == '+' || line[index] == '*' {
		index++
		if index >= len(line) || line[index] == ' ' || line[index] == '\t' {
			return start, line[start:index], true
		}
		return 0, "", false
	}

	digits := index
	for index < len(line) && line[index] >= '0' && line[index] <= '9' {
		index++
	}
	if digits == index || index >= len(line) || (line[index] != '.' && line[index] != ')') {
		return 0, "", false
	}
	index++
	if index >= len(line) || line[index] == ' ' || line[index] == '\t' {
		return start, line[start:index], true
	}
	return 0, "", false
}

// parseDefinition parses a link reference definition line.
func parseDefinition(line string) (Definition, bool) {
	match := definitionPattern.FindStringSubmatchIndex(line)
	if match == nil {
		return Definition{}, false
	}

	label := line[match[2]:match[3]]
	rest := ""
	if match[4] >= 0 {
		rest = strings.TrimSpace(line[match[4]:match[5]])
	}

	destination, title, valid := ParseDestinationTitle(rest)
	return Definition{
		Label:       label,
		Destination: destination,
		Title:       title,
		StartCol:    match[2] - 1,
		EndCol:      match[3] + 1,
		Valid:       valid,
	}, true
}

// ParseDestinationTitle splits a link destination from its optional title.
func ParseDestinationTitle(value string) (string, string, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", "", false
	}

	if value[0] == '<' {
		end := findUnescaped(value, '>', 1)
		if end < 0 {
			return "", "", false
		}
		destination := value[1:end]
		remainder := strings.TrimSpace(value[end+1:])
		title, ok := parseTitle(remainder)
		return destination, title, ok
	}

	fields := strings.Fields(value)
	destination := fields[0]
	remainder := strings.TrimSpace(value[len(destination):])
	title, ok := parseTitle(remainder)
	return destination, title, ok
}

// parseTitle validates and extracts a quoted or parenthesized title.
func parseTitle(remainder string) (string, bool) {
	if remainder == "" {
		return "", true
	}
	if len(remainder) < 2 {
		return "", false
	}

	open := remainder[0]
	var close byte
	switch open {
	case '"':
		close = '"'
	case '\'':
		close = '\''
	case '(':
		close = ')'
	default:
		return "", false
	}

	if remainder[len(remainder)-1] != close {
		return "", false
	}
	return strings.TrimSpace(remainder[1 : len(remainder)-1]), true
}

// parseATXHeading parses an ATX heading line.
func parseATXHeading(line string) (atxHeading, bool) {
	index := 0
	for index < len(line) && index < 3 && line[index] == ' ' {
		index++
	}
	if index >= len(line) || line[index] != '#' {
		return atxHeading{}, false
	}

	start := index
	for index < len(line) && line[index] == '#' {
		index++
	}
	level := index - start
	if level == 0 {
		return atxHeading{}, false
	}

	spaceAfter := index >= len(line) || line[index] == ' ' || line[index] == '\t'
	text := ""
	if index < len(line) {
		text = trimClosingHashes(strings.TrimSpace(line[index:]))
	}

	return atxHeading{
		level:      level,
		text:       text,
		startCol:   start,
		markerEnd:  index,
		spaceAfter: spaceAfter,
	}, true
}

// trimClosingHashes removes optional closing hashes from heading text.
func trimClosingHashes(text string) string {
	trimmed := strings.TrimRight(text, " \t")
	end := len(trimmed)
	for end > 0 && trimmed[end-1] == '#' {
		end--
	}
	if end == len(trimmed) {
		return text
	}
	if end == 0 {
		return ""
	}
	if trimmed[end-1] == ' ' || trimmed[end-1] == '\t' {
		return strings.TrimSpace(trimmed[:end])
	}
	return text
}

// parseFenceOpen parses the opening line of a fenced code block.
func parseFenceOpen(line string) (fenceOpen, bool) {
	index := 0
	for index < len(line) && index < 3 && line[index] == ' ' {
		index++
	}
	if index >= len(line) {
		return fenceOpen{}, false
	}

	marker := line[index]
	if marker != '`' && marker != '~' {
		return fenceOpen{}, false
	}

	start := index
	for index < len(line) && line[index] == marker {
		index++
	}
	length := index - start
	if length < 3 {
		return fenceOpen{}, false
	}

	info := strings.TrimSpace(line[index:])
	if marker == '`' && strings.ContainsRune(info, '`') {
		return fenceOpen{}, false
	}

	return fenceOpen{marker: marker, length: length, info: info, startCol: start}, true
}

// isFenceClose reports whether a line closes a fenced code block.
func isFenceClose(line string, open fenceOpen) bool {
	index := 0
	for index < len(line) && index < 3 && line[index] == ' ' {
		index++
	}

	start := index
	for index < len(line) && line[index] == open.marker {
		index++
	}
	if index-start < open.length {
		return false
	}
	return strings.TrimSpace(line[index:]) == ""
}

// isThematicBreak reports whether a line is a horizontal rule.
func isThematicBreak(line string) bool {
	index := 0
	for index < len(line) && index < 3 && line[index] == ' ' {
		index++
	}
	if index >= len(line) {
		return false
	}

	marker := line[index]
	if marker != '*' && marker != '-' && marker != '_' {
		return false
	}

	count := 0
	for ; index < len(line); index++ {
		switch line[index] {
		case marker:
			count++
		case ' ', '\t':
		default:
			return false
		}
	}
	return count >= 3
}

// setextLevel returns the heading level for a Setext underline.
func setextLevel(line string) (int, bool) {
	match := setextPattern.FindStringSubmatch(line)
	if match == nil {
		return 0, false
	}
	if strings.HasPrefix(match[1], "=") {
		return 1, true
	}
	return 2, true
}

// findUnescaped finds the next unescaped occurrence of a byte.
func findUnescaped(value string, target byte, start int) int {
	escaped := false
	for index := start; index < len(value); index++ {
		if escaped {
			escaped = false
			continue
		}
		if value[index] == '\\' {
			escaped = true
			continue
		}
		if value[index] == target {
			return index
		}
	}
	return -1
}

// firstNonSpace returns the index of the first non-space byte.
func firstNonSpace(line string) int {
	for index := 0; index < len(line); index++ {
		if line[index] != ' ' && line[index] != '\t' {
			return index
		}
	}
	return 0
}

// splitLines normalizes line endings and splits source text into lines.
func splitLines(text string) []string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	return strings.Split(text, "\n")
}
