package services

import (
	"encoding/json"
	"fmt"
	"strings"
)

// pythonReprToJSON converts a Python repr string to valid JSON.
// Handles: single quotes -> double quotes, None -> null, True -> true, False -> false.
// Uses a state-machine approach to correctly handle apostrophes inside string values
// and already-escaped double quotes.
func pythonReprToJSON(input string) (string, error) {
	if input == "" {
		return "", fmt.Errorf("empty input")
	}

	input = strings.TrimSpace(input)
	if input == "" {
		return "", fmt.Errorf("empty input after trimming")
	}

	var out strings.Builder
	out.Grow(len(input))

	i := 0
	n := len(input)

	for i < n {
		ch := input[i]

		// Check for Python keywords outside of strings
		if ch == 'N' && i+4 <= n && input[i:i+4] == "None" && !isIdentChar(safeCharAt(input, i+4)) && !isIdentChar(safeCharAt(input, i-1)) {
			out.WriteString("null")
			i += 4
			continue
		}
		if ch == 'T' && i+4 <= n && input[i:i+4] == "True" && !isIdentChar(safeCharAt(input, i+4)) && !isIdentChar(safeCharAt(input, i-1)) {
			out.WriteString("true")
			i += 4
			continue
		}
		if ch == 'F' && i+5 <= n && input[i:i+5] == "False" && !isIdentChar(safeCharAt(input, i+5)) && !isIdentChar(safeCharAt(input, i-1)) {
			out.WriteString("false")
			i += 5
			continue
		}

		// Single-quoted string -> convert to double-quoted JSON string
		if ch == '\'' {
			jsonStr, advance, err := convertPythonString(input, i)
			if err != nil {
				return "", fmt.Errorf("error at position %d: %w", i, err)
			}
			out.WriteString(jsonStr)
			i += advance
			continue
		}

		// Everything else (numbers, brackets, colons, commas, whitespace) passes through
		out.WriteByte(ch)
		i++
	}

	result := out.String()

	// Validate the output is valid JSON
	if !json.Valid([]byte(result)) {
		return "", fmt.Errorf("conversion produced invalid JSON")
	}

	return result, nil
}

// convertPythonString converts a single-quoted Python string starting at position pos
// to a double-quoted JSON string. Returns the JSON string and the number of characters consumed.
func convertPythonString(input string, pos int) (string, int, error) {
	if pos >= len(input) || input[pos] != '\'' {
		return "", 0, fmt.Errorf("expected single quote at position %d", pos)
	}

	var out strings.Builder
	out.WriteByte('"') // opening double quote

	i := pos + 1 // skip opening '
	n := len(input)

	for i < n {
		ch := input[i]

		if ch == '\\' && i+1 < n {
			next := input[i+1]
			if next == '"' {
				// Already-escaped double quote: \" -> \"
				out.WriteString(`\"`)
				i += 2
				continue
			}
			if next == '\'' {
				// Escaped single quote in Python: \' -> just '
				out.WriteByte('\'')
				i += 2
				continue
			}
			// Other escapes pass through
			out.WriteByte('\\')
			out.WriteByte(next)
			i += 2
			continue
		}

		if ch == '"' {
			// Unescaped double quote inside string - must escape for JSON
			out.WriteString(`\"`)
			i++
			continue
		}

		if ch == '\'' {
			// Potential end of string or apostrophe in a word
			// Heuristic: if next char is a letter/digit, it's an apostrophe (e.g., E'S, O'Brien)
			if i+1 < n && isWordChar(input[i+1]) {
				out.WriteByte('\'')
				i++
				continue
			}
			// It's the closing quote
			out.WriteByte('"') // closing double quote
			return out.String(), i - pos + 1, nil
		}

		out.WriteByte(ch)
		i++
	}

	return "", 0, fmt.Errorf("unterminated string starting at position %d", pos)
}

// isIdentChar returns true if ch is a letter, digit, or underscore (identifier character)
func isIdentChar(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_'
}

// isWordChar returns true if ch is a letter or digit (for apostrophe detection)
func isWordChar(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9')
}

// safeCharAt returns the character at index i, or 0 if out of bounds
func safeCharAt(s string, i int) byte {
	if i < 0 || i >= len(s) {
		return 0
	}
	return s[i]
}

