package csvfmt

import (
	"encoding/csv"
	"fmt"

	"github.com/jimmyfrasche/etlite/internal/format"
	"github.com/jimmyfrasche/etlite/internal/null"
	"github.com/jimmyfrasche/etlite/internal/stdio"
)

//Encoder is a CSV encoder.
type Encoder struct {
	Null    null.Encoding
	Comma   rune
	UseCRLF bool

	NoHeader bool

	csv *csv.Writer
	acc []string

	nm  string
	lno int
}

var _ format.Encoder = (*Encoder)(nil)

func (e *Encoder) ctx() string {
	return fmt.Sprintf("%s:%d:", e.nm, e.lno)
}

//WriteHeader writes the header as the first row of the CSV.
func (e *Encoder) WriteHeader(hdr []string, w stdio.Writer) error {
	e.nm = w.Name()
	e.lno = 1
	e.csv = csv.NewWriter(w.Unwrap())
	if e.Comma == 0 {
		e.Comma = ','
	}
	e.csv.Comma = e.Comma
	e.csv.UseCRLF = e.UseCRLF
	if e.acc == nil {
		e.acc = make([]string, 0, len(hdr))
	}
	if e.NoHeader {
		return nil
	}
	e.lno++
	return wrap(e, e.csv.Write(hdr))
}

//WriteRow writes a row to the CSV, handling all NULL encodings.
func (e *Encoder) WriteRow(row []*string) error {
	e.acc = e.acc[:0]
	for _, col := range row {
		e.acc = append(e.acc, e.Null.Decode(col))
	}
	e.lno++
	return wrap(e, e.csv.Write(e.acc))
}

//Reset flushes and decouples the CSV writer.
func (e *Encoder) Reset() error {
	e.csv.Flush()
	err := e.csv.Error()
	e.csv = nil
	return wrap(e, err)
}

//Close is a no-op.
func (*Encoder) Close() error {
	return nil
}
