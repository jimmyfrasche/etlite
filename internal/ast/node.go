package ast

import (
	"io"

	"github.com/jimmyfrasche/etlite/internal/ast/internal/writer"
	"github.com/jimmyfrasche/etlite/internal/token"
)

//Node is node in the AST
type Node interface {
	Print(io.Writer) error
	token.Poser
	node()
}

//Use [database].
type Use struct {
	token.Position
	DB string
}

var _ Node = (*Use)(nil)

func (*Use) node() {}

//Print stringifies to a writer.
func (u *Use) Print(to io.Writer) error {
	w := writer.New(to)
	w.Str("USE ").Str(u.DB)
	return w.Err()
}
