//Package stdio wraps stdin, stdout, and file operations with the semantics
//required elsewhere in this system and that work around any consequences
//of the sys.env table being user-writable.
package device

import (
	"bufio"
	"io"
	"os"
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

//File is a device backed by an os.File that allows access directly
//to the file handle.
//
//When done with the os.File, users of this interface must ensure the file
//is in a good state for future reading and writing and then call the returned
//reset function that ensures the device can function as a device.
type File interface {
	File() (f *os.File, reset func(), err error)
}
