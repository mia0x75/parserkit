// Package format provides error-preserving SQL output primitives.
package format

import (
	"errors"
	"fmt"
	"io"
	"reflect"
)

var ErrNilWriter = errors.New("format: restore writer is nil")

type RestoreWriter struct {
	out io.Writer
	err error
}

// NewRestoreWriter wraps out, remembering the first write error. A typed
// nil boxed into io.Writer is rejected up front: it would panic on Write.
func NewRestoreWriter(out io.Writer) RestoreWriter {
	w := RestoreWriter{out: out}
	if out == nil || isNilPointer(out) {
		w.err = ErrNilWriter
	}
	return w
}

func isNilPointer(v any) bool {
	value := reflect.ValueOf(v)
	kind := value.Kind()
	return kind == reflect.Pointer && value.IsNil()
}

func (w *RestoreWriter) Err() error {
	if w.err == nil && w.out == nil {
		return ErrNilWriter
	}
	return w.err
}

func (w *RestoreWriter) record(n, want int, err error) {
	if err == nil && n != want {
		err = io.ErrShortWrite
	}
	if err != nil && w.err == nil {
		w.err = fmt.Errorf("write restored sql: %w", err)
	}
}

func (w *RestoreWriter) Write(p []byte) (int, error) {
	if err := w.Err(); err != nil {
		return 0, err
	}
	if len(p) == 0 {
		return 0, nil
	}
	n, err := w.out.Write(p)
	w.record(n, len(p), err)
	return n, w.err
}

func (w *RestoreWriter) writeString(s string) int {
	if w.Err() != nil || s == "" {
		return 0
	}
	n, err := io.WriteString(w.out, s)
	w.record(n, len(s), err)
	return n
}

func (w *RestoreWriter) WriteString(s string) (int, error) {
	n := w.writeString(s)
	return n, w.Err()
}

var _ io.Writer = (*RestoreWriter)(nil)
var _ io.StringWriter = (*RestoreWriter)(nil)
