package csvfmt

import (
	"encoding/csv"
	"fmt"
	"io"

	"github.com/jimmyfrasche/etlite/internal/format"
	"github.com/jimmyfrasche/etlite/internal/internal/null"
	"github.com/jimmyfrasche/etlite/internal/stdio"
)

//Decoder is a CSV decoder.
type Decoder struct {
	Null             null.Encoding
	Comma, Comment   rune
	TrimLeadingSpace bool //unused

	Strict   bool
	NoHeader bool

	csv *csv.Reader
	acc []*string
	hdr []string

	nm  string
	lno int
}

func (d *Decoder) ctx() string {
	return fmt.Sprintf("%s:%d:", d.nm, d.lno)
}

var _ format.Decoder = (*Decoder)(nil)

//ReadHeader configures the decoder and returns the first row of the CSV.
func (d *Decoder) ReadHeader(possibleHeader []string, in stdio.Reader) (string, []string, error) {
	if len(possibleHeader) == 0 && d.NoHeader {
		return "", nil, format.ErrNoHeader
	}
	d.nm = in.Name()
	d.lno = 1
	d.csv = csv.NewReader(in.Unwrap())
	d.csv.Comma = d.Comma
	d.csv.Comment = d.Comment
	d.csv.TrimLeadingSpace = d.TrimLeadingSpace
	d.hdr = possibleHeader
	if d.NoHeader {
		if cap(d.acc) < len(d.hdr) {
			d.acc = make([]*string, 0, len(d.hdr))
		}
		return "", d.hdr, nil
	}

	hdr, err := d.csv.Read()
	if err != nil {
		if err == io.EOF {
			return "", nil, format.ErrNoHeader
		}
		return "", nil, wrap(d, err)
	}
	if d.Strict && len(d.hdr) != len(hdr) && len(d.hdr) != 0 {
		return "", nil, format.NewDimErr(d.ctx(), len(d.hdr), len(hdr))
	}
	if len(d.hdr) == 0 {
		d.hdr = hdr
	}
	//preallocate scratch space
	if cap(d.acc) < len(d.hdr) {
		d.acc = make([]*string, 0, len(d.hdr))
	}
	d.lno++
	return "", d.hdr, nil
}

//Skip rows.
func (d *Decoder) Skip(rows int) error {
	for i := 0; i < rows; i++ {
		_, err := d.csv.Read()
		if err != nil {
			return wrap(d, err)
		}
		d.lno++
	}
	return nil
}

//ReadRow reads a row from the CSV and handles NULL decoding.
func (d *Decoder) ReadRow() ([]*string, error) {
	row, err := d.csv.Read()
	if err != nil {
		return nil, wrap(d, err)
	}
	d.lno++
	if d.Strict && len(row) != len(d.hdr) {
		return nil, format.NewDimErr(d.ctx(), len(d.hdr), len(row))
	}
	if len(row) > len(d.hdr) {
		//not in strict mode, ignore extra fields
		row = row[:len(d.hdr)]
	}
	d.acc = d.acc[:0]
	for _, col := range row {
		d.acc = append(d.acc, d.Null.Encode(col))
	}
	if len(d.acc) < len(d.hdr) {
		//not in strict mode, need to add in extra nulls
		for i := len(d.acc); i < len(d.hdr); i++ {
			d.acc = append(d.acc, nil)
		}
	}
	return d.acc, nil
}

//Reset decouples the CSV reader and zeroes internal scratch space.
func (d *Decoder) Reset() error {
	d.csv = nil
	for i := range d.acc {
		d.acc[i] = nil
	}
	d.acc = d.acc[:0]
	return nil
}

//Close is a no-op.
func (*Decoder) Close() error {
	return nil
}
