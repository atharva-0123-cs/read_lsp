package diagnostics

import (
	"strconv"
	"strings"

	"github.com/atharva-0123-cs/read_lsp/lsp"
)

// referenceUse records a reference label used in a line.
type referenceUse struct {
	Label string
	Line  int
	Start int
	End   int
}

// span is a half-open byte range within a line.
type span struct {
	Start int
	End   int
}

// inlineDiagnostics checks code, links, HTML, and emphasis in a line.
func inlineDiagnostics(document *Document, lineNumber int) ([]lsp.Diagnostic, []referenceUse) {
	line := document.Lines[lineNumber]
	if line == "" {
		return nil, nil
	}

	content, offset, _ := containerPrefix(line)
	if strings.TrimSpace(content) == "" {
		return nil, nil
	}
	if strings.HasPrefix(content, "    ") || strings.HasPrefix(content, "\t") {
		return nil, nil
	}

	maskedCode, result := maskCodeSpans(content, lineNumber, offset)
	maskedHTML, htmlDiagnostics := maskHTMLAndAutolinks(maskedCode, lineNumber, offset)
	result = append(result, htmlDiagnostics...)
	linkDiagnostics, references, destinations := scanLinks(document, maskedHTML, lineNumber, offset)
	result = append(result, linkDiagnostics...)
	result = append(result, emphasisDiagnostics(maskRanges(maskedHTML, destinations), lineNumber, offset)...)
	return result, references
}

// maskCodeSpans blanks inline code and reports unclosed spans.
func maskCodeSpans(line string, lineNumber, offset int) (string, []lsp.Diagnostic) {
	bytes := []byte(line)
	result := []lsp.Diagnostic{}

	for index := 0; index < len(bytes); {
		if bytes[index] == '\\' {
			index += 2
			continue
		}
		if bytes[index] != '`' {
			index++
			continue
		}

		run := runLength(bytes, index, '`')
		closeAt := findCodeSpanClose(bytes, index+run, run)
		if closeAt < 0 {
			result = append(result, newDiagnostic(
				lineNumber, offset+index, offset+index+run, SeverityWarning,
				CodeInlineCode,
				"Unclosed inline code span; add a matching run of "+strconv.Itoa(run)+" backtick(s)",
			))
			for cursor := index; cursor < len(bytes); cursor++ {
				bytes[cursor] = ' '
			}
			break
		}

		for cursor := index; cursor < closeAt+run; cursor++ {
			bytes[cursor] = ' '
		}
		index = closeAt + run
	}

	return string(bytes), result
}

// findCodeSpanClose finds a closing backtick run of the same length.
func findCodeSpanClose(line []byte, start, length int) int {
	for index := start; index < len(line); {
		if line[index] == '\\' {
			index += 2
			continue
		}
		if line[index] != '`' {
			index++
			continue
		}

		run := runLength(line, index, '`')
		if run == length {
			return index
		}
		index += run
	}
	return -1
}

// runLength counts consecutive occurrences of a byte.
func runLength(line []byte, start int, target byte) int {
	length := 0
	for start+length < len(line) && line[start+length] == target {
		length++
	}
	return length
}

// scanLinks validates links and collects reference labels.
func scanLinks(document *Document, line string, lineNumber, offset int) ([]lsp.Diagnostic, []referenceUse, []span) {
	result := []lsp.Diagnostic{}
	references := []referenceUse{}
	destinations := []span{}

	for index := 0; index < len(line); {
		if line[index] == '\\' {
			index += 2
			continue
		}

		bracketStart := -1
		if line[index] == '!' && index+1 < len(line) && line[index+1] == '[' {
			bracketStart = index + 1
		} else if line[index] == '[' {
			bracketStart = index
		} else {
			index++
			continue
		}

		closeBracket := findMatchingBracket(line, bracketStart)
		if closeBracket < 0 {
			index = bracketStart + 1
			continue
		}

		text := line[bracketStart+1 : closeBracket]
		next := closeBracket + 1

		switch {
		case next < len(line) && line[next] == '(':
			closeParen := findMatchingParen(line, next)
			if closeParen < 0 {
				result = append(result, newDiagnostic(
					lineNumber, offset+closeBracket, offset+next+1, SeverityError,
					CodeUnclosedLink,
					"Unclosed link target; add a closing ')'",
				))
				index = len(line)
				continue
			}

			content := line[next+1 : closeParen]
			destination, _, valid := "", "", true
			if strings.TrimSpace(content) != "" {
				destination, _, valid = ParseDestinationTitle(content)
			}
			if !valid {
				result = append(result, newDiagnostic(
					lineNumber, offset+next+1, offset+closeParen, SeverityWarning,
					CodeInvalidLink,
					"Invalid link destination or title",
				))
			}

			destinations = append(destinations, span{Start: next + 1, End: closeParen})
			if strings.HasPrefix(destination, "#") && len(destination) > 1 && !document.AnchorExists(destination) {
				result = append(result, newDiagnostic(
					lineNumber, offset+next+1, offset+closeParen, SeverityInfo,
					CodeMissingAnchor,
					"No heading matches the anchor '"+destination+"'",
				))
			}

			index = closeParen + 1

		case next < len(line) && line[next] == '[':
			var handled bool
			index, handled = scanReference(document, line, lineNumber, offset, text, next, &result, &references, &destinations)
			if !handled {
				index = closeBracket + 1
			}

		case spacedReferenceBracket(line, next) >= 0:
			var handled bool
			index, handled = scanReference(document, line, lineNumber, offset, text, spacedReferenceBracket(line, next), &result, &references, &destinations)
			if !handled {
				index = closeBracket + 1
			}

		default:
			label := strings.TrimSpace(text)
			if label != "" {
				if _, defined := document.DefinitionFor(label); defined {
					references = append(references, referenceUse{
						Label: label,
						Line:  lineNumber,
						Start: offset + bracketStart,
						End:   offset + closeBracket + 1,
					})
				}
			}
			index = closeBracket + 1
		}
	}

	return result, references, destinations
}

