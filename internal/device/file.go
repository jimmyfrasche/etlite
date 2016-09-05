package device

import (
	"bufio"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/jimmyfrasche/etlite/internal/internal/errsys"
)

//FileReader is a file used only for reading
//that satisfies this packages Reader interface.
type FileReader struct {
	name string
	f    *os.File
	*bufio.Reader
}

var _ Reader = (*FileReader)(nil)

//NewFileReader attempts to open a file for reading.
func NewFileReader(name string) (*FileReader, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, errsys.Wrap(err)
	}
	s, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, errsys.Wrap(err)
	}
	if s.IsDir() {
		_ = f.Close()
		return nil, errsys.Newf("%s is a directory", name)
	}
	return &FileReader{
		name:   name,
		f:      f,
		Reader: bufio.NewReader(f),
	}, nil
}

//Name returns the file being read.
func (f *FileReader) Name() string {
	return f.name
}

//Unwrap returns the underlying bufio.Reader of this file.
func (f *FileReader) Unwrap() *bufio.Reader {
	return f.Reader
}

//Close f.
func (f *FileReader) Close() error {
	err := errsys.Wrap(f.Close())
	f.name = "<BROKEN FILE HANDLE>"
	f.f = nil
	f.Reader.Reset(nil)
	f.Reader = nil
	return err
}

//FileWriter represents a file used for writing.
//
//It is written to a tmp file and renamed on Close.
type FileWriter struct {
	name string
	f    *os.File //the tmp file
	*bufio.Writer
}

var _ Writer = (*FileWriter)(nil)

//NewFileWriter creates a temporary file to write to and replaces name on Close.
//
//tmpdirFromEnv is the result of reading TEMP from sys.env.
func NewFileWriter(name string) (*FileWriter, error) {
	f, err := ioutil.TempFile(filepath.Split(name))
	if err != nil {
		return nil, errsys.Wrap(err)
	}
	return &FileWriter{
		name:   name,
		f:      f,
		Writer: bufio.NewWriter(f),
	}, nil
}

//Name reports the name the file will have when closed.
func (f *FileWriter) Name() string {
	return f.name
}

//Unwrap returns the underlying bufio.Writer.
func (f *FileWriter) Unwrap() *bufio.Writer {
	return f.Writer
}

//Close flushes, syncs, and renames the tmp file to Name().
func (f *FileWriter) Close() error {
	//even if something fails we need to break the file handle to prevent
	//undetected erroneous state.
	defer func() {
		f.Writer.Reset(nil)
		f.Writer = nil
		f.f = nil
		f.name = "<BROKEN FILE HANDLE>"
	}()

	//flush out and get rid of the buffer
	if err := f.Flush(); err != nil {
		_ = f.f.Close()
		return errsys.Wrap(err)
	}

	//sync to the disk, even though is probably a lie
	if err := f.f.Sync(); err != nil {
		_ = f.f.Close()
		return errsys.Wrap(err)
	}

	//this is the file we've actually been writing to.
	tmpnm := f.f.Name()

	//get rid of the file handle
	if err := f.f.Close(); err != nil {
		return errsys.Wrap(err)
	}

	//attempt to rename
	if err := os.Rename(tmpnm, f.name); err != nil {
		return errsys.Wrap(err)
	}

	return nil
}
