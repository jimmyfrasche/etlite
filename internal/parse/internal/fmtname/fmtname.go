package fmtname

import (
	"github.com/jimmyfrasche/etlite/internal/internal/errint"
	"github.com/jimmyfrasche/etlite/internal/internal/escape"
	"github.com/jimmyfrasche/etlite/internal/token"
)

func stringify(t token.Value) (string, error) {
	s, ok := t.Unescape()
	if !ok {
		return "", errint.Newf("token not literal or string, got %s", t.Kind)
	}
	return escape.String(s), nil
}

func ToString(name []token.Value) (string, error) {
	if ln := len(name); ln != 1 && ln != 3 {
		return "", errint.Newf("name must be 1 or 3 tokens, got %d", ln)
	}
	s, err := stringify(name[0])
	if err != nil {
		return "", err
	}
	s = escape.String(s)
	if len(name) > 1 {
		if !name[1].Literal(".") {
			return "", errint.Newf("name[1] must be '.', got %v", name[1])
		}
		s += "."
		t, err := stringify(name[2])
		if err != nil {
			return "", err
		}
		s += t
	}
	return s, nil
}
