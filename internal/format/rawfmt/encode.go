package rawfmt

import (
	"fmt"
	"strings"

	"github.com/jimmyfrasche/etlite/internal/format"
	"github.com/jimmyfrasche/etlite/internal/internal/errsys"
	"github.com/jimmyfrasche/etlite/internal/internal/null"
	"github.com/jimmyfrasche/etlite/internal/stdio"
)

//Encoder encodes the raw format
type Encoder struct {
	Tab     rune //If undefined, defaults to \t
	UseCRLF bool //True to use \r\n as the line terminator, otherwise \n.
	Null    null.Encoding

	NoHeader bool //If true, do not write header to output

	w        stdio.Writer
	tab, eol string
	lno      int
}

var _ format.Encoder = (*Encoder)(nil)

func (e *Encoder) ctx() string {
	return fmt.Sprintf("%s:%d:", e.w.Name(), e.lno)
}

func (e *Encoder) write(s string) error {
	_, err := e.w.WriteString(s)
	if err != nil {
		return errsys.WrapWith(e.ctx(), err)
	}
	return nil
}

//WriteHeader encodes the header.
func (e *Encoder) WriteHeader(hdr []string, w stdio.Writer) error {
	e.w = w
	e.eol = "\n"
	if e.UseCRLF {
		e.eol = "\r\n"
	}
	if e.Tab == 0 {
		e.Tab = '\t'
	}
	e.tab = string([]rune{e.Tab})
	e.lno = 1

	if e.NoHeader {
		return nil
	}
	line := strings.Join(hdr, e.tab)
	if err := e.write(line); err != nil {
		return err
	}
	e.lno++
	return e.write(e.eol)
}

//WriteRow encodes a row
func (e *Encoder) WriteRow(row []*string) error {
	for i, s := range row {
		if err := e.write(e.Null.Decode(s)); err != nil {
			return err
		}
		if i != len(row) {
			if err := e.write(e.tab); err != nil {
				return err
			}
		}
	}
	e.lno++
	return e.write(e.eol)
}

//Reset the encoder for reuse.
func (e *Encoder) Reset() error {
	e.w = nil
	return nil
}

//Close the encoder.
func (*Encoder) Close() error {
	return nil
}
