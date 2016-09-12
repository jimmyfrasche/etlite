package lex

import (
	"log"
	"strings"
	"testing"

	"github.com/jimmyfrasche/etlite/internal/token"
)

func fakeLexer(s string) *lexer {
	return &lexer{
		stream: fakeStream(s),
		tokens: make(chan token.Value, 1), //buffer so we don't need goroutine
	}
}

func discard(l *lexer) {
	for range l.tokens {
	}
}

func TestEat(t *testing.T) {
	l := fakeLexer(" \t a")
	eat(l)
	go discard(l)
	if l.c != 'a' {
		t.Fatal("did not eat whitespace")
	}
}

func TestLineComment(t *testing.T) {
	//NB the # is skipped and would work with -
	l := fakeLexer("# this goes to eof")
	go discard(l)
	lineComment(l)
	if !l.eof() {
		t.Fatal("should have lexed all input")
	}

	l = fakeLexer("# this goes to eol\nΣnext line")
	go discard(l)
	lineComment(l)
	if l.c != 'Σ' {
		t.Fatal("did not land on next line")
	}
}

func errtk(t *testing.T, tk token.Value, emsg string) {
	if tk.Kind != token.Illegal {
		t.Fatalf("Expected error got %s:%s", tk.Kind, tk.Value)
	}
	if tk.Err == nil {
		t.Fatal("Expected error message but err == nil")
	}
	if tk.Err.Error() == emsg {
		return
	}
	t.Fatalf("Expected %q got %q", emsg, tk.Err.Error())
}

func TestMultilineComment(t *testing.T) {
	//expects first /* to be consumed already
	l := fakeLexer("  */x")
	go discard(l)
	multilineComment(l)
	if l.c != 'x' {
		t.Fatal("failed to ignore comment")
	}

	l = fakeLexer(" * */x")
	go discard(l)
	//needs to recurse, without trampoline we just call twice
	multilineComment(l)
	multilineComment(l)
	if l.c != 'x' {
		t.Fatal("failed to ignore comment")
	}

	//hits this case two different ways
	for _, in := range []string{"", "*"} {
		l = fakeLexer(in)
		multilineComment(l)
		tk := <-l.tokens
		errtk(t, tk, "EOF in /* */ comment")
	}
}

func chktok(t *testing.T, got token.Value, expk token.Kind, expv string) {
	if got.Kind != expk {
		t.Fatalf("Expected token of type %s got %s", expk, got.Kind)
	}
	if v := got.Value; v != expv {
		t.Fatalf("Expected %q got %q", expv, v)
	}
}

func TestQString(t *testing.T) {
	for i, in := range []string{`"string"`, `"with "" escaped"`} {
		l := fakeLexer(in)
		l.consume() //record "
		l.stringType = '"'
		qstring(l)
		if i == 1 { //need to call twice for manual trampoline
			qstring(l)
		}
		if !l.eof() {
			t.Fatalf("Could not lex %q", in)
		}
		chktok(t, <-l.tokens, token.String, in)
	}

	l := fakeLexer(`"improper string`)
	l.stringType = '"'
	l.consume()
	qstring(l)
	errtk(t, <-l.tokens, `EOF in "string"`)
}

func TestBString(t *testing.T) {
	const valid = "[string]"
	l := fakeLexer(valid)
	l.consume() //record [
	bstring(l)
	if !l.eof() {
		log.Fatalf("failed to lex %s", valid)
	}
	chktok(t, <-l.tokens, token.String, valid)

	const invalid = "[string"
	l = fakeLexer(invalid)
	l.consume() //record [
	bstring(l)
	errtk(t, <-l.tokens, "EOF in [string]")
}

func TestBlob(t *testing.T) {
	const valid = "x'cafebeef21'" //x would be consumed by lexMain
	l := fakeLexer(valid)
	l.consume() //save x
	l.next()
	l.consume() //save '
	blob(l)
	if !l.eof() {
		t.Fatalf("failed to lex %s", valid)
	}
	chktok(t, <-l.tokens, token.String, valid)

	const invalid = "x'cafez'"
	l = fakeLexer(invalid)
	l.consume() //save x
	l.next()
	l.consume() //save '
	blob(l)
	if l.eof() {
		t.Fatalf("failed to not lex %s", invalid)
	}
	errtk(t, <-l.tokens, "invalid blob literal")
}

func TestArgument(t *testing.T) {
	l := fakeLexer("@")
	argument(l)
	errtk(t, <-l.tokens, "unexpected EOF: @ with no argument")

	for _, arg := range []string{"@ENV", "@686", "@A", "@1"} {
		l = fakeLexer(arg)
		argument(l)
		if !l.eof() {
			t.Fatalf("Failed to lex %q", arg)
		}
		chktok(t, <-l.tokens, token.Argument, arg[1:]) //does not save @
	}
}

