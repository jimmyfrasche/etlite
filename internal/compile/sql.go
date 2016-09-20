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
	case ast.CreateTableFrom, ast.CreateTableAs, ast.InsertFrom:
		if ls := len(s.Subqueries); ls != 1 {
			panic(errint.Newf("%s must have exactly 1 etl subquery, found %d", s.Kind, ls))
		}
	case ast.Exec, ast.Query:
		//can have any number
	}

	switch s.Kind {
	case ast.CreateTableFrom, ast.InsertFrom:
		nm := fmtName(s.Name)
		i := s.Subqueries[0]
		rewrite(c.buf, s, nil, false)
		ddl := c.bufStr()
		if s.Kind == ast.CreateTableFrom {
			//TODO when we factor out insert stuff push create table then custom insert importer
			c.compileCreateTableAsImport(nm, ddl, i)
		} else {
			//TODO this
			panic("unimplemented")
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
			c.push(virt.ErrPos(s.Pos()))
		}
	}

	rewrite(c.buf, s, tables, true)
	q := c.bufStr()

	switch s.Kind {
	case ast.Exec:
		c.push(virt.Exec(q))
	case ast.CreateTableAs:
		//need to release the savepoint before the query
		//so as to not interfere with the creation of the table
		c.push(virt.Release())
		fallthrough
	case ast.Query:
		c.push(virt.Query(q))
	}

	if len(tables) > 0 {
		c.push(virt.DropTempTables(tables))
		if s.Kind != ast.CreateTableAs {
			c.push(virt.Release())
		}
	}
	return
}

func (c *compiler) compileTransactor(s *ast.SQL) {
	if len(s.Name) > 1 {
		panic(errint.Newf("impossible savepoint name %#v", s.Name))
	}
	if (s.Kind == ast.Savepoint || s.Kind == ast.Release) && len(s.Name) == 0 {
		panic(errint.New("no savepoint name provided by parser"))
	}

	rewrite(c.buf, s, nil, false)
	q := c.bufStr()

	//normalize name
	name := strings.ToLower(fmtName(s.Name))

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
