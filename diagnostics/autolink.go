package diagnostics

import (
	"strings"

	"github.com/atharva-0123-cs/read_lsp/lsp"
)

type Autolink struct {
	Start       int
	End         int
	Text        string
	Destination string
	Email       bool
}

type angleSpan struct {
	Start   int
	End     int
	Content string
}

func AutolinkAt(line string, character int) (Autolink, bool) {
	character = clampCharacter(character, len(line))
	for _, current := range angleSpans(line) {
		if character < current.Start || character > current.End {
			continue
		}
		if !isAutolink(current.Content) {
			return Autolink{}, false
		}
		return newAutolink(current), true
	}
	return Autolink{}, false
}

func HTMLTagAt(line string, character int) (string, bool) {
	character = clampCharacter(character, len(line))
	for _, current := range angleSpans(line) {
		if character < current.Start || character > current.End {
			continue
		}
		if isAutolink(current.Content) || !isHTMLTag(current.Content) {
			return "", false
		}
		return line[current.Start : current.End+1], true
	}
	return "", false
}

func newAutolink(current angleSpan) Autolink {
	destination := current.Content
	email := isEmailAutolink(current.Content)
	if email && !strings.HasPrefix(strings.ToLower(destination), "mailto:") {
		destination = "mailto:" + destination
	}
	return Autolink{
		Start:       current.Start,
		End:         current.End + 1,
		Text:        current.Content,
		Destination: destination,
		Email:       email,
	}
}

func angleSpans(line string) []angleSpan {
	result := []angleSpan{}
	for index := 0; index < len(line); {
		if line[index] == '\\' {
			index += 2
			continue
		}
		if line[index] != '<' {
			index++
			continue
		}

		end := findUnescaped(line, '>', index+1)
		if end < 0 {
			index++
			continue
		}
		result = append(result, angleSpan{
			Start:   index,
			End:     end,
			Content: line[index+1 : end],
		})
		index = end + 1
	}
	return result
}

func maskHTMLAndAutolinks(line string, lineNumber, offset int) (string, []lsp.Diagnostic) {
	bytes := []byte(line)
	result := []lsp.Diagnostic{}

	for index := 0; index < len(bytes); {
		if bytes[index] == '\\' {
			index += 2
			continue
		}
		if bytes[index] != '<' {
			index++
			continue
		}

		end := findUnescaped(string(bytes), '>', index+1)
		if end < 0 {
			if looksLikeAutolinkStart(string(bytes[index+1:])) {
				result = append(result, newDiagnostic(
					lineNumber, offset+index, offset+len(bytes), SeverityWarning,
					CodeAutolink,
					"Unclosed autolink; add a closing '>'",
				))
				for cursor := index; cursor < len(bytes); cursor++ {
					bytes[cursor] = ' '
				}
			}
			break
		}

		content := string(bytes[index+1 : end])
		if isAutolink(content) || isHTMLTag(content) {
			for cursor := index; cursor <= end; cursor++ {
				bytes[cursor] = ' '
			}
			index = end + 1
			continue
		}
		index = end + 1
	}

	return string(bytes), result
}

func looksLikeAutolinkStart(value string) bool {
	if index := strings.IndexAny(value, " \t"); index >= 0 {
		value = value[:index]
	}
	return value != "" && isAutolink(value)
}

func isAutolink(value string) bool {
	if value == "" || strings.ContainsAny(value, " \t<>") {
		return false
	}
	lower := strings.ToLower(value)
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") || strings.HasPrefix(lower, "mailto:") {
		return true
	}
	return isEmailAutolink(value)
}

func isEmailAutolink(value string) bool {
	if strings.ContainsAny(value, " \t<>\n") {
		return false
	}
	at := strings.IndexByte(value, '@')
	if at <= 0 || at == len(value)-1 {
		return false
	}
	if strings.Contains(value[at+1:], "@") {
		return false
	}
	domain := value[at+1:]
	return strings.Contains(domain, ".") && !strings.HasPrefix(domain, ".") && !strings.HasSuffix(domain, ".")
}

func isHTMLTag(value string) bool {
	if value == "" {
		return false
	}
	switch value[0] {
	case '!', '/':
		return true
	}
	return value[0] >= 'A' && value[0] <= 'Z' || value[0] >= 'a' && value[0] <= 'z'
}

func clampCharacter(value, length int) int {
	if value < 0 {
		return 0
	}
	if value > length {
		return length
	}
	return value
}
