package compile

import (
	"github.com/jimmyfrasche/etlite/internal/ast"
	"github.com/jimmyfrasche/etlite/internal/format/csvfmt"
	"github.com/jimmyfrasche/etlite/internal/format/rawfmt"
	"github.com/jimmyfrasche/etlite/internal/internal/eol"
	"github.com/jimmyfrasche/etlite/internal/internal/errint"
	"github.com/jimmyfrasche/etlite/internal/internal/errusr"
	"github.com/jimmyfrasche/etlite/internal/internal/null"
	"github.com/jimmyfrasche/etlite/internal/virt"
)

const (
	inputFormat  = true
	outputFormat = false
)

func (c *compiler) compileFormat(f ast.Format, read bool) {
	switch f := f.(type) {
	default:
		panic(errint.Newf("unrecognized Format type: %T", f))

	case nil:
		return

	case *ast.FormatCSV:
		c.formatCSV(f, read)

	case *ast.FormatRaw:
		c.formatRaw(f, read)

	case *ast.FormatJSON:
		c.formatJSON(f, read)
	}
}

func (c *compiler) formatCSV(f *ast.FormatCSV, read bool) {
	c.runeOrSub(f.Delim, ',')
	if f.Quote != nil { //XXX unsupported currently since encoding/csv doesn't do that
		panic(errusr.New(f.Quote.Pos(), "specifying quotation for CSV is currently unsupported :("))
	}
	c.runeOrSub(f.Quote, '"')
	if f.Line != 0 && !read {
		panic(errusr.New(f.Pos(), "specifying line ending when writing CSV is unsupported"))
	}
	c.nullOrSub(f.Null, null.Encoding(""))
	c.boolOrSub(f.Header, true)

	useCRLF := false
	switch f.Line {
	case ast.DefaultLineEnding:
		useCRLF = eol.Default
	case ast.CRLF:
		useCRLF = true
	case ast.LF:
		useCRLF = false
	}

	back := func(m *virt.Machine) (delim, quote rune, n null.Encoding, hdr bool, err error) {
		hdr, err = m.PopBool()
		if err != nil {
			return
		}
		n, err = m.PopNullEncoding()
		if err != nil {
			return
		}
		quote, err = m.PopRune()
		if err != nil {
			return
		}
		delim, err = m.PopRune()
		return
	}

	if read { //decoder
		c.push(func(m *virt.Machine) error {
			d, _, n, hdr, err := back(m)
			if err != nil {
				return err
			}
			return m.SetDecoder(&csvfmt.Decoder{
				Null:     n,
				Comma:    d,
				Strict:   f.Strict,
				NoHeader: hdr,
			})
		})
	} else { //encoder
		c.push(func(m *virt.Machine) error {
			d, _, n, hdr, err := back(m)
			if err != nil {
				return err
			}
			return m.SetEncoder(&csvfmt.Encoder{
				Null:     n,
				Comma:    d,
				UseCRLF:  useCRLF,
				NoHeader: hdr,
			})
		})
	}
}

func (c *compiler) formatRaw(f *ast.FormatRaw, read bool) {
	c.runeOrSub(f.Delim, '\t')
	c.nullOrSub(f.Null, null.Encoding(""))
	c.boolOrSub(f.Header, true)

	useCRLF := false
	switch f.Line {
	case ast.DefaultLineEnding:
		useCRLF = eol.Default
	case ast.CRLF:
		useCRLF = true
	case ast.LF:
		useCRLF = false
	}

	back := func(m *virt.Machine) (delim rune, n null.Encoding, hdr bool, err error) {
		hdr, err = m.PopBool()
		if err != nil {
			return
		}
		n, err = m.PopNullEncoding()
		if err != nil {
			return
		}
		delim, err = m.PopRune()
		return
	}

	if read { //decoder
		c.push(func(m *virt.Machine) error {
			d, n, hdr, err := back(m)
			if err != nil {
				return err
			}
			return m.SetDecoder(&rawfmt.Decoder{
				Tab:      d,
				UseCRLF:  useCRLF,
				Null:     n,
				Strict:   f.Strict,
				NoHeader: hdr,
			})
		})
	} else { //encoder
		c.push(func(m *virt.Machine) error {
			d, n, hdr, err := back(m)
			if err != nil {
				return err
			}
			return m.SetEncoder(&rawfmt.Encoder{
				Tab:      d,
				UseCRLF:  useCRLF,
				Null:     n,
				NoHeader: hdr,
			})
		})
	}
}

func (c *compiler) formatJSON(f *ast.FormatJSON, read bool) {
	panic(errint.New("json not implemented yet"))
}
