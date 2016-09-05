//Package escape escapes strings for use by Sqlite.
package escape

import "strings"

//String returns s as a valid sqlite single-quoted string.
func String(s string) string {
	return "'" + strings.Replace(s, "'", "''", -1) + "'"
}