// scanReference validates a reference label and records its use.
func scanReference(
	document *Document,
	line string,
	lineNumber int,
	offset int,
	text string,
	bracket int,
	result *[]lsp.Diagnostic,
	references *[]referenceUse,
	destinations *[]span,
) (int, bool) {
	closeReference := findClosingReference(line, bracket)
	if closeReference < 0 {
		*result = append(*result, newDiagnostic(
			lineNumber, offset+bracket, offset+len(line), SeverityWarning,
			CodeUnclosedLink,
			"Unclosed reference label; add a closing ']'",
		))
		return len(line), true
	}

	label := strings.TrimSpace(line[bracket+1 : closeReference])
	if label == "" {
		label = strings.TrimSpace(text)
	}
	if label != "" {
		*references = append(*references, referenceUse{
			Label: label,
			Line:  lineNumber,
			Start: offset + bracket + 1,
			End:   offset + closeReference,
		})
	}
	*destinations = append(*destinations, span{Start: bracket, End: closeReference + 1})
	return closeReference + 1, true
}

// spacedReferenceBracket finds a reference bracket after whitespace.
func spacedReferenceBracket(line string, next int) int {
	cursor := next
	for cursor < len(line) && (line[cursor] == ' ' || line[cursor] == '\t') {
		cursor++
	}
	if cursor > next && cursor < len(line) && line[cursor] == '[' {
		return cursor
	}
	return -1
}

// findMatchingBracket finds the closing bracket for a link.
func findMatchingBracket(line string, open int) int {
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

// findMatchingParen finds the closing parenthesis for a link target.
func findMatchingParen(line string, open int) int {
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

// findClosingReference finds the closing bracket of a reference label.
func findClosingReference(line string, open int) int {
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

// maskRanges blanks selected spans in a line.
func maskRanges(line string, spans []span) string {
	bytes := []byte(line)
	for _, current := range spans {
		for index := current.Start; index < current.End && index < len(bytes); index++ {
			if index >= 0 {
				bytes[index] = ' '
			}
		}
	}
	return string(bytes)
}

// emphasisDiagnostics reports unpaired emphasis delimiters.
func emphasisDiagnostics(line string, lineNumber, offset int) []lsp.Diagnostic {
	asterisks := []span{}
	underscores := []span{}

	for index := 0; index < len(line); {
		if line[index] == '\\' {
			index += 2
			continue
		}
		if line[index] != '*' && line[index] != '_' {
			index++
			continue
		}

		marker := line[index]
		start := index
		for index < len(line) && line[index] == marker {
			index++
		}
		current := span{Start: start, End: index}

		if marker == '_' && intraWord(line, start, index) {
			continue
		}
		if surroundedBySpace(line, start, index) {
			continue
		}
		if marker == '*' {
			asterisks = append(asterisks, current)
		} else {
			underscores = append(underscores, current)
		}
	}

	result := []lsp.Diagnostic{}
	if total, ok := oddDelimiter(asterisks); ok {
		result = append(result, newDiagnostic(
			lineNumber, offset+total.Start, offset+total.End, SeverityWarning,
			CodeEmphasis,
			"Unclosed emphasis delimiter '*'; add a matching '*'",
		))
	}
	if total, ok := oddDelimiter(underscores); ok {
		result = append(result, newDiagnostic(
			lineNumber, offset+total.Start, offset+total.End, SeverityWarning,
			CodeEmphasis,
			"Unclosed emphasis delimiter '_'; add a matching '_'",
		))
	}
	return result
}

// oddDelimiter returns the first run when delimiter totals are odd.
func oddDelimiter(runs []span) (span, bool) {
	if len(runs) == 0 {
		return span{}, false
	}

	total := 0
	for _, current := range runs {
		total += current.End - current.Start
	}
	if total%2 == 0 {
		return span{}, false
	}
	return runs[0], true
}

// intraWord reports whether a marker is surrounded by word bytes.
func intraWord(line string, start, end int) bool {
	if start == 0 || end >= len(line) {
		return false
	}
	return isWordByte(line[start-1]) && isWordByte(line[end])
}

// surroundedBySpace reports whether a marker is surrounded by whitespace.
func surroundedBySpace(line string, start, end int) bool {
	beforeSpace := start == 0 || line[start-1] == ' ' || line[start-1] == '\t'
	afterSpace := end >= len(line) || line[end] == ' ' || line[end] == '\t'
	return beforeSpace && afterSpace
}

// isWordByte reports whether a byte can be part of a word.
func isWordByte(value byte) bool {
	return value >= '0' && value <= '9' ||
		value >= 'A' && value <= 'Z' ||
		value >= 'a' && value <= 'z' ||
		value == '_'
}
