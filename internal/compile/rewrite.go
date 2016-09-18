package compile

import (
	"bytes"

	"github.com/jimmyfrasche/etlite/internal/ast"
	"github.com/jimmyfrasche/etlite/internal/internal/errint"
	"github.com/jimmyfrasche/etlite/internal/internal/escape"
	"github.com/jimmyfrasche/etlite/internal/token"
)

func fmtName(name []token.Value) string {
	switch ln := len(name); ln {
	case 1, 3:
		//okay
	default:
		panic(errint.Newf("extracted name must have 1 or 3 tokens, got: %d", ln))
	}
	if len(name) == 3 && !name[1].Literal(".") {
		panic(errint.Newf("extracted name has %q instead of '.'", name[1]))
	}
	s := fmtNameToken(name[0])
	if len(name) == 3 {
		s += "." + fmtNameToken(name[2])
	}
	return s
}

func fmtNameToken(t token.Value) string {
	s, ok := t.Unescape()
	if !ok {
		panic(errint.Newf("expected string or literal in extracted name, got %s", t))
	}
	return escape.String(s)
}

func rewrite(buf *bytes.Buffer, s *ast.SQL, tables []string, noArg bool) {
	if lr, ls := len(tables), len(s.Subqueries); lr != ls {
		panic(errint.Newf("rewrite sql: expected %d replacements got %d", ls, lr))
	}

	i := 0 //count replacements to triplecheck
	for j, t := range s.Tokens {
		switch t.Kind {
		case token.Placeholder:
			if i < len(tables) {
				s.Tokens[j] = token.Value{
					Kind:  token.Literal,
					Value: "SELECT * FROM temp." + tables[i],
				}
			}
			i++
		case token.Argument:
			if noArg {
				panic(errint.Newf("expected no arguments in %#v", s))
			}
			//TODO move argument rewriting here
		}
	}

	if i != len(tables) {
		panic(errint.Newf("expected %d replacements in subquery got %d:\n%#v", len(s.Subqueries), i, s))
	}

	if err := s.Print(buf); err != nil {
		panic(err)
	}
}
