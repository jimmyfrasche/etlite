//Package compile collects, compiles, and verifies the semantics
//of nodes read from a chan. (See parse package).
package compile

import (
	"bytes"
	"runtime"
	"strings"

	"github.com/jimmyfrasche/etlite/internal/ast"
	"github.com/jimmyfrasche/etlite/internal/internal/errint"
	"github.com/jimmyfrasche/etlite/internal/internal/errusr"
	"github.com/jimmyfrasche/etlite/internal/internal/savepoint"
	"github.com/jimmyfrasche/etlite/internal/virt"
)

type compiler struct {
	usedStdin, hadDevice bool

	dname, frname string
	used          map[string]bool
	hdr           []string

	inst []virt.Instruction

	//caches for query rewriting
	buf *bytes.Buffer
	r   *ast.SQL

	//track savepoints to ensure no invalid programs
	stack *savepoint.Stack
}

func (c *compiler) push(is ...virt.Instruction) {
	c.inst = append(c.inst, is...)
}

func (c *compiler) bufStr() string {
	s := c.buf.String()
	c.buf.Reset()
	return s
}

func (c *compiler) rec(name string) {
	c.used[strings.ToLower(name)] = true
}

func (c *compiler) nameUsed(nm string) bool {
	return c.used[strings.ToLower(nm)]
}

//Nodes collects and compiles the nodes on from into instructions for our VM.
func Nodes(from <-chan ast.Node, usedStdin bool) (db string, to []virt.Instruction, err error) {
	c := &compiler{
		inst:      make([]virt.Instruction, 0, 128),
		usedStdin: usedStdin,
		used:      map[string]bool{},
		buf:       &bytes.Buffer{},
		r:         &ast.SQL{Kind: ast.Query},
		stack:     savepoint.New(),
	}

	if !usedStdin {
		c.dname = "-"
	}

	defer func() {
		if x := recover(); x != nil {
			e, ok := x.(error)
			if ok {
				if _, ok := e.(runtime.Error); ok {
					panic(x)
				}
			} else {
				panic(x)
			}
			err = e
		}
	}()

	firstStatement := true
	for n := range from {
		c.push(virt.ErrPos(n.Pos()))
		switch n := n.(type) {
		default:
			return "", nil, errint.Newf("internal error: unknown node type %T", n)

		case *ast.Error:
			return "", nil, n

		case *ast.Use:
			if !firstStatement {
				return "", nil, errusr.New(n.Pos(), "USE must be first statement")
			}
			db = n.DB

		case *ast.Assert:
			c.compileAssert(n)

		case *ast.Display:
			c.compileDisplay(n)

		case *ast.Import:
			c.compileImport(n)

		case *ast.SQL:
			c.compileSQL(n)
		}

		firstStatement = false
	}

	if c.stack.InTransaction() {
		c.push(virt.Exec("END TRANSACTION;"))
	} else if c.stack.HasSavepoints() {
		c.push(virt.Exec("RELEASE " + c.stack.Top() + ";"))
	}

	return db, c.inst, nil
}
