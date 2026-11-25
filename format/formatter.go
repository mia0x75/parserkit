package format

import (
	"fmt"
	"io"
)

type Formatter interface {
	io.Writer
	Format(string, ...any) (int, error)
}

const (
	st0 = iota
	stBOL
	stPERC
	stBOLPERC
)

type indentFormatter struct {
	RestoreWriter
	indent      []byte
	indentLevel int
	state       int
}

// IndentFormatter retains %i/%u indentation across successive Format calls.
func IndentFormatter(w io.Writer, indent string) Formatter {
	return &indentFormatter{RestoreWriter: NewRestoreWriter(w), indent: []byte(indent), state: stBOL}
}

func (f *indentFormatter) format(flat bool, pattern string, args ...any) (int, error) {
	if err := f.Err(); err != nil {
		return 0, err
	}
	buf := make([]byte, 0)
	for i := 0; i < len(pattern); i++ {
		c := pattern[i]
		switch f.state {
		case st0:
			switch c {
			case '\n':
				if flat && f.indentLevel != 0 {
					c = ' '
				}
				buf = append(buf, c)
				f.state = stBOL
			case '%':
				f.state = stPERC
			default:
				buf = append(buf, c)
			}
		case stBOL:
			switch c {
			case '\n':
				if flat && f.indentLevel != 0 {
					c = ' '
				}
				buf = append(buf, c)
			case '%':
				f.state = stBOLPERC
			default:
				if !flat {
					for j := 0; j < f.indentLevel; j++ {
						buf = append(buf, f.indent...)
					}
				}
				buf = append(buf, c)
				f.state = st0
			}
		case stBOLPERC:
			switch c {
			case 'i':
				f.indentLevel++
				f.state = stBOL
			case 'u':
				f.indentLevel--
				f.state = stBOL
			default:
				if !flat {
					for j := 0; j < f.indentLevel; j++ {
						buf = append(buf, f.indent...)
					}
				}
				buf = append(buf, '%', c)
				f.state = st0
			}
		case stPERC:
			switch c {
			case 'i':
				f.indentLevel++
				f.state = st0
			case 'u':
				f.indentLevel--
				f.state = st0
			default:
				buf = append(buf, '%', c)
				f.state = st0
			}
		default:
			panic("unexpected formatter state")
		}
	}
	switch f.state {
	case stPERC, stBOLPERC:
		buf = append(buf, '%')
	}
	return fmt.Fprintf(f, string(buf), args...)
}

func (f *indentFormatter) Format(pattern string, args ...any) (int, error) {
	return f.format(false, pattern, args...)
}

type flatFormatter indentFormatter

// FlatFormatter replaces indented newlines with spaces, retaining top-level ones.
func FlatFormatter(w io.Writer) Formatter {
	return (*flatFormatter)(IndentFormatter(w, "").(*indentFormatter))
}

func (f *flatFormatter) Format(pattern string, args ...any) (int, error) {
	return (*indentFormatter)(f).format(true, pattern, args...)
}
