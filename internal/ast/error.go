package ast

import (
	"io"

	"github.com/jimmyfrasche/etlite/internal/token"
	"github.com/jimmyfrasche/etlite/internal/writer"
)

//Error represents an error during parsing.
//
//Token is the token that caused the error.
//
//If its kind is illegal, Err will be nil.
type Error struct {
	Token token.Value
	Err   error
}

var _ Node = (*Error)(nil)

func (*Error) node() {}

//Pos reports the original position in input.
func (e *Error) Pos() token.Position {
	return e.Token.Position
}

//Print stringifies to a writer.
func (e *Error) Print(to io.Writer) error {
	w := writer.New(to)
	w.Sticky(e)
	return w.Err()
}

func (e *Error) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	//illegal token creates its own error string
	return e.Token.String()
}
