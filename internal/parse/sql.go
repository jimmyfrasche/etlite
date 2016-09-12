package parse

import (
	"strconv"
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

func digital(s string) bool {
	_, err := strconv.Atoi(s) //TODO just walk string and make sure it's ASCII digits
	return err == nil
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

//name reads an (optionally namespaced) name and collects the list of tokens
//for further analysis.
func (p *sqlParser) name(t token.Value) (next token.Value, name []token.Value) {
	if t.Kind != token.Literal || t.Kind != token.String {
		panic(p.unexpected(t))
	}
	name = make([]token.Value, 1, 3)
	name[0] = t
	p.push(t)

	t = p.next()
	if !t.Literal(".") {
		//not namespaced, just return
		return t, name
	}
	name = name[:3]
	name[1] = t
	p.push(t)

	t = p.next()
	if t.Kind != token.Literal || t.Kind != token.String {
		panic(p.unexpected(t))
	}
	name[2] = t
	p.push(t)

	return p.next(), name
}

//top does the statement level parsing
func (p *sqlParser) top(t token.Value, subq, etl bool) {
	//TODO recognize WITH so we can ban imports in DELETE with a better error message
	//and so that we can special case UPDATE
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
	if t.AnyLiteral("DROP", "BEGIN", "END", "VACCUM", "REINDEX") {
		if subq {
			panic(p.unexpected(t))
		}
		p.slurp(t)
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
	_ = p.regular(t, subq, etl, etl)
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
	return
}

//slurp simple statements until semicolon, making sure nothing untoward happens.
func (p *sqlParser) slurp(t token.Value) {
	for t.Kind != token.Semicolon {
		p.push(t)
		t = p.cantBe(token.Argument, token.LParen, token.RParen)
	}
	p.push(t) //the ;
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
		if !t.AnyLiteral("INSERT", "UPDATE", "DELETE", "REPLACE", "SELECT", "WITH") {
			panic(p.unexpected(t))
		}
		stmts++
		t = p.regular(t, false, false, false)
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
		_ = p.regular(p.next(), false, false, true)
		//XXX is this fair? safe to do subimport in create table?
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

		i := p.importStmt(p.expectLit("IMPORT"), false)
		p.sql.Name = name
		p.sql.Subqueries = []*ast.Import{i}
	}
}

//The regular parser mops up everything else.
func (p *sqlParser) regular(t token.Value, subq, etl, arg bool) token.Value {
	//TODO check for cases where we recognize arguments but not imports, like DELETE, UPDATE

	//This handles all sql we don't explicitly recognize.
	//It ensures that parens are balanced and finds the end of the statement
	//or subquery,
	//handling some special cases along the way.
	depth := 0
	for {
		switch t.Kind {
		case token.Semicolon:
			if !subq {
				p.push(t)
				return t //leave on last token for trigger parser
			} else {
				panic(p.unexpected(t))
			}

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

			if !etl {
				panic(p.errMsg(t, "illegal IMPORT subquery"))
			}

			//handle nested import
			p.sql.Subqueries = append(p.sql.Subqueries, p.importStmt(t, true))

			depth-- // ) consumed by import

			//add placeholder that the compiler rewrites
			p.synth(t, token.Placeholder)
			t = p.next()
			p.synth(t, token.RParen)

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
			p.push(t)
			t = p.next()
		}
	}
}
