package lex

import "unicode"

type pred func(rune) bool

const eof rune = -1

func space(r rune) bool {
	return unicode.IsSpace(r)
}

//inRange returns a pred that reports whether a <= r <= b.
func inRange(a, b rune) pred {
	return func(r rune) bool {
		return a <= r && r <= b
	}
}

func is(r rune) pred {
	return func(s rune) bool {
		return s == r
	}
}

func not(p pred) pred {
	return func(r rune) bool {
		return !p(r)
	}
}

func or(ps ...pred) pred {
	return func(r rune) bool {
		for _, p := range ps {
			if p(r) {
				return true
			}
		}
		return false
	}
}

func any(runes string) pred {
	return func(r rune) bool {
		for _, rune := range runes {
			if rune == r {
				return true
			}
		}
		return false
	}
}

var (
	reserved     = any("`" + `|/-+%~[]'"<>!=@$?;.()&{}!^:\,`)
	endOfLiteral = or(space, reserved, inRange(0, ' '))
	digit        = inRange('0', '9')
	hex          = or(digit, inRange('a', 'f'), inRange('A', 'F'))
	floatDigit   = or(digit, any("eE."))
)