//There would be a TestLiteral here following the pattern,
//but it's trivial and gets covered elsewhere many times over

func TestNumber(t *testing.T) {
	valid := []string{"1", "23", ".1", ".23", "3.14", "15.1", "15.23", "1e1", "1.e1", "23e23", "0xaF"}
	for _, test := range valid {
		l := fakeLexer(test)
		l.consume()
		number(l)
		if !l.eof() {
			t.Fatalf("failed to lex %s", test)
		}
		chktok(t, <-l.tokens, token.Literal, test)
	}

	invalid := []struct {
		in, err string
	}{
		{"1e", "no exponent on number"},
		{"1e1e1", "invalid numeric literal: only one 'e' allowed"},
		{"1.1.1", "invalid numeric literal: only one . allowed"},
		{"0x", "unexpected end in hex literal"},
	}

	for _, test := range invalid {
		l := fakeLexer(test.in)
		l.consume()
		number(l)
		errtk(t, <-l.tokens, test.err)
	}
}

type mainSimpleTest struct {
	in string
	t  token.Value
}

func mksimple(v string) mainSimpleTest {
	return mainSimpleTest{
		in: v,
		t: token.Value{
			Kind:  token.Literal,
			Value: v,
			Canon: strings.ToUpper(v),
		},
	}
}

var mainSimpleTests = []mainSimpleTest{
	{";", token.Value{Kind: token.Semicolon}},
	{"(", token.Value{Kind: token.LParen}},
	{")", token.Value{Kind: token.RParen}},
	mksimple("%"),
	mksimple("&"),
	mksimple("+"),
	mksimple("~"),
	mksimple(","),
	mksimple("-"),
	mksimple("|"),
	mksimple("||"),
	mksimple(">"),
	mksimple(">>"),
	mksimple(">="),
	mksimple("<"),
	mksimple("<="),
	mksimple("<<"),
	mksimple("<>"),
	mksimple("="),
	mksimple("=="),
	mksimple("*"),
	mksimple("/"),
	mksimple("!="),
}

func TestMainSimpleCase(t *testing.T) {
	l := fakeLexer("")
	lexMain(l)
	if !l.eof() {
		t.Fatal("failed to handle empty input in lexMain")
	}

	for _, test := range mainSimpleTests {
		l := fakeLexer(test.in)
		lexMain(l)
		if tk := <-l.tokens; !test.t.Equal(tk) {
			t.Fatalf("expected %q got %q", test.t.Value, tk.Value)
		}
	}

	invalid := []struct {
		in, err string
	}{
		{"]", "] without ["},
		{"$", "a '$' bind is invalid: only @ binds are allowed"},
		{"^", "unrecognized token: '^'"},
		{"*/", "*/ without /*"},
	}

	for _, test := range invalid {
		l := fakeLexer(test.in)
		lexMain(l)
		errtk(t, <-l.tokens, test.err)
	}
}

func mktk(k token.Kind, v string) token.Value {
	return token.Value{
		Kind:  k,
		Value: v,
	}
}

func mklit(v string) token.Value {
	return mktk(token.Literal, v)
}

func mkarg(v string) token.Value {
	return mktk(token.Argument, v)
}

func mks(v string) token.Value {
	return mktk(token.String, v)
}

const nonsenseSalad = `#!magic
SeLeCt(%)[+]  	"a
("" !",.5+<<>>  from, /*
	* */;4e1 FRO x'00' --toeol
xciting@arg
@1harumph		===`

var salad = []token.Value{
	mklit("SeLeCt"),
	mktk(token.LParen, ""),
	mklit("%"),
	mktk(token.RParen, ""),
	mks("[+]"),
	mks(`"a
("" !"`),
	mklit(","),
	mklit(".5"),
	mklit("+"),
	mklit("<<"),
	mklit(">>"),
	mklit("from"),
	mklit(","),
	mktk(token.Semicolon, ""),
	mklit("4e1"),
	mklit("FRO"),
	mks("x'00'"),
	mklit("xciting"),
	mkarg("arg"),
	mkarg("1"),
	mklit("harumph"),
	mklit("=="),
	mklit("="),
}

func TestMainIntegration(t *testing.T) {
	i := 0
	for tk := range Stream("test", strings.NewReader(nonsenseSalad)) {
		e := salad[i]
		chktok(t, tk, e.Kind, e.Value)
		i++
	}
}
