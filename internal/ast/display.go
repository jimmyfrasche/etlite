package ast

import (
	"io"

	"github.com/jimmyfrasche/etlite/internal/token"
	"github.com/jimmyfrasche/etlite/internal/writer"
)

//Display [format] [device]
type Display struct {
	token.Position
	Format Format
	Device Device
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
