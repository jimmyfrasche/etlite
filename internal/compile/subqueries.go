package compile

import (
	"fmt"
	"strconv"

	"github.com/jimmyfrasche/etlite/internal/ast"
	"github.com/jimmyfrasche/etlite/internal/errint"
	"github.com/jimmyfrasche/etlite/internal/errusr"
	"github.com/jimmyfrasche/etlite/internal/null"
	"github.com/jimmyfrasche/etlite/internal/runefrom"
)

func handleStr(s *string) (interface{}, error) {
	if s == nil {
		return "", nil
	}
	return *s, nil
}

func (c *compiler) strOrSub(f ast.Node) {
	switch f := f.(type) {
	default:
		panic(errint.Newf("expected string or subquery, got %T", f))
	case *ast.String:
		c.pushpush(f.Value)
	case *ast.SQL:
		c.compileSQL(f, handleStr)
	}
}

func (c *compiler) mandatoryStrOrSub(f ast.Node, ctx string) {
	switch f := f.(type) {
	default:
		panic(errint.Newf("expected string or subquery, got %T", f))
	case *ast.String:
		if f.Value == "" {
			panic(errusr.Wrap(f.Pos(), fmt.Errorf("expected string for %s but got empty string", ctx)))
		}
		c.pushpush(f.Value)
	case *ast.SQL:
		p := f.Pos()
		c.compileSQL(f, func(s *string) (interface{}, error) {
			if s == nil || *s == "" {
				return nil, errusr.Wrap(p, fmt.Errorf("expected non-empty string for %s but got empty string", ctx))
			}
			return *s, nil
		})
	}
}

func handleNull(s *string) (interface{}, error) {
	if s == nil {
		return null.Encoding(""), nil
	}
	return null.Encoding(*s), nil
}

func (c *compiler) nullOrSub(f ast.Node, ifnil null.Encoding) {
	switch f := f.(type) {
	default:
		panic(errint.Newf("expected string or subquery, got %T", f))
	case nil:
		c.pushpush(ifnil)
	case *ast.Null:
		c.pushpush(f.Value)
	case *ast.SQL:
		c.compileSQL(f, handleNull)
	}
}

func (c *compiler) intOrSub(f ast.Node, ifnil int) {
	switch f := f.(type) {
	default:
		panic(errint.Newf("expected string or subquery, got %T", f))
	case nil:
		c.pushpush(ifnil)
	case *ast.Int:
		c.pushpush(f.Value)
	case *ast.SQL:
		p := f.Pos()
		c.compileSQL(f, func(s *string) (interface{}, error) {
			if s == nil {
				return 0, nil
			}
			i, err := strconv.Atoi(*s)
			if err != nil {
				return nil, errusr.Wrap(p, err)
			}
			return i, nil
		})
	}
}

func (c *compiler) runeOrSub(f ast.Node, ifnil rune) {
	switch f := f.(type) {
	default:
		panic(errint.Newf("expected string or subquery, got %T", f))
	case nil:
		c.pushpush(ifnil)
	case *ast.Rune:
		c.pushpush(f.Value)
	case *ast.SQL:
		p := f.Pos()
		c.compileSQL(f, func(s *string) (interface{}, error) {
			if s == nil {
				return 0, nil //XXX need to make sure that NULL → default rune
			}
			r, err := runefrom.String(*s)
			if err != nil {
				return nil, errusr.Wrap(p, err)
			}
			return r, nil
		})
	}
}

func (c *compiler) boolOrSub(f ast.Node, ifnil bool) {
	switch f := f.(type) {
	default:
		panic(errint.Newf("expected string or subquery, got %T", f))
	case nil:
		c.pushpush(ifnil)
	case *ast.Bool:
		c.pushpush(f.Value)
	case *ast.SQL:
		p := f.Pos()
		c.compileSQL(f, func(s *string) (interface{}, error) {
			if s == nil {
				return false, nil
			}
			i, err := strconv.Atoi(*s)
			if err != nil {
				return nil, errusr.Wrap(p, err)
			}
			return i != 0, nil
		})
	}
}
