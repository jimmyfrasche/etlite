package ast

import (
	"io"

	"github.com/jimmyfrasche/etlite/internal/token"
	"github.com/jimmyfrasche/etlite/internal/writer"
)

//Device represents the definition of an IO device
//in an import or display statement.
type Device interface {
	//Print for devices does not include TO or FROM.
	Print(io.Writer) error
	Pos() token.Position
	dev()
}

//DeviceFile represents a named file.
type DeviceFile struct {
	token.Position //TO or FROM
	Name           StringOrSQL
}

var _ Device = &DeviceFile{}

func (*DeviceFile) dev() {}

//Pos reports the original position in input.
func (d *DeviceFile) Pos() token.Position {
	return d.Position
}

//Print stringifies to a writer.
func (d *DeviceFile) Print(to io.Writer) error {
	w := writer.New(to)
	stringOrSQL(d.Name, w)
	return w.Err()
}

//DeviceStdio represents stdin or stdout, respectively.
type DeviceStdio struct {
	token.Position //TO or FROM
}

var _ Device = &DeviceStdio{}

func (*DeviceStdio) dev() {}

//Pos reports the original position in input.
func (d *DeviceStdio) Pos() token.Position {
	return d.Position
}

//Print stringifies to a writer.
func (d *DeviceStdio) Print(w io.Writer) error {
	return writer.New(w).Str("-").Err()
}
