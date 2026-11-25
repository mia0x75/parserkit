// Package loc provides byte ranges and on-demand Unicode source positions.
package loc

import (
	"fmt"
	"unicode/utf8"
)

// ByteOffset measures bytes from the start of the original input.
type ByteOffset int

// InvalidOffset marks an unknown byte position.
const InvalidOffset ByteOffset = -1

// Span is a half-open byte range [Start, End). Its zero value is valid and empty.
// Use InvalidSpan for unknown ranges.
type Span struct {
	Start ByteOffset
	End   ByteOffset
}

// InvalidSpan returns an unknown range, not the valid empty range [0, 0).
func InvalidSpan() Span {
	return Span{Start: InvalidOffset, End: InvalidOffset}
}

// Valid reports whether both offsets are nonnegative and ordered.
func (s Span) Valid() bool {
	return s.Start >= 0 && s.End >= s.Start
}

// Empty reports whether s is a valid range containing no bytes.
func (s Span) Empty() bool {
	return s.Valid() && s.Start == s.End
}

// Within reports whether s is valid and fits an input of length bytes.
func (s Span) Within(length int) bool {
	return s.Valid() && s.End <= ByteOffset(length)
}

// Cover returns the smallest range containing the valid inputs, ignoring invalid
// ranges. If neither input is valid, it returns InvalidSpan.
func Cover(a, b Span) Span {
	if !a.Valid() {
		if !b.Valid() {
			return InvalidSpan()
		}
		return b
	}
	if !b.Valid() {
		return a
	}
	return Span{Start: min(a.Start, b.Start), End: max(a.End, b.End)}
}

// Slice returns the original bytes in s, or an error for an invalid or
// out-of-bounds range. It does not require UTF-8 code point boundaries.
func (s Span) Slice(input string) (string, error) {
	if !s.Within(len(input)) {
		return "", fmt.Errorf("loc: span [%d, %d) outside input length %d", s.Start, s.End, len(input))
	}
	return input[s.Start:s.End], nil
}

// Position computes one-based line and Unicode code point column numbers in O(n).
// LF and CRLF each start a new line; a lone CR occupies one column.
// An offset inside a UTF-8 code point maps to that code point's column.
// Invalid UTF-8 bytes each occupy one column. Unknown or out-of-bounds offsets
// return an error and zero line and column numbers.
func Position(input string, offset ByteOffset) (line, column int, err error) {
	if offset < 0 || offset > ByteOffset(len(input)) {
		return 0, 0, fmt.Errorf("loc: offset %d outside input length %d", offset, len(input))
	}
	line, column = 1, 1
	for i := 0; i < int(offset); {
		if input[i] == '\n' {
			line++
			column = 1
			i++
			continue
		}
		isCRLF := input[i] == '\r' && i+1 < len(input) && input[i+1] == '\n'
		if isCRLF {
			i++
			continue
		}
		_, size := utf8.DecodeRuneInString(input[i:])
		if i+size > int(offset) {
			break
		}
		column++
		i += size
	}
	return line, column, nil
}
