package ast

import (
	"io"

	"github.com/jimmyfrasche/etlite/internal/ast/internal/writer"
	"github.com/jimmyfrasche/etlite/internal/internal/escape"
	"github.com/jimmyfrasche/etlite/internal/internal/null"
	"github.com/jimmyfrasche/etlite/internal/token"
)

//Format represents a format specification in an import or display statement.
type Format interface {
	Print(io.Writer) error
	Pos() token.Position
	fmt()
}

//LineEnding is one of DEFAULT, CRLF, LF.
type LineEnding int

const (
	//DefaultLineEnding is the platform specific end of line.
	DefaultLineEnding LineEnding = iota
	//CRLF is the \r\n line ending
	CRLF
	//LF is the \n line ending
	LF
)

func (l LineEnding) String() string {
	switch l {
	case DefaultLineEnding:
		return "DEFAULT"
	case CRLF:
		return "CRLF"
	case LF:
		return "LF"
	}
	return "<UNKNOWN LINE ENDING KIND>"
}

//FormatCSV represents: csv [delim] [quote] [line] [null] [header]
type FormatCSV struct {
	token.Position
	Strict   bool
	Delim    rune
	Quote    rune
	Line     LineEnding
	Null     null.Encoding
	NoHeader bool
}

var _ Format = (*FormatCSV)(nil)

func (*FormatCSV) fmt() {}

//Pos reports the original position in input.
func (f *FormatCSV) Pos() token.Position {
	return f.Position
}

//Print stringifies to a writer.
func (f *FormatCSV) Print(to io.Writer) error {
	w := writer.New(to)
	w.Str("CSV")

	if f.Delim > 0 {
		w.Str(" DELIMITER ").Rune(f.Delim)
	}

	if f.Quote > 0 {
		w.Str(" QUOTE ").Rune(f.Quote)
	}

	if f.Null != "" {
		w.Str(" NULL ").Str(escape.String(string(f.Null)))
	}

	w.Sp().Stringer(f.Line).Sp()

	if f.NoHeader {
		w.Str("NOHEADER ")
	}

	return w.Err()
}

//FormatRaw represents raw [delim] [line] [null] [header]
type FormatRaw struct {
	token.Position
	Strict bool
	Delim  rune
	Line   LineEnding
	Null   null.Encoding
	Header bool
}

var _ Format = (*FormatRaw)(nil)

func (*FormatRaw) fmt() {}

//Pos reports the original position in input.
func (f *FormatRaw) Pos() token.Position {
	return f.Position
}

//Print stringifies to a writer.
func (f *FormatRaw) Print(to io.Writer) error {
	w := writer.New(to)
	w.Str("RAW ")

	if f.Delim > 0 {
		w.Str(" DELIMITER ").Rune(f.Delim)
	}

	if f.Null != "" {
		w.Str(" NULL ").Str(escape.String(string(f.Null)))
	}

	w.Sp().Stringer(f.Line).Sp()

	if f.Header {
		w.Str("HEADER ")
	}

	return w.Err()
}

//FormatJSON is a JSON format directive in a display or import statement.
type FormatJSON struct {
	token.Position
}

var _ Format = (*FormatJSON)(nil)

func (*FormatJSON) fmt() {}

//Pos reports the original position in input.
func (f *FormatJSON) Pos() token.Position {
	return f.Position
}

//Print stringifies to a writer.
func (f *FormatJSON) Print(to io.Writer) error {
	w := writer.New(to)
	w.Str("JSON")
	return w.Err()
}
