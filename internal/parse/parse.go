package parse

import (
	"io"

	"github.com/jimmyfrasche/etlite/internal/ast"
	"github.com/jimmyfrasche/etlite/internal/token"
)

//Tokens converts a stream of tokens into a stream of Nodes.
func Tokens(from <-chan token.Value) (to <-chan ast.Node) {
	out := make(chan ast.Node)
	p := newParser(from, out)
	go p.parse()
	return out
}

type parser struct {
	in   <-chan token.Value
	out  chan<- ast.Node
	last token.Value
}

func newParser(tokens <-chan token.Value, nodes chan<- ast.Node) *parser {
	p := &parser{
		in:  tokens,
		out: nodes,
	}
	return p
}

func (p *parser) parse() {
	defer func() {
		if x := recover(); x != nil {
			if err, ok := x.(error); ok && err != io.EOF { //EOF is done so we skip
				astErr, ok := x.(*ast.Error)
				if !ok {
					panic(x) //not an ast error, pass it along
				}
				p.out <- astErr //send ast error before closing
			}
		}
		close(p.out)
	}()

	for {
		t, ok := <-p.in
		if !ok {
			return
		}
		p.last = t
		p.out <- p.parseETL(t)
	}
}

func (p *parser) next() token.Value {
	t, ok := <-p.in
	if !ok {
		panic(&ast.Error{
			Token: p.last,
			Err:   io.ErrUnexpectedEOF,
		})
	}
	p.last = t
	if !t.Valid() {
		panic(&ast.Error{
			Token: t,
		})
	}
	return t
}

func (p *parser) expect(k token.Kind) token.Value {
	t := p.next()
	if t.Kind == k {
		return t
	}
	panic(p.unexpected(t))
}

func (p *parser) expectLit(s string) token.Value {
	t := p.expect(token.Literal)
	if !t.Literal(s) {
		panic(p.unexpected(t))
	}
	return t
}

func (p *parser) cantBe(ks ...token.Kind) token.Value {
	t := p.next()
	for _, k := range ks {
		if t.Kind == k {
			panic(p.unexpected(t))
		}
	}
	return t
}
