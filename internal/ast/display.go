package ast

import (
	"io"

	"github.com/jimmyfrasche/etlite/internal/ast/internal/writer"
	"github.com/jimmyfrasche/etlite/internal/token"
)

//Display [device] [format] [frame]
type Display struct {
	token.Position
	Device Device
	Format Format
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

	if d.Device != nil {
		w.Str("TO ")
		_ = d.Device.Print(w)
		w.Sp()
	}

	if d.Format != nil {
		w.Str("AS ")
		_ = d.Format.Print(w)
		w.Sp()
	}

	if d.Frame != "" {
		w.Sp().Str(" FRAME ").Str(d.Frame)
	}

	return w.Err()
}
