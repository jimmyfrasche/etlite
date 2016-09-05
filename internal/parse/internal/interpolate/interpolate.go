//Package interpolate desugars @n and @ENV into SQL tokens.
package interpolate

import (
	"errors"
	"strconv"
	"strings"

	"github.com/jimmyfrasche/etlite/internal/token"
)

func lit(s string) token.Value {
	return token.Value{
		Kind:  token.Literal,
		Value: s,
		Canon: strings.ToUpper(s),
	}
}

func esc(s string) token.Value {
	//double up ' and wrap in '
	return token.Value{
		Kind:       token.String,
		StringKind: '\'',
		Value:      "'" + strings.Replace(s, "'", "''", -1) + "'",
	}
}

//Desugar a token of kind argument into a series of tokens,
//representing a subquery.
//It returns an error on @0, which is lexically valid but semantically undefined.
func Desugar(t token.Value) ([]token.Value, error) {
	if t.Kind != token.Argument {
		panic("internal error: must be called with argument token")
	}
	out := []token.Value{lit("SELECT"), lit("value"), lit("FROM"), lit("sys"), lit(".")}
	if n, err := strconv.Atoi(t.Value); err != nil {
		out = append(out, lit("env"), lit("WHERE"), lit("name"), lit("="), esc(t.Value))
	} else {
		if n < 1 {
			return nil, errors.New("minimum command line interpolation is @1")
		}
		out = append(out, lit("args"), lit("WHERE"), lit("rowid"), lit("="), lit(t.Value))
	}
	return out, nil
}
