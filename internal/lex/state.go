package lex

import (
	"unicode"

	"github.com/jimmyfrasche/etlite/internal/token"
)

//state represents states in the lexer state machine.
type state func(*lexer) state

//run trampolines the state machine.
func run(l *lexer) {
	st := lexMain
	for st != nil {
		st = st(l)
	}
	l.close()
}

//eat all whitespace.
func eat(l *lexer) state {
	l.ignoreUntil(not(space))
	l.discard()
	return lexMain
}

//lineComment is triggered by # or -- and reads till \n.
func lineComment(l *lexer) state {
	l.ignoreUntil(is('\n')) //could end on EOF, but that's okay
	l.discard()
	l.next()
	return lexMain
}

//multilineComment is triggered by /* and reads till */.
//
//There is no nesting.
func multilineComment(l *lexer) state {
	l.ignoreUntil(is('*'))
	if l.eof() {
		return l.error("EOF in /* */ comment")
	}

	if !l.maybe('/') {
		//just a * in the comment, ignore and recurse
		return multilineComment
	}

	//reached */, but read head on /
	l.discard() //maybe will consume
	l.next()
	return lexMain
}

//qstring handles single, double, and backtick quote strings
func qstring(l *lexer) state {
	l.until(l.stringType)
	if l.eof() {
		return l.errorf("EOF in %[1]cstring%[1]c", l.stringType)
	}
	l.consume()
	if l.maybe(l.stringType) {
		//two in a row, keep recursing until we get past this
		return qstring
	}
	l.emitString()
	l.next()
	return lexMain
}

//bstring handles [strings].
func bstring(l *lexer) state {
	l.until(']')
	if l.eof() {
		return l.error("EOF in [string]")
	}
	l.consume()
	l.emitString()
	l.next()
	return lexMain
}

//blob handles blob literals.
func blob(l *lexer) state {
	l.untilp(not(hex))
	if l.c != '\'' {
		return l.error("invalid blob literal")
	}
	l.consume()
	l.emitString()
	l.next()
	return lexMain
}

//argument handles env arguments, @ENV_VAR,
//and command line args, @11.
func argument(l *lexer) state {
	l.next()
	l.discard() //discard @ from buffer, only care about the rest
	if l.eof() {
		return l.error("unexpected EOF: @ with no argument")
	}
	l.consume()

	if digit(l.c) {
		l.untilp(not(digit))
	} else {
		l.untilp(endOfLiteral)
	}

	l.emitK(token.Argument)
	return lexMain
}

func literal(l *lexer) state {
	l.untilp(endOfLiteral)
	l.emitLit()
	return lexMain
}

//number emits literals,
//but it makes sure it doesn't split a number up into multiple tokens
//
//Note however that negative numbers do get split into two tokens
//and we rely on the formatter to avoid emitting a space after operators
func number(l *lexer) state {
	if l.c == '0' && l.maybep(any("xX")) {
		if !l.maybep(hex) {
			return l.error("unexpected end in hex literal")
		}
		l.untilp(not(hex))
		l.emitLit()
		return lexMain
	}

	dotSeen := l.c == '.'
	eSeen := false
	ln := 1
	last := l.c

	for {
		l.next()
		if !floatDigit(l.c) {
			break
		}

		if l.c == 'e' || l.c == 'E' {
			if eSeen {
				return l.errorf("invalid numeric literal: only one %q allowed", l.c)
			}
			eSeen = true
		} else if l.c == '.' {
			if dotSeen {
				return l.error("invalid numeric literal: only one . allowed")
			}
			dotSeen = true
		}
		last = l.c

		l.consume()
	}

	if ln == 1 && last == '.' {
		return l.error("unexpected .")
	}

	if last == 'e' || last == 'E' {
		return l.error("no exponent on number")
	}

	l.emitLit()
	return lexMain
}

func lexMain(l *lexer) state {
	if l.eof() {
		return nil
	}

	if unicode.IsSpace(l.c) {
		return eat
	}

	l.stickyPos()

	switch l.c {
	case ']':
		return l.error("] without [")
	case '$', '?', ':':
		return l.errorf("a '%c' bind is invalid: only @ binds are allowed", l.c)
	case '\\', '^', '{', '}':
		return l.errorf("unrecognized token: '%c'", l.c)
	}

	//already tested for whitespace so l.c < ' ' gets all the weirdos
	if l.c < ' ' {
		return l.errorf("invalid control code 0x%X in input", l.c)
	}

	//we pretty much always want to consume the current rune now
	//so it's easier to discard in the few cases we don't at this point
	l.consume()

	if l.c == '.' || digit(l.c) {
		return number
	}

	switch l.c {
	case ';':
		l.emitK(token.Semicolon)
		l.next()
		return lexMain
	case '(':
		l.emitK(token.LParen)
		l.next()
		return lexMain
	case ')':
		l.emitK(token.RParen)
		l.next()
		return lexMain

	case '%', '&', '+', '~', ',':
		l.emitLit()
		l.next()
		return lexMain

	case '-':
		if l.maybe('-') {
			return lineComment
		}
		l.emitLit()
		l.next()
		return lexMain
	case '#':
		return lineComment

	case '|':
		//note that here and below we're relying on the fact that
		//maybe consumes the input in case of a match
		l.maybe('|')
		l.emitLit()
		l.next()
		return lexMain
	case '=':
		l.maybe('=')
		l.emitLit()
		l.next()
		return lexMain
	case '>':
		l.maybep(any("=>"))
		l.emitLit()
		l.next()
		return lexMain
	case '<':
		l.maybep(any("<=>"))
		l.emitLit()
		l.next()
		return lexMain

	case '!':
		if !l.maybe('=') {
			return l.error("! without =")
		}
		l.emitLit()
		l.next()
		return lexMain

	case '*':
		if l.maybe('/') {
			return l.error("*/ without /*")
		}
		l.emitLit()
		l.next()
		return lexMain
	case '/':
		if l.maybe('*') {
			return multilineComment
		}
		l.emitLit()
		l.next()
		return lexMain

	case '@':
		return argument

	case '\'', '"', '`':
		l.stringType = l.c
		return qstring
	case '[':
		l.stringType = '['
		return bstring
	case 'x', 'X':
		if l.maybe('\'') {
			l.stringType = 'x'
			return blob
		}
		return literal

	}

	//all special cases handled, must be a literal
	return literal
}
