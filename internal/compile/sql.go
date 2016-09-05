package compile

import (
	"fmt"

	"github.com/jimmyfrasche/etlite/internal/ast"
	"github.com/jimmyfrasche/etlite/internal/engine"
	"github.com/jimmyfrasche/etlite/internal/errint"
	"github.com/jimmyfrasche/etlite/internal/token"
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
	c.push(engine.MkSavepoint())
}

func (c *compiler) release() {
	c.push(engine.MkRelease())
}

//handler is only non-nil for subqueries in imports
func (c *compiler) compileSQL(s *ast.SQL, handler func(*string) (interface{}, error)) {
	c.sqlDepth++
	defer func() {
		c.sqlDepth--
	}()

	//CREATE TABLE ... FROM IMPORT is a special case
	if len(s.Name) > 0 {
		if ln := len(s.Name); ln == 2 || ln > 3 {
			panic(errint.Newf("table name in CREATE TABLE FROM IMPORT must have 1 or 3 tokens, got %d", ln))
		}
		if len(s.Subqueries) != 1 {
			panic(errint.Newf("found %d imports in CREATE TABLE FROM IMPORT but should only have 1", len(s.Subqueries)))
		}
		if handler != nil {
			panic(errint.New("handler must be nil in CREATE TABLE FROM IMPORT"))
		}

		nm := tblFrom(s.Name)

		ddl, err := s.ToString()
		if err != nil {
			panic(err)
		}

		i := s.Subqueries[0]
		c.push(engine.MkCreateTableFrom(i.Pos(), nm, ddl))
		c.savepoint()
		c.compileCreateTableAsImport(nm, i)
		c.release()
		return
	}

	//regular sql, may or may not have etl subqueries

	//if etl subquery, handle set up
	var tbls []string
	if len(s.Subqueries) > 0 {
		if handler == nil { //otherwise we're in a nested subquery
			c.savepoint()
		}

		//compile the imports
		tbls = make([]string, len(s.Subqueries))
		for i, imp := range s.Subqueries {
			tbls[i] = fmt.Sprintf("[%d-%d]", c.sqlDepth, i)
			c.compileSubImport(imp, tbls[i])
		}

		//rewrite the placeholders to select from our well named tables.
		i := 0
		for j, t := range s.Tokens {
			if t.Kind == token.Placeholder {
				s.Tokens[j] = token.Value{
					Kind:  token.Literal,
					Value: "select * from temp." + tbls[i],
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
	if handler == nil {
		c.push(engine.MkQuery(q))
	} else {
		c.push(engine.MkPushSubquery(q, handler))
	}

	//if this was an etl subquery, handle teardown
	if len(s.Subqueries) > 0 {
		for i := range s.Subqueries {
			c.push(engine.MkDropTempTable(tbls[i]))
		}
		if handler == nil {
			c.release()
		}
	}
}
