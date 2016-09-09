package ast

import (
	"io"

	"github.com/jimmyfrasche/etlite/internal/ast/internal/writer"
	"github.com/jimmyfrasche/etlite/internal/token"
)

//Assert msg, subquery.
type Assert struct {
	token.Position
	Message  token.Value
	Subquery *SQL
}

func (*Assert) node() {}

//Pos reports the original position in input.
func (a *Assert) Pos() token.Position {
	return a.Position
}

//Print stringifies to a writer.
func (a *Assert) Print(to io.Writer) error {
	w := writer.New(to)
	w.Str("ASSERT ").Str(a.Message.Value).Str(", ")
	_ = a.Subquery.Print(w)
	return w.Err()
}
