//Package digital checks whether a string consists of only ASCII digits.
package digital

//String returns true if s consists only of ASCII digits.
func String(s string) bool {
	for i := 0; i < len(s); i++ {
		if b := s[i]; b < '0' || b > '9' {
			return false
		}
	}
	return true
}
