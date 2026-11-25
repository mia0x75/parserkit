package format

import "testing"

func TestOutputFormat(t *testing.T) {
	t.Parallel()
	if got := OutputFormat("\x00'\n\r\\€"); got != `\0''\n\r\€` {
		t.Fatalf("escaped literal = %q", got)
	}
}
