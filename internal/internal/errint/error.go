//Package errint provides additional context for error messages
//that are the result of a programming error.
package errint

import (
	"errors"
	"fmt"
	"runtime/debug"
)

//SkipContext may be set to true during development to elide rigmarole.
var SkipContext = false

//Internal represents an error resulting from bad programming.
//It includes context of what to do if such an error is encountered.
type Internal struct {
	e  error
	st string
}

func getst() string {
	return string(debug.Stack()) + "\n"
}

func (i *Internal) Error() string {
	//TODO include github issue queue and instructions.
	s := "internal error: " + i.e.Error()
	if SkipContext {
		return s
	}
	s += "\n"
	s += i.st
	return s
}

//Unwrap returns the wrapped error.
func (i *Internal) Unwrap() error {
	return i.e
}

//Wrap wraps an error, flagging it as an internal error.
func Wrap(e error) error {
	if e == nil {
		return nil
	}
	if _, ok := e.(*Internal); ok {
		return e
	}
	return &Internal{e, getst()}
}

//New creates an internal error from a string.
func New(s string) error {
	return Wrap(errors.New(s))
}

//Newf creates an internal error from fmt.Errorf.
func Newf(s string, vs ...interface{}) error {
	return Wrap(fmt.Errorf(s, vs...))
}
