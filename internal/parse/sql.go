package parse

import (
	"strings"

	"github.com/jimmyfrasche/etlite/internal/ast"
	"github.com/jimmyfrasche/etlite/internal/internal/digital"
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

//specialSubImport is used only by CreateTableFrom and InsertFrom
func (p *sqlParser) specialSubImport(t token.Value) token.Value {
	var n ast.Node
	n, t = p.importStmt(t, false, false, nil)
	p.sql.Subqueries = []*ast.Import{n.(*ast.Import)}
	return t
}

func (p *sqlParser) name(t token.Value) (token.Value, []token.Value) {
	t, name := p.parser.name(t)
	p.sql.Name = name
	return t, name
}

func (p *sqlParser) chkDigTmp(name []token.Value) {
	if len(name) == 3 {
		s, _ := name[0].Unescape()
		if strings.ToUpper(s) == "TEMP" {
			s, _ = name[2].Unescape()
			if digital.String(s) {
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

	if t.Literal("SAVEPOINT") {
		if subq {
			panic(p.unexpected(t))
		}
		p.savepoint(t)
		return
	}

	if t.Literal("RELEASE") {
		if subq {
			panic(p.unexpected(t))
		}
		p.release(t)
		return
	}

	if t.Literal("BEGIN") {
		if subq {
			panic(p.unexpected(t))
		}
		p.beginTransaction(t)
		return
	}

	if t.AnyLiteral("END", "COMMIT") {
		if subq {
			panic(p.unexpected(t))
		}
		p.endTransaction(t)
		return
	}

	//These are very simple and we just need to make sure nothing's obviously wrong
	//while seeking ;
	if t.AnyLiteral("VACCUM", "REINDEX", "PRAGMA") {
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
		p.sql.Kind = ast.Exec //overridden as necessary
		p.push(t)
		t = p.next()
		temp := false
		if t.AnyLiteral("TEMP", "TEMPORARY") {
			p.push(t)
			t = p.next()
			temp = true
		}
		switch t.Canon {
		case "TRIGGER":
			p.trigger(t)
		case "TABLE":
			p.table(t, temp)
		case "VIRTUAL", "VIEW":
			_ = p.regular(t, 0, false, false, false)
		case "UNIQUE", "INDEX":
			if temp {
				panic(p.unexpected(t))
			}
			_ = p.regular(t, 0, false, false, false)
		}
		return
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
		p.sql.Kind = ast.Query
		_ = p.regular(t, 0, subq, etl, etl)
	}
}

func (p *sqlParser) alterTable(t token.Value) {
	p.sql.Kind = ast.Exec
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
	p.sql.Kind = ast.Exec
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

func (p *sqlParser) savepoint(t token.Value) {
	p.sql.Kind = ast.Savepoint
	p.push(t)
	t = p.next()
	s, ok := t.Unescape()
	if !ok {
		panic(p.unexpected(t))
	}
	if digital.String(s) {
		panic(p.errMsg(t, "digital savepoint names are reserved by etlite"))
	}
	p.sql.Name = []token.Value{t}
	p.push(t)
	p.expect(token.Semicolon)
}

func (p *sqlParser) release(t token.Value) {
	p.sql.Kind = ast.Release
	p.push(t)
	t = p.expectLitOrStr()
	if t.Literal("SAVEPOINT") {
		p.push(t)
		t = p.expectLitOrStr()
	}
	p.sql.Name = []token.Value{t}
	p.push(t)
	p.expect(token.Semicolon)
}

func (p *sqlParser) beginTransaction(t token.Value) {
	p.sql.Kind = ast.BeginTransaction
	p.push(t)
	t = p.next()
	if t.AnyLiteral("DEFERRED", "IMMEDIATE", "EXCLUSIVE") {
		p.push(t)
		t = p.next()
	}
	if !t.Literal("TRANSACTION") {
		panic(p.unexpected(t))
	}
	p.push(t)
	t = p.expect(token.Semicolon)
	p.push(t)
}

func (p *sqlParser) endTransaction(t token.Value) {
	p.sql.Kind = ast.Commit
	p.push(t)
	t = p.expectLit("TRANSACTION")
	p.push(t)
	t = p.expect(token.Semicolon)
	p.push(t)
}

//slurp simple statements until semicolon, making sure nothing untoward happens.
func (p *sqlParser) slurp(t token.Value) {
	p.sql.Kind = ast.Exec
	if t.Literal("PRAGMA") {
		p.sql.Kind = ast.Query //some pragmas output
	}
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

	//XXX could allow etl IFF part of compound operator?
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
	case "SELECT", "VALUES":
		depth := 0
		if subq {
			depth++
		}
		return p.regular(t, depth, subq, etl, arg)
	}
}

func (p *sqlParser) delete(t token.Value, subq, etl, arg bool) token.Value {
	p.sql.Kind = ast.Exec
	if subq {
		panic(p.unexpected(t))
	}
	p.push(t)
	t = p.expectLit("FROM")
	p.push(t)
	t = p.tmpCheck(p.next())
	return p.regular(t, 0, subq, etl, arg)
}

func (p *sqlParser) conflictMethod(t token.Value) {
	if !t.AnyLiteral("REPLACE", "ROLLBACK", "ABORT", "FAIL", "IGNORE") {
		panic(p.unexpected(t))
	}
}

func (p *sqlParser) update(t token.Value, subq, etl, arg bool) token.Value {
	p.sql.Kind = ast.Exec
	if subq {
		panic(p.unexpected(t))
	}
	p.push(t)
	t = p.expectLitOrStr()
	if t.Literal("OR") {
		p.push(t)
		t = p.expect(token.Literal) //ROLLBACK, etc.
		p.conflictMethod(t)
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
	p.sql.Kind = ast.Exec
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
		p.conflictMethod(t)
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
		p.sql.Cols = append(p.sql.Cols, t)
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
	case "FROM":
		p.sql.Kind = ast.InsertFrom
		//we don't consume from and leave the ast for the insert hanging,
		//so that the compiler can add the (?, ?, ..., ?); for however many
		//placeholders are needed.
		return p.specialSubImport(p.expectLit("IMPORT"))
	case "WITH":
		return p.with(t, subq, etl, arg)
	case "VALUES", "SELECT":
		return p.regular(t, 0, subq, etl, arg)
	}
}

//trigger handles triggers which have a special structure
//requiring them to be handled separately.
func (p *sqlParser) trigger(t token.Value) {
	p.sql.Kind = ast.Exec
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
	p.sql.Kind = ast.Exec
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
		if digital.String(s) {
			panic(p.errMsg(last, "digital temporary table names are reserved by etlite"))
		}
	}
	p.sql.Name = name

	if t.Literal("AS") {
		p.push(t)
		_ = p.regular(p.next(), 0, false, true, true)
		return
	}

	//we're at the column definitions now, just need to make sure we handle
	//(()) and catch anything that's obviously wrong.
	if t.Kind != token.LParen {
		panic(p.unexpected(t))
	}
	p.push(t)
	for t.Kind != token.RParen {
		t = p.colDef()
		p.push(t)
	}

	t = p.maybeRun(t, "WITHOUT", "ROWID")
	switch {
	case t.Kind == token.Semicolon: //done
		p.push(t)
		return
	case t.Literal("FROM"):
		p.sql.Kind = ast.CreateTableFrom
		//we do not push "FROM" since this is fake syntax
		//instead, we insert a synthetic semicolon
		p.synth(t, token.Semicolon)

		//we can't use subImport since this is a special case
		t = p.specialSubImport(p.expectLit("IMPORT"))
	}
}

func (p *sqlParser) colDef() token.Value {
	t := p.expectLitOrStr()
	p.push(t)
	//don't store table constraints, only interested in column names
	if !t.AnyLiteral("CONSTRAINT", "PRIMARY", "UNIQUE", "CHECK", "FOREIGN") {
		p.sql.Cols = append(p.sql.Cols, t)
	}
	depth := 0
	for {
		t = p.cantBe(token.Semicolon, token.Argument)
		switch t.Kind {
		case token.LParen:
			depth++
		case token.RParen:
			depth--
			if depth == -1 { //ran out of our () and hit ) from table def
				return t
			}
		case token.Literal:
			if depth == 0 && t.Canon == "," {
				return t
			}
		}
		p.push(t)
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
			fallthrough

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
