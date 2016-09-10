package ast

import (
	"io"

	"github.com/jimmyfrasche/etlite/internal/ast/internal/writer"
	"github.com/jimmyfrasche/etlite/internal/token"
)

//Import [format] [header] [device] [table] [limit] [offset]
type Import struct {
	token.Position
	Temporary bool
	Format    Format
	Header    []string
	Device    Device
	Table     string
	Frame     string
	Limit     IntOrSQL
	Offset    IntOrSQL
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

	if i.Format != nil {
		_ = i.Format.Print(w)
		w.Sp()
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

	if len(i.Table) > 0 {
		w.Str(" AS ").Str(i.Table).Sp()
	}

	if i.Limit != nil {
		w.Str("LIMIT ")
		intOrSQL(i.Limit, w)
		w.Sp()
	}

	if i.Offset != nil {
		w.Str("OFFSET ")
		intOrSQL(i.Offset, w)
		w.Sp()
	}
}
