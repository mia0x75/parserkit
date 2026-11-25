package format

import "bytes"

// OutputFormat preserves the legacy literal escaping used by AST Format methods.
func OutputFormat(s string) string {
	var buf bytes.Buffer
	for _, c := range s {
		switch c {
		case '\x00':
			buf.WriteString(`\0`)
		case '\'':
			buf.WriteString(`''`)
		case '\n':
			buf.WriteString(`\n`)
		case '\r':
			buf.WriteString(`\r`)
		default:
			buf.WriteRune(c)
		}
	}
	return buf.String()
}
