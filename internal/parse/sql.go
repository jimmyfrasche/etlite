package parse

import (
	"strings"

	"github.com/jimmyfrasche/etlite/internal/ast"
	"github.com/jimmyfrasche/etlite/internal/parse/internal/interpolate"
	"github.com/jimmyfrasche/etlite/internal/token"
)

type sqlParser struct {
	*parser
	sql *ast.SQL
}

func newSqlParser(p *parser) *sqlParser {
	return &sqlParser{
		parser: p,
		sql: &ast.SQL{
			Tokens: make([]token.Value, 0, 256),
		},
	}
}

func (p *sqlParser) extend(ts ...token.Value) {
	p.sql.Tokens = append(p.sql.Tokens, ts...)
}

func (p *sqlParser) push(t token.Value) {
	p.extend(t)
}

//synthesize a token and push it.
//Takes the current token as we use that as a rough proxy
//for the position in error messages.
func (p *sqlParser) synth(t token.Value, k token.Kind) {
	p.push(token.Value{
		Position: t.Position,
		Kind:     k,
	})
}

func (p *sqlParser) subImport(t token.Value, subq, etl bool) token.Value {
	if !etl {
		panic(p.errMsg(t, "illegal IMPORT subquery"))
	}

	//magic token for the compiler to rewrite
	p.synth(t, token.Placeholder)
	var n ast.Node
	n, t = p.importStmt(t, subq, true, p.sql)
	p.sql.Subqueries = append(p.sql.Subqueries, n.(*ast.Import))
	return t
}

func digital(s string) bool {
	for i := 0; i < len(s); i++ {
		if b := s[i]; b < '0' || b > '9' {
			return false
		}
	}
	return true
}

func (p *sqlParser) chkDigTmp(name []token.Value) {
	if len(name) == 3 {
		s, _ := name[0].Unescape()
		if strings.ToUpper(s) == "TEMP" {
			s, _ = name[2].Unescape()
			if digital(s) {
				panic(p.errMsg(name[0], "digital temporary table names are reserved by etlite"))
			}
		}
	}
}

func (p *sqlParser) tmpCheck(t token.Value) token.Value {
	t, name := p.name(t)
	p.chkDigTmp(name)
	p.extend(name...)
	return t
}

func (p *sqlParser) chkSysReserved(name []token.Value) {
	if len(name) == 3 {
		s, _ := name[0].Unescape()
		//this fails if it relies on object resolution but catches some misuse.
		//TODO could insert a flag in the AST to check sys names exist if length 1 and name[0] ∈ {args, env}?
		if strings.ToUpper(s) == "SYS" {
			s, _ = name[2].Unescape()
			s = strings.ToUpper(s)
			switch s {
			case "ARGS", "ENV":
				panic(p.errMsg(name[0], "sys.args and sys.env are reserved by etlite"))
			}
		}
	}
}

func (p *sqlParser) tmpOrSysCheck(t token.Value) token.Value {
	t, name := p.name(t)
	p.chkDigTmp(name)
	p.chkSysReserved(name)
	p.extend(name...)
	return t
}

//maybeRun eats possible runs of lits such as "IF", "NOT", "EXISTS".
func (p *sqlParser) maybeRun(t token.Value, lits ...string) token.Value {
	if !t.Literal(lits[0]) {
		return t
	}
	p.push(t) //lits[0]
	for _, lit := range lits[1:] {
		t = p.expectLit(lit)
		p.push(t)
	}
	return p.next()
}

