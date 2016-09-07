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
	case "DISPLAY":
		return p.displayStmt(t)
	case "IMPORT":
		return p.importStmt(t, false)
	default:
		return p.parseSQL(t, false, true)
	}
}

//USE [DATABASE] "name"
func (p *parser) useStmt(t token.Value) *ast.Use {
	u := &ast.Use{
		Position: t.Position,
	}

	t = p.next()
	if t.Literal("DATABASE") || t.Literal("DB") {
		t = p.next()
	}

	nm, ok := t.Unescape()
	if !ok {
		panic(p.expected(token.String, t))
	}
	u.DB = nm

	p.expect(token.Semicolon)

	return u
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
	}
	t = p.next()
	if t.Literal("TEMP") || t.Literal("TEMPORARY") {
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
			if !t.Comma() {
				panic(p.unexpected(t))
			}
		}
	}

	if t.Literal("FROM") {
		i.Device, t = p.deviceExpr(t)
	}

	if !t.Literal("LIMIT") && !t.Literal("OFFSET") {
		s, ok := t.Unescape()
		if !ok {
			panic(p.expected("table name", t))
		}
		i.Table = s
		t = p.next()
	}

	if t.Literal("LIMIT") {
		t = p.next()
		i.Limit, t = p.intOrSq(t, true)
	}

	if t.Literal("OFFSET") {
		t = p.next()
		i.Offset, t = p.intOrSq(t, false)
	}

	if subquery && t.Kind != token.RParen {
		panic(p.expected(token.RParen, t))
	}
	if !subquery && t.Kind != token.Semicolon {
		panic(p.expected(token.Semicolon, t))
	}

	return i
}

