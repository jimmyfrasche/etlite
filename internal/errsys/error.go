//Package errsys provides additional context for error messages
//that are the result of the underlying OS.
package errsys

import (
	"errors"
	"fmt"
	"io"
)

//System represent an OS error.
type System struct {
	ctx string
	e   error
}

func (s System) Error() string {
	return "system error:" + s.ctx + " " + s.e.Error()
}

//Unwrap returns the original error.
func (s System) Unwrap() error {
	return s.e
}

//Wrap wraps an error, flagging it as a system error.
//
//As a special case, io.EOF is returned unwrapped.
func Wrap(e error) error {
	return WrapWith("", e)
}

//WrapWith additionally records a context for where the error originated.
func WrapWith(ctx string, e error) error {
	if e == nil {
		return nil
	}
	//special case this here, rather than everywhere else
	if e == io.EOF {
		return e
	}
	return &System{ctx, e}
}

//New creates a system error from a string.
func New(s string) error {
	return Wrap(errors.New(s))
}

//Newf creates a system error from fmt.Errorf.
func Newf(s string, vs ...interface{}) error {
	return Wrap(fmt.Errorf(s, vs...))
}
