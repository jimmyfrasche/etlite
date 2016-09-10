//Package compile collects, compiles, and verifies the semantics
//of nodes read from a chan. (See parse package).
package compile

import (
	"runtime"

	"github.com/jimmyfrasche/etlite/internal/ast"
	"github.com/jimmyfrasche/etlite/internal/internal/errint"
	"github.com/jimmyfrasche/etlite/internal/internal/errusr"
	"github.com/jimmyfrasche/etlite/internal/virt"
)

type compiler struct {
	usedStdin bool
	inst      []virt.Instruction
}

func (c *compiler) push(is ...virt.Instruction) {
	c.inst = append(c.inst, is...)
}

//Nodes collects and compiles the nodes on from into instructions for our VM.
func Nodes(from <-chan ast.Node, usedStdin bool) (db string, to []virt.Instruction, err error) {
	c := &compiler{
		inst:      make([]virt.Instruction, 0, 128),
		usedStdin: usedStdin,
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

	return db, c.inst, nil
}