//top does the statement level parsing
func (p *sqlParser) top(t token.Value, subq, etl bool) {
	if t.Kind != token.Literal {
		panic(p.unexpected(t))
	}

	//Forbidden statements
	if t.AnyLiteral("ANALYZE", "EXPLAIN", "ROLLBACK") {
		panic(p.errMsg(t, "ANALYZE and EXPLAIN and ROLLBACK are not allowed"))
	}

	//Savepoint and release need a check to see they don't step on
	//reserved savepoints
	if t.AnyLiteral("SAVEPOINT", "RELEASE") {
		if subq {
			panic(p.unexpected(t))
		}
		p.saverelease(t)
		return
	}

	//These are very simple and we just need to make sure nothing's obviously wrong
	//while seeking ;
	if t.AnyLiteral("BEGIN", "END", "VACCUM", "REINDEX") {
		if subq {
			panic(p.unexpected(t))
		}
		p.slurp(t)
		return
	}

	//for these two we validate no reserved names are injured.
	if t.Literal("ALTER") {
		if subq {
			panic(p.unexpected(t))
		}
		p.alterTable(t)
		return
	}
	if t.Literal("DROP") {
		if subq {
			panic(p.unexpected(t))
		}
		p.drop(t)
		return
	}

	if t.Literal("CREATE") {
		if subq {
			panic(p.unexpected(t))
		}
		p.push(t)
		t = p.next()
		temp := false
		if t.AnyLiteral("TEMP", "TEMPORARY") {
			p.push(t)
			t = p.next()
			temp = true
		}
		if t.Literal("TRIGGER") {
			p.trigger(t)
			return
		} else if t.Literal("TABLE") {
			p.table(t, temp)
			return
		}
	}

	//the stutter is not an accident: except for some special cases these are the same
	switch t.Canon {
	case "INSERT", "REPLACE":
		_ = p.insert(t, subq, etl, etl)
	case "DELETE":
		_ = p.delete(t, subq, etl, etl)
	case "UPDATE":
		_ = p.update(t, subq, etl, etl)
	case "WITH":
		_ = p.with(t, subq, etl, etl)
	default:
		_ = p.regular(t, 0, subq, etl, etl)
	}
}

func (p *sqlParser) alterTable(t token.Value) {
	p.push(t)
	t = p.expectLit("TABLE")
	p.push(t)
	t = p.tmpOrSysCheck(t)
	switch t.Canon {
	default:
		panic(p.unexpected(t))
	case "RENAME":
		p.push(t)
		t = p.expectLit("TO")
		p.push(t)
		t = p.tmpOrSysCheck(t)
		if t.Kind != token.Semicolon {
			panic(p.unexpected(t))
		}
		p.push(t)
	case "ADD":
		p.push(t)
		_ = p.regular(t, 0, false, false, false)
	}
}

func (p *sqlParser) drop(t token.Value) {
	p.push(t)
	t = p.expect(token.Literal)
	if !t.Literal("TABLE") {
		p.slurp(t)
	}
	p.push(t)
	t = p.maybeRun(p.next(), "IF", "EXISTS")
	t = p.tmpOrSysCheck(t)
	if t.Kind != token.Semicolon {
		panic(p.unexpected(t))
	}
	p.push(t)
}

func (p *sqlParser) saverelease(t token.Value) {
	p.push(t)
	t = p.next()
	s, ok := t.Unescape()
	if !ok {
		panic(p.unexpected(t))
	}
	if digital(s) {
		panic(p.errMsg(t, "digital savepoint names are reserved by etlite"))
	}
	p.push(t)
	p.expect(token.Semicolon)
}

//slurp simple statements until semicolon, making sure nothing untoward happens.
func (p *sqlParser) slurp(t token.Value) {
	for t.Kind != token.Semicolon {
		p.push(t)
		t = p.cantBe(token.Argument, token.LParen, token.RParen)
	}
	p.push(t) //the ;
}

func (p *sqlParser) with(t token.Value, subq, etl, arg bool) token.Value {
	p.push(t)
	first := true
	for {
		t = p.expectLitOrStr() // name or possibly RECURSIVE if first time through

		rec := false
		if t.Literal("RECURSIVE") {
			if !first {
				panic(p.unexpected(t))
			}
			p.push(t)
			t = p.expectLitOrStr()
			rec = true
		}
		first = false

		p.push(t)
		t = p.next()
		if t.Kind == token.LParen { //optional column names
			p.push(t)
			for t.Kind != token.RParen {
				t = p.next()
				p.push(t)
			}
			t = p.next()
		}
		if t.Literal("AS") {
			p.push(t)
		} else {
			panic(p.unexpected(t))
		}

		//The table expression
		t = p.expect(token.LParen)
		p.push(t)
		t = p.expect(token.Literal)
		if t.Literal("WITH") {
			if rec {
				panic(p.unexpected(t))
			}
			t = p.with(t, true, etl, arg)
		} else {
			if !rec {
				t = p.regular(t, 1, true, etl, arg)
			} else {
				//XXX would be safe if it were import union [noimports]
				//XXX could special case that
				t = p.regular(t, 1, true, false, arg)
			}
		}

		t = p.expect(token.Literal)
		if !t.Literal(",") {
			break
		}
	}

	//XXX could allow etl IFF if part of compound operator?
	switch t.Canon {
	default:
		panic(p.unexpected(t))
	case "INSERT", "REPLACE":
		return p.insert(t, subq, etl, arg)
	case "UPDATE":
		//probably too complicated to do any special checking here but should at least check table name
		return p.update(t, subq, etl, arg)
	case "DELETE":
		return p.delete(t, subq, etl, arg)
	case "SELECT":
		depth := 0
		if subq {
			depth++
		}
		return p.regular(t, depth, subq, etl, arg)
	}
}

