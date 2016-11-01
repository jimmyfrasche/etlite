package token

import "fmt"

//Posers can report their Position.
type Poser interface {
	Pos() Position
}

//Position of a token in input.
type Position struct {
	Name       string
	Line, Rune int
}

func (p Position) String() string {
	return fmt.Sprintf("%s:%d:%d", p.Name, p.Line, p.Rune)
}

func (p Position) Pos() Position {
	return p
}
