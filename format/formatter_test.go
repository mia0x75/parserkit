package format

import (
	"errors"
	"io"
	"strings"
	"testing"
)

func TestFormatter(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		flat    bool
		formats []string
		want    string
	}{
		{name: "indent", formats: []string{"abc%d%%e%i\nx\ny\n%uz\n"}, want: "abc3%e\n\tx\n\ty\nz\n"},
		{name: "flat", flat: true, formats: []string{"abc%d%%e%i\nx\ny\n%uz\n%i\n"}, want: "abc3%e x y z\n "},
		{name: "separate calls", formats: []string{"abc%d%%e%i\nx\n", "y\n%uz\n"}, want: "abc3%e\n\tx\n\ty\nz\n"},
		{name: "nested", formats: []string{"%i%i%d\n%ux\n%uy\n"}, want: "\t\t3\n\tx\ny\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out strings.Builder
			f := IndentFormatter(&out, "\t")
			if tt.flat {
				f = FlatFormatter(&out)
			}
			for _, pattern := range tt.formats {
				var args []any
				if strings.Contains(pattern, "%d") {
					args = []any{3}
				}
				before := out.Len()
				n, err := f.Format(pattern, args...)
				if err != nil || n != out.Len()-before {
					t.Fatalf("Format() = %d, %v", n, err)
				}
			}
			if out.String() != tt.want {
				t.Fatalf("output = %q, want %q", out.String(), tt.want)
			}
		})
	}
}

func TestFormatterErrors(t *testing.T) {
	t.Parallel()
	want := errors.New("mock output failure")
	w := &failingWriter{err: want}
	f := IndentFormatter(w, "\t")
	if _, err := f.Format("%i%s\n", "first"); !errors.Is(err, want) {
		t.Fatalf("error = %v, want %v", err, want)
	}
	if _, err := f.Write([]byte("later")); !errors.Is(err, want) {
		t.Fatalf("Write error = %v", err)
	}
	if _, err := f.Format("later"); !errors.Is(err, want) || w.calls != 1 {
		t.Fatalf("first error lost: %v, writer calls %d", err, w.calls)
	}
	if _, err := FlatFormatter(&failingWriter{short: true}).Format("short"); !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("short write error = %v", err)
	}
	if _, err := IndentFormatter(nil, "\t").Format("nil"); !errors.Is(err, ErrNilWriter) {
		t.Fatalf("nil writer error = %v", err)
	}
}
