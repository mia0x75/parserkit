package loc_test

import (
	"testing"

	"github.com/mia0x75/parserkit/loc"
)

func TestInvalidSpan(t *testing.T) {
	t.Parallel()
	s := loc.InvalidSpan()
	if s.Start != loc.InvalidOffset || s.End != loc.InvalidOffset {
		t.Fatalf("InvalidSpan() = %+v, want both offsets -1", s)
	}
	if s.Valid() || s.Empty() || s.Within(0) {
		t.Fatal("unknown span must not be valid, empty, or within input")
	}
}

func TestSpanValidAndEmpty(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		span      loc.Span
		wantValid bool
		wantEmpty bool
	}{
		{name: "zero value", wantValid: true, wantEmpty: true},
		{name: "empty at offset", span: loc.Span{Start: 3, End: 3}, wantValid: true, wantEmpty: true},
		{name: "nonempty", span: loc.Span{Start: 2, End: 5}, wantValid: true},
		{name: "unknown", span: loc.InvalidSpan()},
		{name: "negative start", span: loc.Span{Start: -1, End: 2}},
		{name: "negative end", span: loc.Span{Start: 0, End: -1}},
		{name: "reversed", span: loc.Span{Start: 3, End: 2}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.span.Valid(); got != tt.wantValid {
				t.Errorf("%+v.Valid() = %t, want %t", tt.span, got, tt.wantValid)
			}
			if got := tt.span.Empty(); got != tt.wantEmpty {
				t.Errorf("%+v.Empty() = %t, want %t", tt.span, got, tt.wantEmpty)
			}
		})
	}
}

func TestSpanWithin(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		span   loc.Span
		length int
		want   bool
	}{
		{name: "empty input", want: true},
		{name: "negative length", length: -1},
		{name: "full input", span: loc.Span{Start: 0, End: 5}, length: 5, want: true},
		{name: "inside input", span: loc.Span{Start: 1, End: 3}, length: 5, want: true},
		{name: "empty at eof", span: loc.Span{Start: 5, End: 5}, length: 5, want: true},
		{name: "end past eof", span: loc.Span{Start: 1, End: 6}, length: 5},
		{name: "empty past eof", span: loc.Span{Start: 6, End: 6}, length: 5},
		{name: "unknown", span: loc.InvalidSpan(), length: 5},
		{name: "reversed", span: loc.Span{Start: 3, End: 2}, length: 5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.span.Within(tt.length); got != tt.want {
				t.Errorf("%+v.Within(%d) = %t, want %t", tt.span, tt.length, got, tt.want)
			}
		})
	}
}

func TestCover(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		a    loc.Span
		b    loc.Span
		want loc.Span
	}{
		{
			name: "both unknown",
			a:    loc.InvalidSpan(),
			b:    loc.InvalidSpan(),
			want: loc.InvalidSpan(),
		},
		{
			name: "invalid ranges normalize to unknown",
			a:    loc.Span{Start: 4, End: 2},
			b:    loc.Span{Start: -2, End: -2},
			want: loc.InvalidSpan(),
		},
		{
			name: "unknown does not include zero",
			a:    loc.InvalidSpan(),
			b:    loc.Span{Start: 5, End: 8},
			want: loc.Span{Start: 5, End: 8},
		},
		{
			name: "reversed range ignored",
			a:    loc.Span{Start: 6, End: 2},
			b:    loc.Span{Start: 5, End: 8},
			want: loc.Span{Start: 5, End: 8},
		},
		{
			name: "disjoint ranges",
			a:    loc.Span{Start: 2, End: 4},
			b:    loc.Span{Start: 6, End: 9},
			want: loc.Span{Start: 2, End: 9},
		},
		{
			name: "nested ranges",
			a:    loc.Span{Start: 2, End: 9},
			b:    loc.Span{Start: 4, End: 6},
			want: loc.Span{Start: 2, End: 9},
		},
		{
			name: "empty ranges",
			a:    loc.Span{},
			b:    loc.Span{Start: 3, End: 3},
			want: loc.Span{Start: 0, End: 3},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := loc.Cover(tt.a, tt.b); got != tt.want {
				t.Errorf("Cover(%+v, %+v) = %+v, want %+v", tt.a, tt.b, got, tt.want)
			}
			if got := loc.Cover(tt.b, tt.a); got != tt.want {
				t.Errorf("Cover(%+v, %+v) = %+v, want %+v", tt.b, tt.a, got, tt.want)
			}
		})
	}
}

