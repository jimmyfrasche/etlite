package ast

import (
	"io"

	"github.com/jimmyfrasche/etlite/internal/token"
	"github.com/jimmyfrasche/etlite/internal/writer"
)

//Node is node in the AST
type Node interface {
	Print(io.Writer) error
	Pos() token.Position
	node()
}

//Use [database].
type Use struct {
	token.Position
	DB string
}

var _ Node = (*Use)(nil)

func (*Use) node() {}

//Pos reports the original position in input.
func (u *Use) Pos() token.Position {
	return u.Position
}

//Print stringifies to a writer.
func (u *Use) Print(to io.Writer) error {
	w := writer.New(to)
	w.Str("USE ").Str(u.DB)
	return w.Err()
}
