//Package synth provides sql synthesis helpers needed by the compiler and vm.
package synth

import "strings"

type builder struct {
	b []string
}

func build(s ...string) *builder {
	b := &builder{
		b: s,
	}
	return b
}

func (b *builder) push(s ...string) {
	b.b = append(b.b, s...)
}

func (b *builder) csv(values []string, each func(string)) {
	for i, value := range values {
		each(value)
		if i != len(values)-1 {
			b.push(",")
		}
	}
}

func (b *builder) values(hdr []string) {
	b.push("VALUES (")
	b.csv(hdr, func(string) {
		b.push("?")
	})
	b.push(");")
}

func (b *builder) join() string {
	return strings.Join(b.b, " ")
}

//CreateTable synthesizes a create (temporary) table statement
//using the given header.
func CreateTable(temporary bool, name string, header []string) string {
	b := build("CREATE")

	if temporary {
		b.push("TEMPORARY")
	}

	b.push("TABLE", name, "(")

	b.csv(header, func(h string) {
		b.push(h, "TEXT")
	})

	b.push(");")

	return b.join()
}

//Insert synthesizes an insert statement into the table name,
//using the given header and with placeholders for each item
//in the header.
func Insert(name string, header []string) string {
	b := build("INSERT INTO", name, "(")

	b.csv(header, func(h string) {
		b.push(h)
	})

	b.push(")")
	b.values(header)
	return b.join()
}

//Values synthesizes just the VALUES (?, ..., ?) portion of an insert
//statement, following preface, if provided.
func Values(preface string, header []string) string {
	b := build(preface)
	b.values(header)
	return b.join()
}
