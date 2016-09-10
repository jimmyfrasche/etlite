//Package runefrom provides a helper for converting an arbitrary string to a single rune.
package runefrom

import (
	"errors"
	"fmt"
	"strconv"
	"unicode/utf8"
)

//String converts a nonempty string to a rune,
//returning an error if s contains more than one rune or the rune is invalid.
func String(s string) (rune, error) {
	if s == "" {
		return -1, errors.New("no unicode code point specified")
	}
	r, _, t, err := strconv.UnquoteChar(s, 0)
	if err != nil {
		return -1, err
	}
	if r == utf8.RuneError {
		return -1, errors.New("invalid unicode code point")
	}
	if len(t) > 0 {
		return -1, fmt.Errorf("expected a single unicode code point, got %q", s)
	}
	return r, nil
}
