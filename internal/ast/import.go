package ast

import (
	"io"

	"github.com/jimmyfrasche/etlite/internal/ast/internal/writer"
	"github.com/jimmyfrasche/etlite/internal/token"
)

//Import [temp] [table] [header] [device] [format] [frame] [limit] [offset]
type Import struct {
	token.Position
	Temporary bool
	Table     string
	Header    []string
	Device    Device
	Format    Format
	Frame     string
	Limit     int
	Offset    int
}

var _ Node = (*Import)(nil)

func (*Import) node() {}

//Pos reports the original position in input.
func (i *Import) Pos() token.Position {
	return i.Position
}

func printCols(w *writer.Writer, cols []string) {
	for i, c := range cols {
		w.Str(c)
		if i != len(cols)-1 {
			w.Str(", ")
		}
	}
}

//Print stringifies to a writer.
func (i *Import) Print(to io.Writer) error {
	w := writer.New(to)
	i.print(w)
	return w.Err()
}

func (i *Import) print(w *writer.Writer) {
	w.Str("IMPORT ")

	if i.Temporary {
		w.Str("TEMPORARY ")
	}

	if len(i.Table) > 0 {
		w.Str(i.Table).Sp()
	}

	if len(i.Header) > 0 {
		w.Str("(")
		printCols(w, i.Header)
		w.Str(") ")
	}

	if i.Device != nil {
		w.Str("FROM ")
		_ = i.Device.Print(w)
		w.Sp()
	}

	if i.Format != nil {
		w.Str("WITH ")
		_ = i.Format.Print(w)
		w.Sp()
	}

	if i.Frame != "" {
		w.Str("FRAME ").Str(i.Frame).Sp()
	}

	if i.Limit > 0 {
		w.Str("LIMIT ").Int(i.Limit).Sp()
	}

	if i.Offset > 0 {
		w.Str("OFFSET ").Int(i.Offset).Sp()
	}
}
