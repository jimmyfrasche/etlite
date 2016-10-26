package compile

import (
	"strings"

	"github.com/jimmyfrasche/etlite/internal/ast"
	"github.com/jimmyfrasche/etlite/internal/token"
	"github.com/jimmyfrasche/etlite/internal/virt"
)

func (c *compiler) compileAssert(a *ast.Assert) {
	msg, _ := a.Message.Unescape()
	var stmt string
	if ts := a.Subquery.Tokens; len(ts) == 1 && ts[0].Kind == token.Argument {
		s, isNum := parseArg(ts[0])
		q := []string{"SELECT (SELECT value FROM sys.", "env", " WHERE ", "name", "=", s, ") IS NULL"}
		if isNum {
			q[1], q[3] = "args", "rowid"
		}
		stmt = strings.Join(q, "")
	} else {
		stmt = c.rewrite(a.Subquery, nil, false)
	}
	c.push(virt.ErrPos(a.Pos()))
	c.push(virt.Assert(a.Pos(), msg, stmt))
}
