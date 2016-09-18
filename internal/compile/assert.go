package compile

import (
	"github.com/jimmyfrasche/etlite/internal/ast"
	"github.com/jimmyfrasche/etlite/internal/virt"
)

func (c *compiler) compileAssert(a *ast.Assert) {
	msg, _ := a.Message.Unescape()
	//TODO pull rewriting into this package
	err := a.Subquery.Print(c.buf)
	if err != nil {
		panic(err)
	}
	c.push(virt.Assert(a.Pos(), msg, c.bufStr()))
}
