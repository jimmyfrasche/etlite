package lex

import "github.com/jimmyfrasche/etlite/internal/token"

//streamSource are the methods of bufio.Reader we use.
type streamSource interface {
	ReadRune() (rune, int, error)
	UnreadRune() error
}

//streamSink are the methods of bytes.Buffer we use.
type streamSink interface {
	WriteRune(rune) (int, error)
	Reset()
	String() string
}

//stream represents our position in the input stream
//and provides primitives for reading
type stream struct {
	//name of the input stream
	name string
	//line number in stream
	line int
	//offset (in runes) in stream and last offset
	off, lastOff int
	//current and last rune in stream.
	c, lastC rune

	src  streamSource
	sink streamSink

	//err is any error we ran across trying to read runes
	err error
}

func newStream(name string, src streamSource, sink streamSink) *stream {
	s := &stream{
		name:    name,
		lastOff: -2, //illegal value to catch misuse
		lastC:   -2, //illegal value to catch misuse
		line:    1,
		src:     src,
		sink:    sink,
	}
	s.next() //prime pump
	return s
}

//pos returns the current position in the stream.
func (s *stream) pos() token.Position {
	return token.Position{
		Name: s.name,
		Line: s.line,
		Rune: s.off,
	}
}

//discard buffered input.
func (s *stream) discard() {
	s.sink.Reset()
}

func (s *stream) value() string {
	v := s.sink.String()
	s.discard()
	return v
}

func (s *stream) consume() {
	if s.c < 0 {
		//this would be a bad programming error
		panic("invalid rune to consume")
	}
	_, _ = s.sink.WriteRune(s.c)
	s.lastC, s.lastOff = -2, -2
}

func (s *stream) rewind() {
	if s.lastC == -2 || s.lastOff == -2 {
		panic("internal error rewind called twice or after commit")
	}
	if !s.eof() {
		if err := s.src.UnreadRune(); err != nil {
			//trying to unread past start of input, something is wrong with the code
			//should be caught by above, but no point taking a chance
			panic(err)
		}
	}
	if s.c == '\n' {
		s.line--
	}
	s.c, s.lastC = s.lastC, -2
	s.off, s.lastOff = s.lastOff, -2
}

func (s *stream) next() {
	//eof is impassable
	if s.eof() {
		return
	}

	r, _, err := s.src.ReadRune()
	if err != nil {
		r = eof
		s.err = err
	}

	s.lastC, s.c = s.c, r
	s.lastOff = s.off

	if r == '\n' {
		s.off = 0
		s.line++
	} else {
		s.off++
	}
}

func (s *stream) eof() bool {
	return s.c == eof
}

func (s *stream) until(r rune) {
	s.untilp(is(r))
}

func (s *stream) untilp(p pred) {
	for {
		s.next()
		if p(s.c) || s.eof() {
			return
		}
		s.consume()
	}
}

//ignoreUntil is untilp except that it never consumes any input.
func (s *stream) ignoreUntil(p pred) {
	for {
		s.next()
		if p(s.c) || s.eof() {
			return
		}
	}
}

//maybe consumes the next rune if it is r and rewinds if not.
//
//It always returns false if at the end of input.
//
//Note that it's generally not safe to call !maybe unless you fail if !maybe.
func (s *stream) maybe(r rune) bool {
	return s.maybep(is(r))
}

//maybep consumes the next rune if it matches p and rewinds if not.
//
//It always returns false if at the end of input.
//
//Note that it's generally not safe to call !maybep unless you fail if !maybep.
func (s *stream) maybep(p pred) bool {
	if s.eof() {
		return false
	}
	s.next()
	if !s.eof() && p(s.c) {
		s.consume()
		return true
	}
	s.rewind()
	return false
}
