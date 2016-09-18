package compile

import (
	"strconv"

	"github.com/jimmyfrasche/etlite/internal/ast"
	"github.com/jimmyfrasche/etlite/internal/internal/errint"
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
	case ast.CreateTableFrom, ast.InsertFrom: //TODO should collect columns in parser
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
	panic("TODO")
	rewrite(c.buf, s, nil, false)
	q := c.bufStr()
	c.push(virt.Query(q)) //TODO need special cases for this so we can track open transactions
}
