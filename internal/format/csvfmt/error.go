package csvfmt

import (
	"encoding/csv"

	"github.com/jimmyfrasche/etlite/internal/format"
	"github.com/jimmyfrasche/etlite/internal/internal/errsys"
)

func wrap(c interface {
	ctx() string
}, err error) error {
	if err == nil {
		return nil
	}
	ctx := c.ctx()
	switch err {
	case csv.ErrTrailingComma,
		csv.ErrBareQuote,
		csv.ErrQuote,
		csv.ErrFieldCount:
		return format.Wrap(ctx, err)
	}
	if _, ok := err.(*csv.ParseError); ok {
		return format.Wrap(ctx, err)
	}
	//everything else comes from I/O
	return errsys.Wrap(err)
}
