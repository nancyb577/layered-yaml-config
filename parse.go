// Package yamlconf parses a restricted, well-defined subset of YAML:
// nested maps, block-style lists of scalars, and scalar values (string,
// int, float, bool, null). It exists so that reading simple config files
// doesn't require pulling in a full YAML implementation as a dependency.
//
// Not supported (yet): lists of maps, flow style ({}/[]), anchors and
// aliases, multi-line strings, and multi-document files.
package yamlconf

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

type rawLine struct {
	num      int
	indent   int
	key      string
	value    string
	hasValue bool
	listItem bool
}

// Parse reads r and returns the top-level map it describes.
func Parse(r io.Reader) (map[string]interface{}, error) {
	lines, err := readLines(r)
	if err != nil {
		return nil, err
	}
	if len(lines) == 0 {
		return map[string]interface{}{}, nil
	}
	i := 0
	return parseBlock(lines, &i, lines[0].indent)
}

// ParseFile opens path and parses it with Parse.
func ParseFile(path string) (map[string]interface{}, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Parse(f)
}

func readLines(r io.Reader) ([]rawLine, error) {
	var lines []rawLine
	scanner := bufio.NewScanner(r)
	num := 0
	for scanner.Scan() {
		num++
		trimmed := strings.TrimRight(scanner.Text(), " \t")
		if strings.TrimSpace(trimmed) == "" {
			continue
		}
		content := strings.TrimLeft(trimmed, " ")
		if strings.HasPrefix(content, "#") {
			continue
		}
		if strings.HasPrefix(content, "\t") {
			return nil, fmt.Errorf("line %d: tabs are not allowed for indentation", num)
		}
		indent := len(trimmed) - len(content)
		content = strings.TrimRight(stripComment(content), " \t")
		if content == "-" || strings.HasPrefix(content, "- ") {
			value := strings.TrimSpace(content[1:])
			lines = append(lines, rawLine{
				num:      num,
				indent:   indent,
				value:    value,
				hasValue: value != "",
				listItem: true,
			})
			continue
		}
		idx := strings.Index(content, ":")
		if idx == -1 {
			return nil, fmt.Errorf("line %d: expected \"key: value\", got %q", num, content)
		}
		key := strings.TrimSpace(content[:idx])
		if key == "" {
			return nil, fmt.Errorf("line %d: empty key", num)
		}
		value := strings.TrimSpace(content[idx+1:])
		lines = append(lines, rawLine{
			num:      num,
			indent:   indent,
			key:      key,
			value:    value,
			hasValue: value != "",
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return lines, nil
}

// parseBlock consumes every consecutive line at exactly the given indent,
// treating deeper indents as nested maps under the preceding key.
func parseBlock(lines []rawLine, i *int, indent int) (map[string]interface{}, error) {
	result := map[string]interface{}{}
	for *i < len(lines) {
		ln := lines[*i]
		if ln.indent < indent {
			break
		}
		if ln.indent > indent {
			return nil, fmt.Errorf("line %d: unexpected indentation", ln.num)
		}
		if ln.listItem {
			return nil, fmt.Errorf("line %d: list item not under a key", ln.num)
		}
		*i++
		if ln.hasValue {
			result[ln.key] = parseScalar(ln.value)
			continue
		}
		if *i >= len(lines) || lines[*i].indent <= indent {
			result[ln.key] = map[string]interface{}{}
			continue
		}
		if lines[*i].listItem {
			list, err := parseListBlock(lines, i, lines[*i].indent)
			if err != nil {
				return nil, err
			}
			result[ln.key] = list
			continue
		}
		child, err := parseBlock(lines, i, lines[*i].indent)
		if err != nil {
			return nil, err
		}
		result[ln.key] = child
	}
	return result, nil
}

// parseListBlock consumes every consecutive list-item line ("- value") at
// exactly the given indent and returns their scalar values in order.
func parseListBlock(lines []rawLine, i *int, indent int) ([]interface{}, error) {
	var result []interface{}
	for *i < len(lines) {
		ln := lines[*i]
		if ln.indent < indent {
			break
		}
		if ln.indent > indent {
			return nil, fmt.Errorf("line %d: unexpected indentation", ln.num)
		}
		if !ln.listItem {
			return nil, fmt.Errorf("line %d: expected a list item (\"- value\") or dedent, got %q", ln.num, ln.key)
		}
		*i++
		result = append(result, parseScalar(ln.value))
	}
	return result, nil
}

// stripComment removes a trailing "# ..." comment from a line, honoring
// quotes so a '#' inside a quoted scalar isn't mistaken for one. As in
// YAML, a '#' only starts a comment at the start of the content or when
// preceded by whitespace, so it doesn't break unquoted values like URLs
// that happen to contain a '#'.
func stripComment(s string) string {
	inQuote := byte(0)
	for i := 0; i < len(s); i++ {
		c := s[i]
		if inQuote != 0 {
			if inQuote == '"' && c == '\\' && i+1 < len(s) {
				i++
				continue
			}
			if c == inQuote {
				inQuote = 0
			}
			continue
		}
		switch c {
		case '"', '\'':
			inQuote = c
		case '#':
			if i == 0 || s[i-1] == ' ' || s[i-1] == '\t' {
				return s[:i]
			}
		}
	}
	return s
}

func parseScalar(s string) interface{} {
	if len(s) >= 2 {
		switch {
		case s[0] == '"' && s[len(s)-1] == '"':
			return unescapeDouble(s[1 : len(s)-1])
		case s[0] == '\'' && s[len(s)-1] == '\'':
			// Single-quoted YAML strings have no backslash escapes; a
			// doubled quote is the only way to represent a literal one.
			return strings.ReplaceAll(s[1:len(s)-1], "''", "'")
		}
	}
	switch s {
	case "", "null", "~":
		return nil
	case "true":
		return true
	case "false":
		return false
	}
	if n, err := strconv.Atoi(s); err == nil {
		return n
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}
	return s
}

// unescapeDouble processes the backslash escape sequences allowed inside a
// double-quoted YAML scalar (the content passed in has already had its
// surrounding quotes stripped). An unrecognized escape is passed through
// with the backslash dropped, rather than treated as an error, since the
// caller has no way to report one back through parseScalar.
func unescapeDouble(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c != '\\' || i+1 >= len(s) {
			b.WriteByte(c)
			continue
		}
		next := s[i+1]
		switch next {
		case '\\', '"', '/':
			b.WriteByte(next)
			i++
		case 'n':
			b.WriteByte('\n')
			i++
		case 't':
			b.WriteByte('\t')
			i++
		case 'r':
			b.WriteByte('\r')
			i++
		case 'a':
			b.WriteByte('\a')
			i++
		case 'b':
			b.WriteByte('\b')
			i++
		case 'f':
			b.WriteByte('\f')
			i++
		case 'v':
			b.WriteByte('\v')
			i++
		case '0':
			b.WriteByte(0)
			i++
		case 'x':
			if i+4 <= len(s) {
				if v, err := strconv.ParseUint(s[i+2:i+4], 16, 8); err == nil {
					b.WriteByte(byte(v))
					i += 3
					continue
				}
			}
			b.WriteByte(next)
			i++
		case 'u':
			if i+6 <= len(s) {
				if v, err := strconv.ParseUint(s[i+2:i+6], 16, 32); err == nil {
					b.WriteRune(rune(v))
					i += 5
					continue
				}
			}
			b.WriteByte(next)
			i++
		default:
			b.WriteByte(next)
			i++
		}
	}
	return b.String()
}
