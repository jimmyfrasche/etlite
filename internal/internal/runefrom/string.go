//Package runefrom provides a helper for converting an arbitrary string to a single rune.
package runefrom

import (
	"errors"
	"fmt"
	"unicode/utf8"
)

//String converts a nonempty string to a rune,
//returning an error if s contains more than one rune or the rune is invalid.
func String(s string) (rune, error) { //BUG(jmf): should handle \0 \n \t
	if s == "" {
		return -1, errors.New("no unicode code point specified")
	}
	r, sz := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError {
		return -1, errors.New("invalid unicode code point")
	}
	if s[sz:] != "" {
		return -1, fmt.Errorf("expected a single unicode code point, got %q", s)
	}
	return r, nil
}
