package csvfmt

import (
	"encoding/csv"
	"fmt"

	"github.com/jimmyfrasche/etlite/internal/device"
	"github.com/jimmyfrasche/etlite/internal/format"
	"github.com/jimmyfrasche/etlite/internal/internal/null"
)

//Encoder is a CSV encoder.
type Encoder struct {
	Null    null.Encoding
	Comma   rune
	Quote   rune //unused
	UseCRLF bool

	NoHeader bool

	csv *csv.Writer
	acc []string

	nm  string
	lno int

	resumed bool
}

var _ format.Encoder = (*Encoder)(nil)

func (e *Encoder) ctx() string {
	return fmt.Sprintf("%s:%d:", e.nm, e.lno)
}

func (*Encoder) Name() string {
	return "CSV"
}

func (e *Encoder) Init(w device.Writer) error {
	e.nm = w.Name()
	e.csv = csv.NewWriter(w.Unwrap())
	if e.Quote < 0 {
		e.Quote = '"'
	}
	if e.Comma < 0 {
		e.Comma = ','
	}
	e.csv.Comma = e.Comma
	e.csv.UseCRLF = e.UseCRLF
	e.resumed = false
	e.lno = 1
	return nil
}

//WriteHeader writes the header as the first row of the CSV.
func (e *Encoder) WriteHeader(_ string, hdr []string) error {
	if e.acc == nil {
		e.acc = make([]string, 0, len(hdr))
	}
	if e.NoHeader || e.resumed {
		return nil
	}

	if err := e.csv.Write(hdr); err != nil {
		return wrap(e, err)
	}
	e.lno++
	return nil
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
	e.resumed = true
	return wrap(e, e.csv.Error())
}

//Close is a no-op.
func (e *Encoder) Close() error {
	e.csv = nil
	return nil
}
