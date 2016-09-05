package lex

import (
	"bytes"
	"strings"
	"testing"

	"github.com/jimmyfrasche/etlite/internal/token"
)

func fakeStream(s string) *stream {
	return newStream(
		"test",
		strings.NewReader(s),
		&bytes.Buffer{},
	)
}

func TestEmptyStream(t *testing.T) {
	s := fakeStream("")
	if s.c != eof {
		t.Fatal("empty stream should immediately reply with eof")
	}
}

func TestStreamStep(t *testing.T) {
	const input = "abcdefgh"
	s := fakeStream(input)
	if s.c != 'a' {
		t.Fatal("inital stream should report a as first char")
	}
	s.consume()
	for _, r := range input[1:] {
		s.next()
		if s.c != r {
			t.Fatalf("Expected '%c' got '%c'", r, s.c)
		}
		s.consume()
	}
	s.next() //step off last character
	if !s.eof() {
		t.Log(s.value())
		t.Fatal("exhausted stream should be at EOF")
	}
	if v := s.value(); input != v {
		t.Fatalf("expected %q after read but got %q", input, v)
	}
	if s.value() != "" {
		t.Fatal("discarding buffer failed")
	}
}

func TestMaybe(t *testing.T) {
	s := fakeStream("ab")
	if s.maybe('c') {
		t.Fatal("should not get c")
	}
	if !s.maybe('b') {
		t.Fatal("should get b")
	}
	if s.maybe('b') {
		t.Fatal("should not get b twice")
	}
	s.next()
	if !s.eof() {
		t.Fatal("should be at eof")
	}
	if s.maybe('∞') {
		t.Fatal("maybe against eof should always return false")
	}
}

func poseq(a, b token.Position) bool {
	return a.Line == b.Line && a.Rune == b.Rune
}

func TestPosHandling(t *testing.T) {
	s := fakeStream("ab\nc\nd∃f\ng\n")
	poss := []token.Position{
		{Line: 1, Rune: 1}, //a
		{Line: 1, Rune: 2}, //b
		{Line: 2, Rune: 0}, //\n
		{Line: 2, Rune: 1}, //c
		{Line: 3, Rune: 0}, //\n
		{Line: 3, Rune: 1}, //d
		{Line: 3, Rune: 2}, //∃
		{Line: 3, Rune: 3}, //f
		{Line: 4, Rune: 0}, //\n
		{Line: 4, Rune: 1}, //g
		{Line: 5, Rune: 0}, //\n
	}
	bad := func(p token.Position) bool {
		return !poseq(p, s.pos())
	}
	if bad(poss[0]) {
		t.Fatal("wrong out the gate")
	}

	for i, pos := range poss {
		if i == 0 {
			continue
		}

		s.next()
		if bad(pos) {
			t.Fatalf("pos %d invalid, expected %#v got %#v", i, pos, s.pos())
		}
		s.rewind()
		if bad(poss[i-1]) {
			t.Fatalf("failed to rewind properly from pos %d", i)
		}
		s.next()
	}
}

func TestUntil(t *testing.T) {
	front := "ab\nc\nd"
	back := "f\n"
	s := fakeStream(front + "∃" + back + "g")
	s.consume()
	s.until('∃')
	if v := s.value(); v != front {
		t.Fatalf("expected %q got %q", front, v)
	}
	if s.c != '∃' {
		t.Fatalf("should have ∃ got %c", s.c)
	}
	s.until('g')
	if v := s.value(); v != back {
		t.Fatalf("expected %q got %q", back, v)
	}
}
