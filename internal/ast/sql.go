package ast

import (
	"bytes"
	"io"

	"github.com/jimmyfrasche/etlite/internal/ast/internal/writer"
	"github.com/jimmyfrasche/etlite/internal/internal/errint"
	"github.com/jimmyfrasche/etlite/internal/token"
)

//go:generate stringer -type=Kind

type Kind int

const (
	Invalid Kind = iota
	Query
	Exec
	CreateTableFrom
	CreateTableAs
	InsertFrom
	Savepoint
	Release
	BeginTransaction
	Commit
)

//A SQL statement or subquery
//(not including outer parentheses or final semicolon).
//
//It is up to a third party to rewrite subqueries to contain only valid sql
type SQL struct {
	Kind       Kind
	Subqueries []*Import
	Name       []token.Value //only set if CREATE TABLE ... FROM IMPORT
	Cols       []token.Value //recorded for INSERT
	Tokens     []token.Value
}

var _ Node = (*SQL)(nil)

func (*SQL) node() {}

func (*SQL) int() {}

func (*SQL) rune() {}

func (*SQL) null() {}

func (*SQL) str() {}

func (*SQL) bool() {}

//Pos reports the original position in input.
func (s *SQL) Pos() token.Position {
	return s.Tokens[0].Position
}

//ToString calls Print on a bytes.Buffer.
//It is only safe to call after replacing argument and placeholder tokens.
func (s *SQL) ToString() (string, error) {
	var b bytes.Buffer
	if err := s.Print(&b); err != nil {
		return "", err
	}
	return b.String(), nil
}

//Print stringifies to a writer.
func (s *SQL) Print(to io.Writer) error {
	if s.Kind == Invalid {
		return errint.New("Improperly constructed SQL")
	}
	w := writer.New(to)

	//To avoid handling precedence and such we do not handle unary + or - when lexing
	//so here we must not put a space between +, - and a numeric literal
	//to further simplify this we only emit spaces between two literals.
	//This is not very pretty but it ensures everything works,
	//as long as the underlying SQL is valid.

	var (
		lastWasLit  bool
		placeholder int
	)
	for i, tok := range s.Tokens {

		isLit := tok.Kind == token.Literal && !tok.Op()
		if lastWasLit && isLit {
			w.Sp()
		}
		lastWasLit = isLit

		switch tok.Kind {
		case token.Illegal: //shouldn't happen but why not check anyway?
			w.Sticky(tok.Err)
			return nil
		case token.Argument:
			//parser rewrites arguments to sql
			w.Sticky(errint.Newf("unexpected token at %d, %q", i, tok))
			return nil

		case token.Placeholder:
			if placeholder < 0 || placeholder >= len(s.Subqueries) {
				w.Sticky(errint.Newf("invalid subquery index %d", placeholder))
				return nil
			}

			w.Str("(")
			_ = s.Subqueries[placeholder].Print(w)
			w.Str(")")
			placeholder++

		default:
			w.Stringer(tok)
		}
	}

	return w.Err()
}
