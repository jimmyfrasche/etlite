package parse

import (
	"github.com/jimmyfrasche/etlite/internal/ast"
	"github.com/jimmyfrasche/etlite/internal/parse/internal/fmtname"
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
		i, _ := p.importStmt(t, false, true, nil)
		return i
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
		a.Subquery = &ast.SQL{
			Tokens: []token.Value{t},
		}
	}

	p.expect(token.Semicolon)

	return a
}

//DISPLAY [TO device] [AS format] [FRAME name]
func (p *parser) displayStmt(t token.Value) *ast.Display {
	d := &ast.Display{
		Position: t.Position,
	}
	if t.Literal("TO") {
		d.Device, t = p.deviceExpr(t)
	}
	d.Frame, t = p.frameExpr(t)
	if t.Literal("AS") {
		d.Format, t = p.formatExpr(p.next())
	}
	if t.Kind != token.Semicolon {
		panic(p.expected(token.Semicolon, t))
	}
	return d
}

//IMPORT [TEMP] [table] [header] [FROM device] [WITH format] [FRAME name] [LIMIT n] [OFFSET n]
func (p *parser) importStmt(t token.Value, subquery, compound bool, sql *ast.SQL) (ast.Node, token.Value) {
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

	if t.Kind == token.Literal && !t.AnyLiteral("FROM", "WITH", "FRAME", "LIMIT", "OFFSET", "UNION", "INTERSECT", "EXCEPT") {
		var name []token.Value
		t, name = p.name(t)
		s, err := fmtname.ToString(name)
		if err != nil {
			panic(err)
		}
		i.Name = s
	}

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
			t = p.next()
		}
	}

	if t.Literal("FROM") {
		i.Device, t = p.deviceExpr(t)
	}

	if t.Literal("WITH") {
		i.Format, t = p.formatExpr(p.next())
	}

	i.Frame, t = p.frameExpr(t)

	if t.Literal("LIMIT") {
		i.Limit, t = p.int(p.next())
	}

	if t.Literal("OFFSET") {
		i.Offset, t = p.int(p.next())
	}

	if t.AnyLiteral("UNION", "INTERSECT", "EXCEPT") {
		if !compound {
			panic(p.unexpected(t))
		}
		//first term in a compound chain is import, lift result into sql
		if sql == nil {
			sql, t = p.liftSQL(t, i)
			return sql, t
		}
		//otherwise we're already in the sql parser
		//and we know we're not done parsing.
		return i, t
	}

	end := token.Semicolon
	if subquery {
		end = token.RParen
	}
	if t.Kind != end {
		panic(p.expected(end, t))
	}
	return i, t
}

//Any random, regular SQL.
func (p *parser) parseSQL(t token.Value, subquery, allowETLsq bool) *ast.SQL {
	sp := newSqlParser(p)
	sp.top(t, subquery, allowETLsq)
	return sp.sql
}

func (p *parser) liftSQL(t token.Value, i *ast.Import) (*ast.SQL, token.Value) {
	sp := newSqlParser(p)
	sp.synth(t, token.Placeholder) //for the import
	sp.sql.Subqueries = append(sp.sql.Subqueries, i)
	t = sp.regular(t, 0, false, true, true)
	return sp.sql, t
}
