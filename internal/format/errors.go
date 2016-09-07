package format

import (
	"errors"
	"fmt"
)

var (
	//ErrNoHeader means that the names of columns need
	//to be specified by the input but were not.
	ErrNoHeader = errors.New("column names cannot be derived")
)

type dimError struct {
	ctx           string
	expected, got int
}

//NewDimErr creates a new dimension error when the input, described by ctx,
//expected n column but got m.
func NewDimErr(ctx string, expected, got int) error {
	return &dimError{
		ctx:      ctx,
		expected: expected,
		got:      got,
	}
}

func (d *dimError) Error() string {
	return fmt.Sprintf("%s expected %d columns but got %d", d.ctx, d.expected, d.got)
}

type fmtError struct {
	ctx string
	e   error
}

//Wrap an error as a format error.
func Wrap(ctx string, err error) error {
	if err == nil {
		return nil
	}
	return &fmtError{
		ctx: ctx,
		e:   err,
	}
}

func (f *fmtError) Error() string {
	if f.ctx == "" {
		return f.e.Error()
	}
	return fmt.Sprintf("%s %s", f.ctx, f.e)
}

//IsFormatErr returns whether e is an error in the format.
func IsFormatErr(e error) bool {
	if e == nil {
		return false
	}
	if e == ErrNoHeader {
		return true
	}
	switch e.(type) {
	case *dimError, *fmtError:
		return true
	}
	return false
}
