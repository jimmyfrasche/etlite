//Package writer exports a simplified and customized writer.
package writer

import (
	"fmt"
	"io"
	"strconv"
	"unicode/utf8"
)

type stringWriter interface {
	io.Writer
	WriteString(string) (int, error)
}

type fakeStringWriter struct {
	io.Writer
}

func (f *fakeStringWriter) WriteString(s string) (int, error) {
	return io.WriteString(f, s)
}

//Writer provides helpful utility methods and
//saves error handling till the end.
type Writer struct {
	stringWriter
	err error
}

//New wraps a standard io.Writer.
func New(w io.Writer) *Writer {
	if already, ok := w.(*Writer); ok {
		return already
	}
	out := &Writer{}
	if s, ok := w.(stringWriter); ok {
		out.stringWriter = s
	} else {
		out.stringWriter = &fakeStringWriter{w}
	}
	return out
}

//Unwrap returns the underlying writer.
func (w *Writer) Unwrap() io.Writer {
	if s, ok := w.stringWriter.(*fakeStringWriter); ok {
		return s.Writer
	}
	return w.stringWriter
}

//Err reports the first first error during processing.
func (w *Writer) Err() error {
	return w.err
}

//Sticky given a nonnil error sets that to the stickied error,
//halting any further output
func (w *Writer) Sticky(err error) *Writer {
	if w.err == nil {
		w.err = err
	}
	return w
}

//With runs f on this w if w is not in an error state
//and stickies any nonnil error returned by f.
func (w *Writer) With(f func(*Writer) error) *Writer {
	if w.err != nil {
		return w
	}
	return w.Sticky(f(w))
}

//Bs writes bytes.
func (w *Writer) Bs(p []byte) *Writer {
	if w.err != nil {
		return w
	}
	_, w.err = w.Write(p)
	return w
}

//Str writes strings.
func (w *Writer) Str(s string) *Writer {
	if w.err != nil {
		return w
	}
	_, w.err = w.WriteString(s)
	return w
}

//Stringer writes the result of f.String() if w is not in an error state.
func (w *Writer) Stringer(f fmt.Stringer) *Writer {
	if w.err != nil {
		return w
	}
	w.Str(f.String())
	return w
}

func (w *Writer) Int(i int) *Writer {
	return w.Str(strconv.Itoa(i))
}

type runeWriter interface {
	WriteRune(rune) (int, error)
}

//Rune writes a single rune.
func (w *Writer) Rune(r rune) *Writer {
	if w.err != nil {
		return w
	}
	switch r {
	case '\t':
		return w.Str("TAB")
	case '\'':
		return w.Str(`"'"`)
	}
	w.Str(`"`)
	if rw, ok := w.stringWriter.(runeWriter); ok {
		_, w.err = rw.WriteRune(r)
	} else {
		var buf [4]byte
		nb := utf8.EncodeRune(buf[:], r)
		_, w.err = w.Write(buf[:nb])
	}
	w.Str(`"`)
	return w
}

//Sp renders a space.
func (w *Writer) Sp() *Writer {
	w.Str(" ")
	return w
}

//Tab renders a tab.
func (w *Writer) Tab() *Writer {
	w.Str("\t")
	return w
}

//Nl renders a \n newline.
func (w *Writer) Nl() *Writer {
	w.Str("\n")
	return w
}
