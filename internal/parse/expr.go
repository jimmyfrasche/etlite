package parse

import (
	"strconv"

	"github.com/jimmyfrasche/etlite/internal/ast"
	"github.com/jimmyfrasche/etlite/internal/internal/null"
	"github.com/jimmyfrasche/etlite/internal/parse/internal/runefrom"
	"github.com/jimmyfrasche/etlite/internal/token"
)

//TO|FROM STDIN|STDOUT|filename
func (p *parser) deviceExpr(toFrom token.Value) (ast.Device, token.Value) {
	t := p.next()

	if toFrom.Literal("TO") {
		if t.Literal("STDIN") {
			panic(p.errMsg(t, "expected STDIN or filename, got STDOUT"))
		} else if t.Literal("STDOUT") {
			return &ast.DeviceStdio{toFrom.Position}, p.next()
		}
	} else {
		if t.Literal("STDIN") {
			return &ast.DeviceStdio{toFrom.Position}, p.next()
		} else if t.Literal("STDOUT") {
			panic(p.errMsg(t, "expected STDIN or filename, got STDOUT"))
		}
	}

	//here t cannot be STDIN or STDOUT, so it must be a filename
	_, ok := t.Unescape()
	if !ok {
		panic(p.unexpected(t))
	}

	d := &ast.DeviceFile{
		Name: t,
	}
	return d, p.next()
}

func (p *parser) formatExpr(t token.Value) (ast.Format, token.Value) {
	switch t.Canon {
	case "JSON":
		return p.formatJSON(t)
	case "RAW":
		return p.formatRaw(t)
	case "CSV":
		return p.formatCSV(t)
	default:
		return nil, t
	}
}

func (p *parser) formatJSON(t token.Value) (ast.Format, token.Value) {
	return &ast.FormatJSON{
		Position: t.Position,
	}, p.next()
}

func (p *parser) formatRaw(t token.Value) (ast.Format, token.Value) {
	f := &ast.FormatRaw{
		Position: t.Position,
		Delim:    -1,
	}
	t = p.next()
	if t.Literal("STRICT") {
		f.Strict = true
		t = p.next()
	}
	f.Delim, t = p.delim(t)
	f.Line, t = p.lineEnding(t)
	f.Null, t = p.null(t)
	if t.Literal("HEADER") || t.Literal("HDR") {
		f.Header = true
		t = p.next()
	}
	return f, t
}

func (p *parser) formatCSV(t token.Value) (ast.Format, token.Value) {
	f := &ast.FormatCSV{
		Position: t.Position,
		Delim:    -1,
		Quote:    -1,
	}
	t = p.next()
	if t.Literal("STRICT") {
		f.Strict = true
		t = p.next()
	}
	f.Delim, t = p.delim(t)
	f.Quote, t = p.quote(t)
	f.Line, t = p.lineEnding(t)
	f.Null, t = p.null(t)
	if t.Literal("NOHEADER") || t.Literal("NOHDR") {
		f.NoHeader = true
		t = p.next()
	}
	return f, t
}

func (p *parser) lineEnding(t token.Value) (ast.LineEnding, token.Value) {
	if !t.Literal("EOL") {
		return ast.DefaultLineEnding, t
	}
	t = p.expect(token.Literal)
	switch t.Canon {
	case "DEFAULT":
		return ast.DefaultLineEnding, p.next()
	case "LF", "UNIX":
		return ast.LF, p.next()
	case "CRLF", "WINDOWS":
		return ast.CRLF, p.next()
	default:
		panic(p.expected("a line ending (DEFAULT, LF or UNIX, CRLF or WINDOWS)", t))
	}
}

func (p *parser) rune(t token.Value) (rune, token.Value) {
	if t.Literal("TAB") {
		return '\t', p.next()
	}
	s, ok := t.Unescape()
	if !ok {
		panic(p.unexpected(t))
	}
	r, err := runefrom.String(s)
	if err != nil {
		panic(p.mkErr(t, err))
	}
	return r, p.next()
}

func (p *parser) delim(t token.Value) (rune, token.Value) {
	if t.Literal("DELIM") || t.Literal("DELIMITER") {
		return p.rune(p.next())
	}
	return -1, t
}

func (p *parser) int(t token.Value) int {
	if t.Kind != token.Literal {
		panic(p.unexpected(t))
	}
	i, err := strconv.Atoi(t.Value)
	if err != nil {
		panic(p.mkErr(t, err))
	}
	return i
}

func (p *parser) quote(t token.Value) (rune, token.Value) {
	if t.Literal("QUOTE") {
		return p.rune(p.next())
	}
	return -1, t
}

func (p *parser) null(t token.Value) (null.Encoding, token.Value) {
	if t.Literal("NULL") {
		t = p.next()
		s, ok := t.Unescape()
		if !ok {
			panic(p.expected("null encoding", t))
		}
		return null.Encoding(s), p.next()
	}
	return "", t
}
