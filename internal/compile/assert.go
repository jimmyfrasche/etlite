package compile

import (
	"github.com/jimmyfrasche/etlite/internal/ast"
	"github.com/jimmyfrasche/etlite/internal/virt"
)

func (c *compiler) compileAssert(a *ast.Assert) {
	msg, _ := a.Message.Unescape()
	q, err := a.Subquery.ToString()
	if err != nil {
		panic(err)
	}
	c.push(virt.MkAssert(a.Pos(), msg, q))
}
