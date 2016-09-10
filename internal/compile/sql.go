package compile

import (
	"strconv"

	"github.com/jimmyfrasche/etlite/internal/ast"
	"github.com/jimmyfrasche/etlite/internal/internal/errint"
	"github.com/jimmyfrasche/etlite/internal/token"
	"github.com/jimmyfrasche/etlite/internal/virt"
)

func tblFrom(nm []token.Value) string {
	var s string
	//we assume the escaping, if any, has been applied
	for _, t := range nm {
		s += t.Value
	}
	return s
}

func (c *compiler) savepoint() {
	c.push(virt.MkSavepoint())
}

func (c *compiler) release() {
	c.push(virt.MkRelease())
}

func (c *compiler) compileSQL(s *ast.SQL) {

	//CREATE TABLE ... FROM IMPORT is a special case
	if len(s.Name) > 0 {
		if ln := len(s.Name); ln == 2 || ln > 3 {
			panic(errint.Newf("table name in CREATE TABLE FROM IMPORT must have 1 or 3 tokens, got %d", ln))
		}
		if len(s.Subqueries) != 1 {
			panic(errint.Newf("found %d imports in CREATE TABLE FROM IMPORT but should only have 1", len(s.Subqueries)))
		}

		nm := tblFrom(s.Name)

		ddl, err := s.ToString()
		if err != nil {
			panic(err)
		}

		i := s.Subqueries[0]
		c.push(virt.MkCreateTableFrom(i.Pos(), nm, ddl))
		c.savepoint()
		c.compileCreateTableAsImport(nm, i)
		c.release()
		return
	}

	//regular sql, may or may not have etl subqueries

	//if etl subquery, handle set up
	var tbls []string
	if len(s.Subqueries) > 0 {
		c.savepoint()

		//compile the imports
		tbls = make([]string, len(s.Subqueries))
		for i, imp := range s.Subqueries {
			tbls[i] = "[" + strconv.Itoa(i) + "]"
			c.compileSubImport(imp, tbls[i])
		}

		//rewrite the placeholders to select from our well named tables.
		i := 0
		for j, t := range s.Tokens {
			if t.Kind == token.Placeholder {
				s.Tokens[j] = token.Value{
					Kind:  token.Literal,
					Value: "select * from temp." + tbls[i], //TODO create synthetic tokens
				}
				i++
			}
		}
		if i != len(s.Subqueries) {
			panic(errint.Newf("expected %d placeholders in subquery got %d:\n%v", len(s.Subqueries), i, s))
		}

	}

	q, err := s.ToString()
	if err != nil {
		panic(err)
	}
	c.push(virt.MkQuery(q))

	//if this was an etl subquery, handle teardown
	if len(s.Subqueries) > 0 {
		c.push(virt.MkDropTempTables(tbls))
		c.release()
	}
}
