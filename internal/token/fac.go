package token

import (
	"strings"

	"github.com/jimmyfrasche/etlite/internal/internal/escape"
)

func NewString(s string) Value {
	return Value{
		Kind:       String,
		Value:      escape.String(s),
		StringKind: '\'',
	}
}

func NewLiteral(s string) Value {
	return Value{
		Kind:  Literal,
		Value: s,
		Canon: strings.ToUpper(s),
	}
}
