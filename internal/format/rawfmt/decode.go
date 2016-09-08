package rawfmt

import (
	"bufio"
	"fmt"
	"io"

	"github.com/jimmyfrasche/etlite/internal/device"
	"github.com/jimmyfrasche/etlite/internal/format"
	"github.com/jimmyfrasche/etlite/internal/internal/errsys"
	"github.com/jimmyfrasche/etlite/internal/internal/null"
)

//Decoder decodes the raw format
type Decoder struct {
	Tab     rune
	UseCRLF bool //True to use \r\n as the line terminator, otherwise \n.
	Null    null.Encoding

	Strict   bool //When true reports an error if there are more or less fields than required
	NoHeader bool //True if there is no header in the input

	hdr []string //we stash this here if a header is provided and none in input
	r   *bufio.Reader
	err error
	nm  string
	lno int

	sacc []rune
	facc []string
	racc []*string

	resumed bool
}

func (d *Decoder) ctx() string {
	return fmt.Sprintf("%s:%d:", d.nm, d.lno)
}

func (*Decoder) Name() string {
	return "RAW"
}

var _ format.Decoder = (*Decoder)(nil)

func (d *Decoder) Init(r device.Reader) error {
	d.nm = r.Name()
	d.r = r.Unwrap()
	d.resumed = false
	d.lno = 1
	return nil
}

//ReadHeader decodes the header
func (d *Decoder) ReadHeader(_ string, potentialHeader []string) ([]string, error) {
	if !d.resumed && len(potentialHeader) == 0 && d.NoHeader {
		return nil, format.ErrNoHeader
	}

	//reset/cfg internal buffers
	d.hdr = potentialHeader
	sz := len(potentialHeader)
	if cap(d.facc) < sz {
		d.facc = make([]string, 0, sz)
	}
	if cap(d.racc) < sz {
		d.racc = make([]*string, 0, sz)
	}
	if d.sacc == nil {
		d.sacc = make([]rune, 0, 1024)
	}
	d.sacc = d.sacc[:0]
	d.facc = d.facc[:0]
	d.racc = d.racc[:0]
	if d.resumed || d.NoHeader {
		return d.hdr, nil
	}

	rs, err := d.read()
	if err != nil {
		return nil, errsys.WrapWith(d.ctx(), err)
	}
	if d.Strict && len(rs) != len(d.hdr) && len(d.hdr) != 0 {
		return nil, format.NewDimErr(d.ctx(), len(d.hdr), len(rs))
	}
	if len(d.hdr) == 0 {
		d.hdr = rs
	}
	if cap(d.racc) < len(d.hdr) {
		d.racc = make([]*string, 0, len(d.hdr))
	}
	return rs, nil
}

//Skip rows.
func (d *Decoder) Skip(rows int) error {
	//XXX this could be made much more performant if it ignores formatting and just recognizes.
	for i := 0; i < rows; i++ {
		if _, err := d.ReadRow(); err != nil {
			return err
		}
	}
	return nil
}

//ReadRow reads an individual row
func (d *Decoder) ReadRow() ([]*string, error) {
	rs, err := d.read()
	if err != nil {
		return nil, errsys.WrapWith(d.ctx(), err)
	}
	if d.Strict && len(rs) != len(d.hdr) {
		return nil, format.NewDimErr(d.ctx(), len(d.hdr), len(rs))
	}
	if len(rs) > len(d.hdr) {
		//not in strict mode, ignore extra fields
		rs = rs[:len(d.hdr)]
	}
	d.racc = d.racc[:0]
	for _, r := range rs {
		d.racc = append(d.racc, d.Null.Encode(r))
	}
	if len(d.racc) < len(d.hdr) {
		//not in strict mode, need to add in additional nulls
		for i := len(d.racc); i < len(d.hdr); i++ {
			d.racc = append(d.racc, nil)
		}
	}
	return d.racc, nil
}

//Reset the decoder for reuse
func (d *Decoder) Reset() error {
	d.hdr = nil
	d.sacc = d.sacc[:0]
	d.facc = d.facc[:0]
	for i := range d.racc {
		d.racc[i] = nil
	}
	d.racc = d.racc[:0]
	d.resumed = true
	return nil
}

//Close the decoder
func (d *Decoder) Close() error {
	d.r = nil
	return nil
}

func (d *Decoder) readField() (f string, eol bool, err error) {
	if d.err != nil {
		err := d.err
		d.err = nil
		return "", false, err
	}
	d.sacc = d.sacc[:0]
	for {
		c, _, err := d.r.ReadRune()
		switch {
		default:
			d.sacc = append(d.sacc, c)
		case c == d.Tab: // new field
			return string(d.sacc), false, nil
		case !d.UseCRLF && c == '\n': //newline
			d.lno++
			return string(d.sacc), true, nil
		case err != nil: //new line if EOF
			d.err = err
			return string(d.sacc), err == io.EOF, nil
		case d.UseCRLF && c == '\r': //the awful case
			c, _, err = d.r.ReadRune()
			if c == '\n' { //got \r\n
				d.lno++
				return string(d.sacc), true, nil
			}
			//otherwise record initial \r and put back c
			d.sacc = append(d.sacc, '\r')
			_ = d.r.UnreadRune() //only an error if nothing read yet
			//if the second read returned an error bail with what we have now
			if err != nil {
				d.err = err
				return string(d.sacc), err == io.EOF, nil
			}
		}
	}
}

func (d *Decoder) read() ([]string, error) {
	if d.err != nil {
		err := d.err
		d.err = nil
		return nil, err
	}
	d.facc = d.facc[:0]
	for {
		f, eol, err := d.readField()
		if err != nil {
			d.err = err
			return d.facc, nil
		}
		d.facc = append(d.facc, f)
		if eol {
			return d.facc, nil
		}
	}
}
