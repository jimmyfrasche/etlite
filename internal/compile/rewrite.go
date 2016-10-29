package compile

import (
	"strings"

	"github.com/jimmyfrasche/etlite/internal/ast"
	"github.com/jimmyfrasche/etlite/internal/internal/digital"
	"github.com/jimmyfrasche/etlite/internal/internal/errint"
	"github.com/jimmyfrasche/etlite/internal/internal/escape"
	"github.com/jimmyfrasche/etlite/internal/token"
)

func parseArg(t token.Value) (s string, isNum bool) {
	isNum = digital.String(t.Value)
	s = t.Value
	if !isNum {
		s = escape.String(s)
	}
	return
}

func (c *compiler) appendSynth(qp string) {
	c.r.Tokens = append(c.r.Tokens, token.Value{
		Kind:  token.Literal,
		Value: qp,
	})
}

func (c *compiler) rewrite(s *ast.SQL, tables []string, noArg bool) string {
	if lr, ls := len(tables), len(s.Subqueries); lr != ls {
		panic(errint.Newf("rewrite sql: expected %d replacements got %d", ls, lr))
	}

	if c.r.Tokens == nil {
		c.r.Tokens = make([]token.Value, 0, 2*len(s.Tokens))
	}

	i := 0 //count replacements to triplecheck
	for _, t := range s.Tokens {
		switch t.Kind {
		case token.Placeholder:
			if i < len(tables) {
				c.appendSynth("SELECT * FROM temp." + tables[i])
			}
			i++
		case token.Argument:
			if noArg {
				panic(errint.Newf("expected no arguments in %#v", s))
			}
			s, isNum := parseArg(t)
			q := []string{"(SELECT value FROM sys.", "env", " WHERE ", "name", "=", s, ")"}
			if isNum {
				q[1], q[3] = "args", "rowid"
			}
			c.appendSynth(strings.Join(q, ""))
		default:
			c.r.Tokens = append(c.r.Tokens, t)
		}
	}

	if i != len(tables) {
		panic(errint.Newf("expected %d replacements in subquery got %d:\n%#v", len(s.Subqueries), i, s))
	}

	if err := c.r.Print(c.buf); err != nil {
		panic(err)
	}

	c.r.Tokens = c.r.Tokens[:0]

	return c.bufStr()
}
