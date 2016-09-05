package parse

import (
	"strconv"
	"strings"

	"github.com/jimmyfrasche/etlite/internal/ast"
	"github.com/jimmyfrasche/etlite/internal/internal/null"
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

	//here t cannot be STDIN or STDOUT, so it must be a filename or subquery
	var n ast.StringOrSQL
	if t.Kind == token.LParen {
		n = p.parseSQL(p.next(), true, true)
	} else {
		s, ok := t.Unescape()
		if !ok {
			panic(p.unexpected(t))
		}
		n = &ast.String{
			Position: t.Position,
			Value:    s,
		}
	}

	d := &ast.DeviceFile{
		Position: toFrom.Position,
		Name:     n,
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
	}
	t = p.next()
	if t.Literal("STRICT") {
		f.Strict = true
		t = p.next()
	}
	f.Delim, t = p.delim(t)
	f.Line, t = p.lineEnding(t)
	f.Null, t = p.null(t)
	f.Header, t = p.header(t)
	return f, t
}

func (p *parser) formatCSV(t token.Value) (ast.Format, token.Value) {
	f := &ast.FormatCSV{
		Position: t.Position,
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
	f.Header, t = p.header(t)
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

func (p *parser) delim(t token.Value) (ast.RuneOrSQL, token.Value) {
	if t.Literal("DELIM") || t.Literal("DELIMITER") {
		return p.runeOrSq(p.next(), "delimiter")
	}
	return nil, t
}

func (p *parser) quote(t token.Value) (ast.RuneOrSQL, token.Value) {
	if t.Literal("QUOTE") {
		return p.runeOrSq(p.next(), "quote")
	}
	return nil, t
}

func (p *parser) null(t token.Value) (ast.NullOrSQL, token.Value) {
	if t.Literal("NULL") {
		var n ast.NullOrSQL
		if n, t = p.maybeSq(t); n != nil {
			return n, t
		}
		s, ok := t.Unescape()
		if !ok {
			panic(p.expected("null encoding", t))
		}
		return &ast.Null{
			Position: t.Position,
			Value:    null.Encoding(s),
		}, p.next()
	}
	return nil, t
}

func (p *parser) header(t token.Value) (ast.BoolOrSQL, token.Value) {
	if t.Literal("HEADER") {
		t = p.next()
		var n ast.BoolOrSQL
		if n, t = p.maybeSq(t); n != nil {
			return n, t
		}

		s, ok := t.Unescape()
		if !ok {
			panic(p.expected("boolean", t))
		}
		b, err := strconv.ParseBool(strings.ToUpper(s))
		if err != nil {
			panic(p.expected("boolean", t))
		}
		return &ast.Bool{
			Position: t.Position,
			Value:    b,
		}, p.next()
	}
	return nil, t
}
