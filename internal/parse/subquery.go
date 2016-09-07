package parse

import (
	"strconv"

	"github.com/jimmyfrasche/etlite/internal/ast"
	"github.com/jimmyfrasche/etlite/internal/internal/runefrom"
	"github.com/jimmyfrasche/etlite/internal/parse/internal/interpolate"
	"github.com/jimmyfrasche/etlite/internal/token"
)

func (p *parser) maybeSq(t token.Value) (*ast.SQL, token.Value) {
	if t.Kind == token.LParen {
		//only called in import/display where nested import's are forbidden.
		return p.parseSQL(p.next(), true, false), p.next()
	}
	if t.Kind == token.Argument {
		ts, err := interpolate.Desugar(t)
		if err != nil {
			panic(p.mkErr(t, err))
		}
		return &ast.SQL{
			Tokens: ts,
		}, p.next()
	}
	return nil, t
}

func (p *parser) intOrSq(t token.Value, limit bool) (ast.IntOrSQL, token.Value) {
	var n *ast.SQL
	if n, t = p.maybeSq(t); n != nil {
		return n, t
	}
	if t.Kind != token.Literal {
		panic(p.expected("integer or subquery", t))
	}
	i, err := strconv.Atoi(t.Value)
	if err != nil {
		panic(p.expected("integer or subquery", t))
	}
	if i < 1 {
		//counterintuitive, but matches how sqlite handles limit
		if limit {
			return nil, p.next()
		}
		panic(p.expected("integer or subquery", t))
	}
	return &ast.Int{
		Position: t.Position,
		Value:    i,
	}, p.next()
}

func (p *parser) runeOrSq(t token.Value, what string) (ast.RuneOrSQL, token.Value) {
	var n *ast.SQL
	if n, t = p.maybeSq(t); n != nil {
		return n, t
	}
	s, ok := t.Unescape()
	if !ok {
		panic(p.expected(what, t))
	}
	r, err := runefrom.String(s)
	if err != nil {
		panic(p.mkErr(t, err))
	}
	return &ast.Rune{
		Position: t.Position,
		Value:    r,
	}, p.next()
}