func (p *sqlParser) delete(t token.Value, subq, etl, arg bool) token.Value {
	if subq {
		panic(p.unexpected(t))
	}
	p.push(t)
	t = p.expectLit("FROM")
	p.push(t)
	t = p.tmpCheck(p.next())
	return p.regular(t, 0, subq, etl, arg)
}

func (p *sqlParser) update(t token.Value, subq, etl, arg bool) token.Value {
	if subq {
		panic(p.unexpected(t))
	}
	p.push(t)
	t = p.expectLitOrStr()
	if t.Literal("OR") {
		p.push(t)
		t = p.expect(token.Literal) //ROLLBACK, etc.
		p.push(t)
		t = p.next()
	}
	t = p.tmpCheck(t)
	if !t.Literal("SET") {
		panic(p.unexpected(t))
	}
	return p.regular(t, 0, subq, etl, arg)
}

func (p *sqlParser) insert(t token.Value, subq, etl, arg bool) token.Value {
	if subq {
		panic(p.unexpected(t))
	}
	replace := t.Literal("REPLACE")
	p.push(t)

	t = p.expect(token.Literal)
	if t.Literal("OR") {
		if replace {
			panic(p.unexpected(t))
		}
		p.push(t)
		t = p.expect(token.Literal) //ROLLBACK, etc.
		p.push(t)
		t = p.next()
	}
	if !t.Literal("INTO") {
		panic(p.unexpected(t))
	}
	p.push(t)

	t = p.tmpCheck(p.next())
	if t.Kind != token.LParen {
		panic(p.unexpected(t))
	}
	p.push(t)

loop:
	for {
		t = p.expectLitOrStr()
		p.push(t)
		t = p.next()
		switch {
		default:
			panic(p.unexpected(t))
		case t.Literal(","):
			p.push(t)
		case t.Kind == token.RParen:
			p.push(t)
			break loop
		}
	}

	t = p.expect(token.Literal)
	switch t.Canon {
	default:
		panic(p.unexpected(t))
	case "DEFAULT": //not to be confused with the above
		p.push(t)
		t = p.expectLit("VALUES")
		p.push(t)
		t = p.expect(token.Semicolon)
		p.push(t)
		return t //this isn't used anywhere, but needed for symmetry
	case "IMPORT":
		//TODO we could add a special FROM IMPORT here, with a little work
		panic(p.errMsg(t, "INSERT ... IMPORT is currently unsupported"))
	case "WITH":
		return p.with(t, subq, etl, arg)
	case "VALUES", "SELECT":
		return p.regular(t, 0, subq, etl, arg)
	}
}

//trigger handles triggers which have a special structure
//requiring them to be handled separately.
func (p *sqlParser) trigger(t token.Value) {
	//skip till begin
	for !t.Literal("BEGIN") {
		p.push(t)
		t = p.cantBe(token.Argument, token.LParen, token.RParen, token.Semicolon)
	}

	p.push(t) //BEGIN

	stmts := 0
	for {
		t = p.next()
		//end at the END
		if t.Literal("END") {
			if stmts == 0 {
				panic(p.errMsg(t, "trigger has no actions"))
			}
			p.push(t)
			t = p.expect(token.Semicolon)
			p.push(t)
			return
		}

		//otherwise, make sure we have a valid head
		switch t.Canon {
		default:
			panic(p.unexpected(t))
		case "INSERT", "REPLACE":
			t = p.insert(t, false, false, false)
		case "UPDATE":
			t = p.update(t, false, false, false)
		case "DELETE":
			t = p.delete(t, false, false, false)
		case "SELECT":
			t = p.regular(t, 0, false, false, false)
		}
		stmts++
	}
}

