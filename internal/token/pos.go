package token

import "fmt"

//Position of a token in input.
type Position struct {
	Name       string
	Line, Rune int
}

func (p Position) String() string {
	return fmt.Sprintf("%s:%d:%d", p.Name, p.Line, p.Rune)
}
