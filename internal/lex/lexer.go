package lex

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/jimmyfrasche/etlite/internal/token"
)

type lexer struct {
	*stream
	start      token.Position
	stringType rune //the start rune for a string
	tokens     chan token.Value
	closed     bool
}

//Stream lexes src, using name as context in errors.
func Stream(name string, src io.Reader) (tokens <-chan token.Value) {
	L := &lexer{
		stream: newStream(name, bufio.NewReader(src), &bytes.Buffer{}),
		tokens: make(chan token.Value),
	}
	go run(L)
	return L.tokens
}

func (l *lexer) stickyPos() {
	l.start = l.pos()
}

func (l *lexer) error(s string) state {
	l.emitErr(errors.New(s))
	return nil
}

func (l *lexer) errorf(spec string, vs ...interface{}) state {
	l.emitErr(fmt.Errorf(spec, vs...))
	return nil
}

func (l *lexer) close() {
	if !l.closed {
		l.closed = true
		close(l.tokens)
	}
}

func (l *lexer) emit(t token.Value) {
	t.Position = l.start
	if !t.Empty() {
		t.Value = l.value()
		if t.Kind == token.Literal {
			t.Canon = strings.ToUpper(t.Value)
		}
	} else {
		l.discard()
	}
	l.tokens <- t

	if t.Kind == token.Illegal || l.stream.err == io.EOF {
		l.close()
	} else if l.stream.err != nil {
		//turn around and emit an error token after this valid token
		//if there was an IO error.
		l.emitErr(l.stream.err)
	}
}

func (l *lexer) emitErr(err error) {
	l.emit(token.Value{
		Kind: token.Illegal,
		Err:  err,
	})
}

func (l *lexer) emitString() {
	l.emit(token.Value{
		Kind:       token.String,
		StringKind: l.stringType,
	})
}

func (l *lexer) emitLit() {
	l.emitK(token.Literal)
}

func (l *lexer) emitK(k token.Kind) {
	l.emit(token.Value{
		Kind: k,
	})
}
