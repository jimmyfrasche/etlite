package ast

import (
	"io"

	"github.com/jimmyfrasche/etlite/internal/ast/internal/writer"
	"github.com/jimmyfrasche/etlite/internal/token"
)

//Device represents the definition of an IO device
//in an import or display statement.
type Device interface {
	//Print for devices does not include TO or FROM.
	Print(io.Writer) error
	token.Poser
	dev()
}

//DeviceFile represents a named file.
type DeviceFile struct {
	Name token.Value
}

var _ Device = &DeviceFile{}

func (*DeviceFile) dev() {}

//Pos reports the original position in input.
func (d *DeviceFile) Pos() token.Position {
	return d.Name.Pos()
}

//Print stringifies to a writer.
func (d *DeviceFile) Print(to io.Writer) error {
	w := writer.New(to)
	w.Str(d.Name.Value)
	return w.Err()
}

//DeviceStdio represents stdin or stdout, respectively.
type DeviceStdio struct {
	token.Position //TO or FROM
}

var _ Device = &DeviceStdio{}

func (*DeviceStdio) dev() {}

//Print stringifies to a writer.
func (d *DeviceStdio) Print(w io.Writer) error {
	return writer.New(w).Str("-").Err()
}