func TestSpanSlice(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		input   string
		span    loc.Span
		want    string
		wantErr bool
	}{
		{name: "empty input"},
		{name: "zero range", input: "abc"},
		{name: "full input", input: "abc", span: loc.Span{Start: 0, End: 3}, want: "abc"},
		{name: "half open", input: "abc", span: loc.Span{Start: 1, End: 2}, want: "b"},
		{name: "empty at eof", input: "abc", span: loc.Span{Start: 3, End: 3}},
		{name: "multibyte", input: "a€🙂z", span: loc.Span{Start: 1, End: 8}, want: "€🙂"},
		{name: "raw byte", input: "€", span: loc.Span{Start: 1, End: 2}, want: "\x82"},
		{name: "crlf", input: "a\r\nb", span: loc.Span{Start: 1, End: 3}, want: "\r\n"},
		{name: "unknown", input: "abc", span: loc.InvalidSpan(), wantErr: true},
		{name: "negative start", input: "abc", span: loc.Span{Start: -1, End: 1}, wantErr: true},
		{name: "reversed", input: "abc", span: loc.Span{Start: 2, End: 1}, wantErr: true},
		{name: "end past eof", input: "abc", span: loc.Span{Start: 0, End: 4}, wantErr: true},
		{name: "empty past eof", input: "abc", span: loc.Span{Start: 4, End: 4}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := tt.span.Slice(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Slice() error = %v, want error %t", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("Slice() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPosition(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		input      string
		offset     loc.ByteOffset
		wantLine   int
		wantColumn int
		wantErr    bool
	}{
		{name: "empty input", wantLine: 1, wantColumn: 1},
		{name: "start", input: "abc", wantLine: 1, wantColumn: 1},
		{name: "ascii eof", input: "abc", offset: 3, wantLine: 1, wantColumn: 4},
		{name: "multibyte", input: "a€🙂z", offset: 8, wantLine: 1, wantColumn: 4},
		{name: "inside multibyte", input: "a€🙂z", offset: 2, wantLine: 1, wantColumn: 2},
		{name: "combining code point", input: "e\u0301", offset: 3, wantLine: 1, wantColumn: 3},
		{name: "lf before break", input: "a\nb", offset: 1, wantLine: 1, wantColumn: 2},
		{name: "lf after break", input: "a\nb", offset: 2, wantLine: 2, wantColumn: 1},
		{name: "lf eof", input: "a\nb", offset: 3, wantLine: 2, wantColumn: 2},
		{name: "crlf before break", input: "a\r\nb", offset: 1, wantLine: 1, wantColumn: 2},
		{name: "crlf between bytes", input: "a\r\nb", offset: 2, wantLine: 1, wantColumn: 2},
		{name: "crlf after break", input: "a\r\nb", offset: 3, wantLine: 2, wantColumn: 1},
		{name: "crlf eof", input: "a\r\nb", offset: 4, wantLine: 2, wantColumn: 2},
		{name: "mixed lines", input: "€\r\n🙂\n", offset: 10, wantLine: 3, wantColumn: 1},
		{name: "consecutive breaks", input: "\n\r\n", offset: 3, wantLine: 3, wantColumn: 1},
		{name: "lone cr", input: "a\rb", offset: 3, wantLine: 1, wantColumn: 4},
		{name: "tab is one code point", input: "a\tb", offset: 3, wantLine: 1, wantColumn: 4},
		{name: "invalid utf8 byte", input: "a\xffb", offset: 2, wantLine: 1, wantColumn: 3},
		{name: "unknown", input: "abc", offset: loc.InvalidOffset, wantErr: true},
		{name: "negative", input: "abc", offset: -2, wantErr: true},
		{name: "past eof", input: "abc", offset: 4, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			line, column, err := loc.Position(tt.input, tt.offset)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Position() error = %v, want error %t", err, tt.wantErr)
			}
			if line != tt.wantLine || column != tt.wantColumn {
				t.Errorf("Position() = (%d, %d), want (%d, %d)", line, column, tt.wantLine, tt.wantColumn)
			}
		})
	}
}
