package parse

import (
	"github.com/jimmyfrasche/etlite/internal/ast"
	"github.com/jimmyfrasche/etlite/internal/parse/internal/interpolate"
	"github.com/jimmyfrasche/etlite/internal/token"
)

//parseETL is the initial state that dispatches to other productions.
func (p *parser) parseETL(t token.Value) ast.Node {
	if !t.Head(false) {
		panic(p.unexpected(t))
	}
	switch t.Canon {
	case "USE":
		return p.useStmt(t)
	case "ASSERT":
		return p.assertStmt(t)
	case "DISPLAY":
		return p.displayStmt(t)
	case "IMPORT":
		return p.importStmt(t, false)
	default:
		return p.parseSQL(t, false, true)
	}
}

//USE [DB|DATABASE] "name"
func (p *parser) useStmt(t token.Value) *ast.Use {
	u := &ast.Use{
		Position: t.Position,
	}

	t = p.next()
	if t.AnyLiteral("DATABASE", "DB") {
		t = p.next()
	}

	nm, ok := t.Unescape()
	if !ok {
		panic(p.unexpected(t))
	}
	u.DB = nm

	p.expect(token.Semicolon)

	return u
}

//ASSERT "message", subquery
func (p *parser) assertStmt(t token.Value) *ast.Assert {
	a := &ast.Assert{
		Position: t.Position,
	}

	t = p.expect(token.String)
	a.Message = t

	t = p.expectLit(",")

	t = p.next()
	switch t.Kind {
	default:
		panic(p.expected("@ or subquery", t))
	case token.LParen:
		a.Subquery = p.parseSQL(t, true, false)
		//trim off ()
		a.Subquery.Tokens = a.Subquery.Tokens[1 : len(a.Subquery.Tokens)-1]
	case token.Argument:
		ts, err := interpolate.DesugarAssert(t)
		if err != nil {
			panic(p.mkErr(t, err))
		}
		a.Subquery = &ast.SQL{
			Tokens: ts,
		}
	}

	p.expect(token.Semicolon)

	return a
}

//DISPLAY [format] [device]
func (p *parser) displayStmt(t token.Value) *ast.Display {
	d := &ast.Display{
		Position: t.Position,
	}
	d.Format, t = p.formatExpr(p.next())
	if t.Literal("TO") {
		d.Device, t = p.deviceExpr(t)
	}
	if t.Kind != token.Semicolon {
		panic(p.expected(token.Semicolon, t))
	}
	return d
}

//IMPORT [format] [header] [device] [table] [limit] [offset]
func (p *parser) importStmt(t token.Value, subquery bool) *ast.Import {
	i := &ast.Import{
		Position: t.Position,
		Header:   make([]string, 0, 16),
		Limit:    -1,
		Offset:   -1,
	}
	t = p.next()
	if t.AnyLiteral("TEMP", "TEMPORARY") {
		i.Temporary = true
		t = p.next()
	}
	i.Format, t = p.formatExpr(t)

	//slurp header
	if t.Kind == token.LParen {
		t = p.next()
		for {
			f, ok := t.Unescape()
			if !ok {
				panic(p.unexpected(t))
			}
			i.Header = append(i.Header, f)
			t = p.next()
			if t.Kind == token.RParen {
				t = p.next()
				break
			}
			if !t.Literal(",") {
				panic(p.unexpected(t))
			}
		}
	}

	if t.Literal("FRAME") {
		t = p.expectLitOrStr()
		s, _ := t.Unescape()
		i.Frame = s
		t = p.next()
	}

	if t.Literal("FROM") {
		i.Device, t = p.deviceExpr(t)
	}

	end := token.Semicolon
	if subquery {
		end = token.RParen
	}
	if !t.Literal("LIMIT") && !t.Literal("OFFSET") && t.Kind != end {
		//TODO allow qualified names, use name from new sql parser but turn it into a string:
		//pull impl from compiler
		_, ok := t.Unescape()
		if !ok {
			panic(p.expected("table name", t))
		}
		i.Table = t.Value
		t = p.next()
	}

	if t.Literal("LIMIT") {
		i.Limit = p.int(p.next())
		t = p.next()
	}

	if t.Literal("OFFSET") {
		i.Offset = p.int(p.next())
		t = p.next()
	}

	if t.Kind != end {
		panic(p.expected(end, t))
	}

	return i
}

//Any random, regular SQL.
func (p *parser) parseSQL(t token.Value, subquery, allowETLsq bool) *ast.SQL {
	sp := newSqlParser(p)
	sp.top(t, subquery, allowETLsq)
	return sp.sql
}
