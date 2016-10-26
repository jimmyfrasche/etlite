package compile

import (
	"github.com/jimmyfrasche/etlite/internal/ast"
	"github.com/jimmyfrasche/etlite/internal/format/csvfmt"
	"github.com/jimmyfrasche/etlite/internal/format/rawfmt"
	"github.com/jimmyfrasche/etlite/internal/internal/eol"
	"github.com/jimmyfrasche/etlite/internal/internal/errint"
	"github.com/jimmyfrasche/etlite/internal/internal/errusr"
	"github.com/jimmyfrasche/etlite/internal/virt"
)

const (
	inputFormat  = true
	outputFormat = false
)

func (c *compiler) compileFormat(f ast.Format, read bool) {
	c.push(virt.ErrPos(f.Pos()))
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
	if f.Quote > 0 { //XXX unsupported currently since encoding/csv doesn't do that
		panic(errusr.New(f.Pos(), "specifying quotation for CSV is currently unsupported :("))
	}
	if f.Line != 0 && !read {
		panic(errusr.New(f.Pos(), "specifying line ending when writing CSV is unsupported"))
	}

	useCRLF := false
	switch f.Line {
	default:
		panic(errint.Newf("format csv undefined line ending %d", f.Line))
	case ast.DefaultLineEnding:
		useCRLF = eol.Default
	case ast.CRLF:
		useCRLF = true
	case ast.LF:
		useCRLF = false
	}

	if read { //decoder
		d := &csvfmt.Decoder{
			Null:     f.Null,
			Quote:    f.Quote,
			Comma:    f.Delim,
			NoHeader: f.NoHeader,
			UseCRLF:  useCRLF,
		}
		c.push(virt.SetDecoder(d))
	} else { //encoder
		e := &csvfmt.Encoder{
			Null:     f.Null,
			Quote:    f.Quote,
			Comma:    f.Delim,
			NoHeader: f.NoHeader,
			UseCRLF:  useCRLF,
		}
		c.push(virt.SetEncoder(e))
	}
}

func (c *compiler) formatRaw(f *ast.FormatRaw, read bool) {
	useCRLF := false
	switch f.Line {
	default:
		panic(errint.Newf("format raw undefined line ending %d", f.Line))
	case ast.DefaultLineEnding:
		useCRLF = eol.Default
	case ast.CRLF:
		useCRLF = true
	case ast.LF:
		useCRLF = false
	}

	if read { //decoder
		d := &rawfmt.Decoder{
			Tab:      f.Delim,
			UseCRLF:  useCRLF,
			Null:     f.Null,
			Strict:   f.Strict,
			NoHeader: !f.Header,
		}
		c.push(virt.SetDecoder(d))
	} else { //encoder
		e := &rawfmt.Encoder{
			Tab:      f.Delim,
			UseCRLF:  useCRLF,
			Null:     f.Null,
			NoHeader: !f.Header,
		}
		c.push(virt.SetEncoder(e))
	}
}

func (c *compiler) formatJSON(f *ast.FormatJSON, read bool) {
	panic(errint.New("json not implemented yet"))
}
