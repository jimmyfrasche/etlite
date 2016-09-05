package parse

import (
	"fmt"

	"github.com/jimmyfrasche/etlite/internal/ast"
	"github.com/jimmyfrasche/etlite/internal/internal/errusr"
	"github.com/jimmyfrasche/etlite/internal/token"
)

func (p *parser) mkErr(t token.Value, err error) error {
	return &ast.Error{
		Token: t,
		Err:   err,
	}
}

func (p *parser) errMsg(t token.Value, msg string, vs ...interface{}) error {
	return p.mkErr(t, errusr.Wrap(t.Position, fmt.Errorf(msg, vs...)))
}

func (p *parser) unexpected(t token.Value) error {
	return p.errMsg(t, "unexpected %s", t)
}

func (p *parser) expected(what interface{}, got token.Value) error {
	return p.errMsg(got, "expected %s, but got %s", what, got)
}
