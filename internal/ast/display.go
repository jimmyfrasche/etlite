package ast

import (
	"io"

	"github.com/jimmyfrasche/etlite/internal/ast/internal/writer"
	"github.com/jimmyfrasche/etlite/internal/token"
)

//Display [format] [device]
type Display struct {
	token.Position
	Format Format
	Device Device
	Frame  string
}

var _ Node = (*Display)(nil)

func (*Display) node() {}

//Pos reports the original position in input.
func (d *Display) Pos() token.Position {
	return d.Position
}

//Print stringifies to a writer.
func (d *Display) Print(to io.Writer) error {
	w := writer.New(to)
	w.Str("DISPLAY ")

	if d.Format != nil {
		_ = d.Format.Print(w)
		w.Sp()
	}

	if d.Device != nil {
		w.Str("TO ")
		_ = d.Device.Print(w)
	}

	return w.Err()
}
