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

	//normalize name
	name := strings.ToLower(fmtName(s.Name))

	//make sure these stack correctly
	switch s.Kind {
	case ast.BeginTransaction:
		if len(c.save) > 0 {
			panic(errusr.New(s.Pos(), "attempting to start transaction with open savepoints"))
		}
		if c.inTransaction {
			panic(errusr.New(s.Pos(), "attempting to start transaction in transaction"))
		}

	case ast.Commit:
		if !c.inTransaction {
			panic(errusr.New(s.Pos(), "no open transaction to commit"))
		}
		c.save = c.save[:0]
		c.inTransaction = false

	case ast.Savepoint:
		c.save = append(c.save, name)

	case ast.Release:
		p := c.saveRFind(name)
		if p < 0 {
			panic(errusr.Newf(s.Pos(), "attempting to release unknown savepoint %s", name))
		}
		c.save = c.save[:len(c.save)-p]
	}

	rewrite(c.buf, s, nil, false)
	q := c.bufStr()
	c.push(virt.Exec(q)) //TODO pass name to compiler so it can ensure proper rollback
}

func (c *compiler) saveRFind(name string) int {
	for i := len(c.save) - 1; i >= 0; i-- {
		if c.save[i] == name {
			return i
		}
	}
	return -1
}
