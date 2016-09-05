package stdio

import (
	"bufio"
	"os"

	"github.com/jimmyfrasche/etlite/internal/errsys"
)

type stdout struct {
	*bufio.Writer
}

var _ Writer = (*stdout)(nil)

//Name always returns -.
func (s *stdout) Name() string {
	return "-"
}

//Unwrap returns the underlying bufio.Writer.
func (s *stdout) Unwrap() *bufio.Writer {
	return s.Writer
}

//Close flushes stdout and resets the underlying bufio.Writer
//to read from stdout again, making it re-entrant,
//in a manner of speaking.
func (s *stdout) Close() error {
	if err := s.Flush(); err != nil {
		return errsys.Wrap(err)
	}
	s.Reset(os.Stdout)
	return nil
}

type stdin struct {
	*bufio.Reader
}

var _ Reader = (*stdin)(nil)

//Name always returns -.
func (s *stdin) Name() string {
	return "-"
}

//Unwrap returns the underlying bufio.Reader.
func (s *stdin) Unwrap() *bufio.Reader {
	return s.Reader
}

//Close is a no-op for stdin.
//While all readers are read to exhaustion, this is necessary
//as stdin is the default input device, which may be immediately
//closed to read from another input device then later set as the input device.
//If it has already been exhausted it will EOF.
func (s *stdin) Close() error {
	return nil
}

var (
	//Stdout is os.Stdout wrapped to be a stdio.Writer.
	Stdout = &stdout{bufio.NewWriter(os.Stdout)}
	//Stdin is os.Stdin wrapped to be a stdio.Reader.
	Stdin = &stdin{bufio.NewReader(os.Stdin)}
)
