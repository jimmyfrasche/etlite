package ast

import (
	"io"
	"strconv"

	"github.com/jimmyfrasche/etlite/internal/errint"
	"github.com/jimmyfrasche/etlite/internal/internal/escape"
	"github.com/jimmyfrasche/etlite/internal/null"
	"github.com/jimmyfrasche/etlite/internal/token"
	"github.com/jimmyfrasche/etlite/internal/writer"
)

//Int is a literal int.
type Int struct {
	token.Position
	Value int
}

var _ Node = (*Int)(nil)

func (*Int) node() {}

func (*Int) int() {}

//Pos reports the original position in input.
func (i *Int) Pos() token.Position {
	return i.Position
}

func (i *Int) print(w *writer.Writer) {
	w.Str(strconv.Itoa(i.Value))
}

//Print stringifies to a writer.
func (i *Int) Print(to io.Writer) error {
	w := writer.New(to)
	i.print(w)
	return w.Err()
}

func intOrSQL(n Node, w *writer.Writer) {
	switch n := n.(type) {
	case *Int:
		n.print(w)
	case *SQL:
		_ = n.Print(w)
	default:
		w.Sticky(errint.New("expected Int or SQL node"))
	}
}

//IntOrSQL is either an Int or SQL Node.
type IntOrSQL interface {
	Node
	int()
}

//Rune is a literal rune.
type Rune struct {
	token.Position
	Value rune
}

var _ Node = (*Rune)(nil)

func (*Rune) node() {}

func (*Rune) rune() {}

//Pos reports the original position in input.
func (r *Rune) Pos() token.Position {
	return r.Position
}

func (r *Rune) print(w *writer.Writer) {
	if r.Value == '\'' {
		w.Str(`"'"`)
		return
	}
	w.Str("'").Rune(r.Value).Str("'")
}

//Print stringifies to a writer.
func (r *Rune) Print(to io.Writer) error {
	w := writer.New(to)
	r.print(w)
	return w.Err()
}

type RuneOrSQL interface {
	Node
	rune()
}

func runeOrSQL(n Node, w *writer.Writer) {
	switch n := n.(type) {
	case *Rune:
		n.print(w)
	case *SQL:
		_ = n.Print(w)
	default:
		w.Sticky(errint.New("expected Rune or SQL node"))
	}
}

//Null is a literal representing a null.Encoding
type Null struct {
	token.Position
	Value null.Encoding
}

var _ Node = (*Null)(nil)

func (*Null) node() {}

func (*Null) null() {}

//Pos reports the original position in input.
func (n *Null) Pos() token.Position {
	return n.Position
}

func (n *Null) print(w *writer.Writer) {
	w.Str(escape.String(string(n.Value)))
}

//Print stringifies to a writer.
func (n *Null) Print(to io.Writer) error {
	w := writer.New(to)
	n.print(w)
	return w.Err()
}

type NullOrSQL interface {
	Node
	null()
}

func nullOrSQL(n Node, w *writer.Writer) {
	switch n := n.(type) {
	case *Null:
		n.print(w)
	case *SQL:
		_ = n.Print(w)
	default:
		w.Sticky(errint.New("expected Null or SQL node"))
	}
}

//String is a literal string
type String struct {
	token.Position
	Value string
}

var _ Node = (*String)(nil)

func (*String) node() {}

func (*String) str() {}

//Pos reports the original position in input.
func (s *String) Pos() token.Position {
	return s.Position
}

func (s *String) print(w *writer.Writer) {
	w.Str(escape.String(s.Value))
}

//Print stringifies to a writer.
func (s *String) Print(to io.Writer) error {
	w := writer.New(to)
	s.print(w)
	return w.Err()
}

type StringOrSQL interface {
	Node
	str()
}

func stringOrSQL(n Node, w *writer.Writer) {
	switch n := n.(type) {
	case *String:
		n.print(w)
	case *SQL:
		_ = n.Print(w)
	default:
		w.Sticky(errint.New("expected String or SQL node"))
	}
}

//Bool is a literal bool
type Bool struct {
	token.Position
	Value bool
}

var _ Node = (*Bool)(nil)

func (*Bool) node() {}

func (*Bool) bool() {}

//Pos reports the original position in input.
func (b *Bool) Pos() token.Position {
	return b.Position
}

func (b *Bool) print(w *writer.Writer) {
	if b.Value {
		w.Str("1")
	} else {
		w.Str("0")
	}
}

//Print stringifies to a writer.
func (b *Bool) Print(to io.Writer) error {
	w := writer.New(to)
	b.print(w)
	return w.Err()
}

type BoolOrSQL interface {
	Node
	bool()
}

func boolOrSQL(n Node, w *writer.Writer) {
	switch n := n.(type) {
	case *Bool:
		n.print(w)
	case *SQL:
		_ = n.Print(w)
	default:
		w.Sticky(errint.New("expected Bool or SQL node"))
	}
}
