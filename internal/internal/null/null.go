//Package null handles marshalling of SQL NULL to and from plain text
//for formats that do not have a native definition equivalent to NULL.
package null

//Encoding defines how SQL NULL is encoded in text.
type Encoding string

//Encode returns nil if s == n and n is not the empty string.
func (n Encoding) Encode(s string) *string {
	if n != "" && s == string(n) {
		return nil
	}
	return &s
}

//Decode returns n if s is nil.
func (n Encoding) Decode(s *string) string {
	if s == nil {
		return string(n)
	}
	return *s
}
