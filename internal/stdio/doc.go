//Package stdio wraps stdin, stdout, and file operations with the semantics
//required elsewhere in this system and that work around any consequences
//of the sys.env table being user-writable.
package stdio

import (
	"bufio"
	"io"
)

//Reader is the interface provided by all readers in this package.
type Reader interface {
	io.ReadCloser
	//Unwrap returns the underlying bufio.Reader for external packages
	//that need direct access.
	Unwrap() *bufio.Reader
	//Name returns the name of the file being read or - for stdin.
	Name() string
}

//Writer is the interface provided by all writers in this package.
type Writer interface {
	io.WriteCloser
	Flush() error
	WriteString(string) (int, error)
	//Unwrap returns the underlying bufio.Writer for external packages
	//that need direct access.
	Unwrap() *bufio.Writer
	//Name returns the target name of the file or - for stdout.
	Name() string
}
