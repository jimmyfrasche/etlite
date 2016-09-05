//Package errusr wraps user errors with token position information.
package errusr

import (
	"errors"
	"fmt"
	"io"

	"github.com/jimmyfrasche/etlite/internal/token"
)

//User represents an error related to the input script.
type User struct {
	p   token.Position
	err error
}

//Unwrap returns the original error.
func (u User) Unwrap() error {
	return u.err
}

func (u User) Error() string {
	return fmt.Sprintf("%s: %s", u.p, u.err)
}

//Wrap an error with the position information in p.
//
//Wrapping io.EOF changes the error to io.ErrUnexpectedEOF.
func Wrap(p token.Position, err error) error {
	if err == nil {
		return nil
	}
	if err == io.EOF {
		return Wrap(p, io.ErrUnexpectedEOF)
	}
	return &User{
		p:   p,
		err: err,
	}
}

//New creates a new user error from a string.
func New(p token.Position, s string) error {
	return Wrap(p, errors.New(s))
}

//Newf creates a new formatted user error.
func Newf(p token.Position, spec string, vs ...interface{}) error {
	return Wrap(p, fmt.Errorf(spec, vs...))
}
