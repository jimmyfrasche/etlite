package ast

import (
	"io"

	"github.com/jimmyfrasche/etlite/internal/ast/internal/writer"
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
	Strict bool
	Delim  RuneOrSQL
	Quote  RuneOrSQL
	Line   LineEnding
	Null   NullOrSQL
	Header BoolOrSQL
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

	if f.Delim != nil {
		w.Str(" DELIMITER ")
		runeOrSQL(f.Delim, w)
	}

	if f.Quote != nil {
		w.Str(" QUOTE ")
		runeOrSQL(f.Quote, w)
	}

	if f.Null != nil {
		w.Str(" NULL ")
		nullOrSQL(f.Null, w)
	}

	w.Sp().Stringer(f.Line).Sp()

	if f.Header != nil {
		w.Str("HEADER ")
		boolOrSQL(f.Header, w)
	}

	return w.Err()
}

//FormatRaw represents raw [delim] [line] [null] [header]
type FormatRaw struct {
	token.Position
	Strict bool
	Delim  RuneOrSQL
	Line   LineEnding
	Null   NullOrSQL
	Header BoolOrSQL
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

	if f.Delim != nil {
		w.Str(" DELIMITER ")
		runeOrSQL(f.Delim, w)
	}

	if f.Null != nil {
		w.Str(" NULL ")
		nullOrSQL(f.Null, w)
	}

	w.Sp().Stringer(f.Line).Sp()

	if f.Header != nil {
		w.Str("HEADER ")
		boolOrSQL(f.Header, w)
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