//Any random, regular SQL.
func (p *parser) parseSQL(t token.Value, subquery, allowETLsq bool) *ast.SQL { //TODO split this into type dispatching to methods
	s := &ast.SQL{
		Tokens: make([]token.Value, 0, 64),
	}
	extend := func(ts ...token.Value) {
		s.Tokens = append(s.Tokens, ts...)
	}
	synth := func(k token.Kind) {
		s.Tokens = append(s.Tokens, token.Value{
			Kind: k,
		})
	}
	push := func() {
		s.Tokens = append(s.Tokens, t)
	}

	switch t.Canon {
	case "ANALYZE", "EXPLAIN":
		panic(p.errMsg(t, "ANALYZE and EXPLAIN are not allowed"))
	}

	//These are very simple statements that cannot contain subqueries or arguments
	//this let's us recognize them better, though still approximately.
	switch t.Canon {
	case "SAVEPOINT", "RELEASE", "ROLLBACK", "DROP", "BEGIN", "END", "VACUUM", "REINDEX":
		for t.Kind != token.Semicolon {
			push()
			t = p.next()
			switch t.Kind {
			case token.Argument, token.LParen, token.RParen:
				panic(p.unexpected(t))
			}
		}
		push() //the semicolon
		return s
	}

	//triggers have some syntax not seen elsewhere that we want to maintain and cannot be approached as we would otherwise
	//tables also need to be handled specially in the case of
	//	CREATE TABLE t (...) FROM IMPORT ...
	if t.Literal("CREATE") {
		push()
		t = p.next()
		if t.Literal("TEMP") || t.Literal("TEMPORARY") {
			push()
			t = p.next()
		}
		if t.Literal("TRIGGER") {
			for !t.Literal("BEGIN") { //spin past all the for each row etc. boilerplate
				push()
				t = p.next()
				switch t.Kind {
				case token.Argument, token.LParen, token.RParen, token.Semicolon:
					panic(p.unexpected(t))
				}
			}
			push() //push BEGIN
			for {
				t = p.next()
				if t.Literal("END") {
					push()
					t = p.expect(token.Semicolon)
					push()
					return s
				}
				//make sure we have something reasonably valid
				switch t.Canon {
				case "INSERT", "UPDATE", "DELETE", "REPLACE", "SELECT", "WITH":
				default:
					panic(p.unexpected(t))
				}
				extend(p.parseSQL(t, false, false).Tokens...)
			}
		} else if t.Literal("TABLE") {
			push()
			t = p.next()
			//IF NOT EXISTS
			if t.Literal("IF") {
				push()
				t = p.expectLit("NOT")
				push()
				t = p.expectLit("EXISTS")
				push()
			}

			//Table name is a str/lit optionally followed by a . then a str/lit
			var name []token.Value
			if t.Kind == token.Literal {
				name = append(name, t)
			} else if t.Kind == token.String {
				name = append(name, t)
			} else {
				panic(p.unexpected(t))
			}
			push()
			t = p.next()
			if t.Literal(".") {
				if t.Kind == token.Literal {
					name = append(name, t)
				} else if t.Kind == token.String {
					name = append(name, t)
				} else {
					panic(p.unexpected(t))
				}
				push()
				t = p.next()
			}

			if t.Literal("AS") {
				push()
				//table being created from select,
				//need to bail and let the regular sql recognition happen
				goto regular
			}

			//now we're in the column definitions
			if t.Kind != token.LParen {
				panic(p.unexpected(t))
			}
			push()
			depth := 1
		loop:
			for {
				t = p.next()
				switch t.Kind {
				case token.LParen:
					depth++
				case token.RParen:
					depth--
					if depth == 0 {
						push()
						break loop
					}
				case token.Semicolon, token.Argument:
					panic(p.unexpected(t))
				default:
					if t.Head(false) {
						panic(p.unexpected(t))
					}
				}
				push()
			}

			//WITHOUT ROWID
			t = p.next()
			if t.Literal("WITHOUT") {
				push()
				t = p.expectLit("ROWID")
				push()
				t = p.next()
			}

			switch {
			case t.Kind == token.Semicolon:
				push()
				return s
			case t.Literal("FROM"):
				//we do NOT push "FROM" since this is fake syntax
				s.Tokens = append(s.Tokens, token.Value{
					Kind: token.Semicolon,
				})

				//next better be import
				i := p.importStmt(p.expectLit("IMPORT"), false)
				s.Name = name
				s.Subqueries = append(s.Subqueries, i)
				return s
			}
		}
	}

regular:
	//not a trigger or simple statement, so we might have embedded import statements
	//and this could be a subquery
	//either of which means we have to keep track of parens.
	//We also desugar interpolation so we don't need to muck around with it later.
	pcount := 0
	for {
		switch t.Kind {
		case token.Semicolon:
			if !subquery {
				push()
				return s
			}
			//why is there a ; in a subquery?
			panic(p.unexpected(t))

		case token.RParen:
			pcount--
			if pcount < 0 {
				panic(p.errMsg(t, "unbalanced parentheses: ())"))
			}
			push()
			if pcount == 0 && subquery {
				return s
			}
			t = p.next()

		case token.LParen:
			pcount++
			push()
			t = p.next()

			if !t.Literal("IMPORT") {
				//quick sanity check while we're here,
				//we have a non-subquery head in a subquery position:
				//definite error.
				if t.Head(false) && !t.Head(true) {
					panic(p.unexpected(t))
				}

				//we could even end up back in this case if t = (,
				//but that is the correct behavior, handling the case (((etc.
				//for regular sql subqueries we just slurp em up
				continue
			}
			if !allowETLsq {
				panic(p.errMsg(t, "illegal IMPORT subquery"))
			}
			//parse import subquery and add to stack
			s.Subqueries = append(s.Subqueries, p.importStmt(t, true))
			// import eats closing )
			pcount--
			//add synthetic tokens to sql:
			//	placeholder marks where to inject the result the import
			//	) is to add the ) consumed by parsing import
			synth(token.Placeholder)
			synth(token.RParen)
			t = p.next()

		case token.Argument: //XXX or is it?
			if !allowETLsq {
				panic(p.errMsg(t, "cannot use @ substitutions in triggers"))
			}
			//this is a layer violation,
			//but we need to do this for handling sql subqueries in
			//extensions so might as well get it all out of the way now
			//cf. maybeSq
			ts, err := interpolate.Desugar(t)
			if err != nil {
				panic(p.mkErr(t, err))
			}
			synth(token.LParen)
			extend(ts...)
			synth(token.RParen)
			t = p.next()

		default:
			//otherwise just eat it up as probably valid sql and continue
			push()
			t = p.next()
		}
	}
}
