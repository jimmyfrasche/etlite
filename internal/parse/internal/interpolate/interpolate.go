//Package interpolate desugars @n and @ENV into SQL tokens.
package interpolate

import (
	"strconv"
	"strings"

	"github.com/jimmyfrasche/etlite/internal/internal/errint"
	"github.com/jimmyfrasche/etlite/internal/internal/errusr"
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

func parse(t token.Value) (int, error) {
	if t.Kind != token.Argument {
		return 0, errint.New("interpolator must be called with argument token")
	}
	n, err := strconv.Atoi(t.Value)
	if err != nil {
		return 0, nil
	}
	if n < 1 {
		return 0, errusr.New(t.Position, "minimum command line interpolation is @1")
	}
	return n, nil
}

//Desugar a token of kind argument into a series of tokens,
//representing a subquery.
//It returns an error on @0, which is lexically valid but semantically undefined.
func Desugar(t token.Value) ([]token.Value, error) {
	n, err := parse(t)
	if err != nil {
		return nil, err
	}
	out := []token.Value{lit("SELECT"), lit("value"), lit("FROM"), lit("sys"), lit(".")}
	if n > 0 {
		out = append(out, lit("args"), lit("WHERE"), lit("rowid"), lit("="), lit(t.Value))
	} else {
		out = append(out, lit("env"), lit("WHERE"), lit("name"), lit("="), esc(t.Value))
	}
	return out, nil
}

func DesugarAssert(t token.Value) ([]token.Value, error) {
	n, err := parse(t)
	if err != nil {
		return nil, err
	}
	lp := token.Value{Kind: token.LParen}
	rp := token.Value{Kind: token.RParen}
	out := []token.Value{lit("SELECT"), lp, lit("SELECT"), lit("value"), lit("FROM"), lit("sys"), lit(".")}
	if n > 0 {
		out = append(out, lit("args"), lit("WHERE"), lit("rowid"), lit("="), lit(t.Value))
	} else {
		out = append(out, lit("env"), lit("WHERE"), lit("name"), lit("="), esc(t.Value))
	}
	out = append(out, rp, lit("IS"), lit("NULL"))
	return out, nil
}
