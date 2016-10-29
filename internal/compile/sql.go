package compile

import (
	"strconv"
	"strings"

	"github.com/jimmyfrasche/etlite/internal/ast"
	"github.com/jimmyfrasche/etlite/internal/internal/errint"
	"github.com/jimmyfrasche/etlite/internal/internal/errusr"
	"github.com/jimmyfrasche/etlite/internal/virt"
)

func (c *compiler) compileSQL(s *ast.SQL) {
	switch s.Kind {
	default:
		panic(errint.Newf("got unknown or invalid sql kind %d", s.Kind))
	case ast.Savepoint, ast.Release, ast.BeginTransaction, ast.Commit:
		if ls := len(s.Subqueries); ls != 0 {
			panic(errint.Newf("%s cannot have etl subqueries, found %d", s.Kind, ls))
		}
		c.compileTransactor(s)
		return
	case ast.CreateTableFrom, ast.InsertUsing:
		if ls := len(s.Subqueries); ls != 1 {
			panic(errint.Newf("%s must have exactly 1 etl subquery, found %d", s.Kind, ls))
		}
	case ast.Exec, ast.Query:
		//can have any number
	}

	switch s.Kind {
	case ast.CreateTableFrom, ast.InsertUsing:
		nm := s.Name.String()
		if s.Kind == ast.CreateTableFrom {
			c.compileCreateTableAsImport(nm, s)
		} else {
			c.compileInsertUsing(nm, s)
		}
		return
	}

	var tables []string
	if len(s.Subqueries) > 0 {
		c.push(virt.Savepoint())

		//compile the imports
		tables = make([]string, len(s.Subqueries))
		for i, imp := range s.Subqueries {
			tables[i] = "[" + strconv.Itoa(i) + "]"
			c.compileSubImport(imp, tables[i])
		}
	}
	c.push(virt.ErrPos(s.Pos()))

	q := c.rewrite(s, tables, false)

	switch s.Kind {
	case ast.Exec:
		c.push(virt.Exec(q))
	case ast.Query:
		c.push(virt.Query(q))
	}

	if len(tables) > 0 {
		c.push(virt.DropTempTables(tables))
		c.push(virt.Release())
	}
	return
}

func (c *compiler) compileTransactor(s *ast.SQL) {
	if s.Name.HasSchema() {
		panic(errint.Newf("impossible savepoint name %#v", s.Name))
	}
	if (s.Kind == ast.Savepoint || s.Kind == ast.Release) && s.Name.Empty() {
		panic(errint.New("no savepoint name provided by parser"))
	}

	q := c.rewrite(s, nil, false)

	//normalize name
	name := strings.ToLower(s.Name.Object())

	c.push(virt.ErrPos(s.Pos()))
	//make sure these stack correctly
	switch s.Kind {
	default:
		panic(errint.Newf("no valid transaction type, got %d", s.Kind))

	case ast.BeginTransaction:
		if err := c.stack.Begin(); err != nil {
			panic(errusr.Wrap(s.Pos(), err))
		}
		c.push(virt.BeginTransaction(q))

	case ast.Commit:
		if err := c.stack.End(); err != nil {
			panic(errusr.Wrap(s.Pos(), err))
		}
		c.push(virt.CommitTransaction(q))

	case ast.Savepoint:
		c.stack.Savepoint(name)
		c.push(virt.UserSavepoint(name, q))

	case ast.Release:
		if err := c.stack.Release(name); err != nil {
			panic(errusr.Wrap(s.Pos(), err))
		}
		c.push(virt.UserRelease(name, q))
	}
}