//table parses create table statements to handle CREATE TABLE FROM special form.
func (p *sqlParser) table(t token.Value, temp bool) {
	p.push(t)
	t = p.maybeRun(p.next(), "IF", "NOT", "EXISTS")

	//get the name in case this is CREATE TABLE FROM
	//and validate that it's not reserved.
	var name []token.Value
	t, name = p.name(t)
	p.extend(name...)
	if len(name) == 3 {
		s, _ := name[0].Unescape()
		if strings.ToUpper(s) == "TEMP" {
			if temp {
				panic(p.unexpected(name[0]))
			}
			temp = true
		}
	}
	if temp {
		last := name[len(name)-1]
		s, _ := last.Unescape()
		if digital(s) {
			panic(p.errMsg(last, "digital temporary table names are reserved by etlite"))
		}
	}

	if t.Literal("AS") {
		p.push(t)
		_ = p.regular(p.next(), 0, false, false, true)
		//XXX is this fair? safe to do subimport in create table?
		//XXX it wouldn't respect the usual rules and if it failed
		//XXX the table wouldn't exist, unlike with import statement. Must think.
		//XXX note in ast, special case compiler to treat like normal subquery import
		//XXX but release savepoint before create and drop tables after.
		return
	}

	//we're at the column definitions now, just need to make sure we handle
	//(()) and catch anything that's obviously wrong.
	if t.Kind != token.LParen {
		panic(p.unexpected(t))
	}
	p.push(t)
	depth := 1
loop:
	for {
		t = p.cantBe(token.Semicolon, token.Argument)
		switch t.Kind {
		case token.LParen:
			depth++
		case token.RParen:
			depth--
			if depth == 0 {
				p.push(t)
				break loop
			}
		case token.Literal:
			if t.Head(false) {
				panic(p.unexpected(t))
			}
		}
		p.push(t)
	}

	t = p.maybeRun(t, "WITHOUT", "ROWID")
	switch {
	case t.Kind == token.Semicolon: //done
		p.push(t)
		return
	case t.Literal("FROM"):
		//we do not push "FROM" since this is fake syntax
		//instead, we insert a synthetic semicolon
		p.synth(t, token.Semicolon)

		//we can't use subImport since this is a special case
		//TODO will be less of a special case when INSERT FROM is implemented
		var n ast.Node
		n, t = p.importStmt(p.expectLit("IMPORT"), false, false, nil)
		p.sql.Name = name
		p.sql.Subqueries = []*ast.Import{n.(*ast.Import)}
	}
}

//The regular parser mops up everything else.
func (p *sqlParser) regular(t token.Value, depth int, subq, etl, arg bool) token.Value {
	//This handles all sql we don't explicitly recognize.
	//It ensures that parens are balanced and finds the end of the statement
	//or subquery,
	//handling some special cases along the way.
	for {
		switch t.Kind {
		case token.Semicolon:
			if subq {
				panic(p.unexpected(t))
			}
			p.push(t)
			return t //leave on last token for trigger parser

		case token.RParen:
			depth--
			if depth < 0 {
				panic(p.errMsg(t, "unbalanced parentheses: ())"))
			}
			p.push(t)
			if depth == 0 && subq {
				return t //leave on last token for trigger parser
			}
			t = p.next()

		case token.LParen:
			depth++
			p.push(t)
			t = p.next()
			if t.Kind != token.Literal {
				continue
			}

			switch t.Canon {
			case "WITH":
				t = p.with(t, true, etl, arg)

			case "IMPORT":
				//t is )
				t = p.subImport(t, true, etl)
			}

		case token.Argument:
			if !arg {
				panic(p.errMsg(t, "illegal @ substitution"))
			}
			//TODO move all sql rewriting to compiler, and just fallthrough

			ts, err := interpolate.Desugar(t)
			if err != nil {
				panic(p.mkErr(t, err))
			}
			p.synth(t, token.LParen)
			p.extend(ts...)
			p.synth(t, token.RParen)
			t = p.next()

		default:
			//check for compound operators
			if t.AnyLiteral("UNION", "INTERSECT", "EXCEPT") {
				p.push(t)
				lastWasU := t.Literal("UNION")
				t = p.expect(token.Literal)
				if t.Literal("ALL") {
					if !lastWasU {
						panic(p.unexpected(t))
					}
					p.push(t)
					t = p.expect(token.Literal)
				}

				switch t.Canon {
				default:
					panic(p.unexpected(t))

				case "IMPORT":
					t = p.subImport(t, subq, etl)
					continue //need to recognize t next go round

				case "SELECT", "VALUES":
					//just continue along as we were
				}
			}
			p.push(t)
			t = p.next()
		}
	}
}
