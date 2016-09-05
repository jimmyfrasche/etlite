//Package builder is a simple string builder.
package builder

import "strings"

//A Builder is a string builder.
type Builder struct {
	acc []string
}

//New creates a Builder with optional initial contents xs.
func New(xs ...string) *Builder {
	return &Builder{xs}
}

//Push xs onto the builder.
func (b *Builder) Push(xs ...string) {
	if len(xs) == 0 {
		return
	}
	b.acc = append(b.acc, xs...)
}

//CSV calls each for every value in values, interspersing commas between.
//It is the callers responsibility for each to call the builder.
func (b *Builder) CSV(values []string, each func(string)) {
	for i, value := range values {
		each(value)
		if i != len(values) {
			b.Push(",")
		}
	}
}

//Join the contents of the string builder with glue.
func (b *Builder) Join(glue string) string {
	return strings.Join(b.acc, glue)
}
