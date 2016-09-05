package token

import (
	"fmt"
	"strings"
)

//Value of token in input.
type Value struct {
	Position
	Kind
	//Value is only valid if !Kind.Empty()
	Value string
	//Canon is only valid if Kind == Literal and is strings.ToUpper(Value)
	Canon string
	//StringKind is only valid if Kind == String.
	//If valid it will be one of ' " ` [  or x (for blob literals).
	StringKind rune
	//Err is only non-nil if Kind == Illegal.
	Err error
}

//Equal if the same kind and value.
//Illegal, String, and Placeholder tokens are never equal.
func (v Value) Equal(s Value) bool {
	if v.Kind != s.Kind {
		return false
	}
	if v.Kind == LParen || v.Kind == RParen || v.Kind == Semicolon {
		return true
	}
	//Placeholders, and illegals are never equal, don't care about strings.
	if v.Kind != Literal {
		return false
	}
	return v.Canon == s.Canon
}

//Literal if v.Kind == Literal and v.Canon == s.
func (v Value) Literal(s string) bool {
	return v.Kind == Literal && v.Canon == s
}

//Valid token.
func (v Value) Valid() bool {
	return v.Kind != Illegal
}

func (v Value) String() string {
	if v.Kind.Empty() {
		return v.Kind.String()
	}
	if v.Kind == Illegal {
		return fmt.Sprintf("%s: illegal token %q", v.Position, v.Value)
	}
	return v.Value
}

//Unescape a string (false if not string or literal).
func (v Value) Unescape() (string, bool) {
	if v.Kind != String && v.Kind != Literal {
		return "", false
	}
	if v.Kind == Literal {
		return v.Value, true
	}
	end := len(v.Value) - 1
	switch v.StringKind {
	case 'x':
		return v.Value[2:end], true
	case '[':
		return v.Value[1:end], true
	}
	s := v.Value[1:end]
	switch v.StringKind {
	case '\'':
		return strings.Replace(s, "''", "'", -1), true
	case '`':
		return strings.Replace(s, "``", "`", -1), true
	}
	return strings.Replace(s, `""`, `"`, -1), true
}
