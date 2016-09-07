package format

import "github.com/jimmyfrasche/etlite/internal/device"

//Decoder specifies the protocol for a format to be imported.
//
//For each table,
//	ReadHeader will be called once.
//	ReadRow will be called 0 or more times.
//	Reset will be called once.
//
//When a Decoder is retired Close will be called once.
type Decoder interface {
	//ReadHeader will not be called if NoHeader is set in the ImportSpec.
	//
	//If ReadHeader is called on a format, or instance, that does not contain a header,
	//and no header is provided, ReadHeader must return ErrNoHeader.
	//
	//If a nonempty header is passed to ReadHeader, it must be returned.
	//
	//If the first return is not the empty string, it may be used as the name of the table.
	//
	//ReadHeader must never return data and an error.
	//
	//Clients must not assume it is safe to modify the returned slice.
	ReadHeader(header []string, r device.Reader) (string, []string, error) //TODO need to pass in table name in case format contains more than one table

	Skip(rows int) error

	//ReadRow will be called until it returns io.EOF
	//
	//ReadRow must never return data and an error.
	//
	//Clients must not assume it is safe to modify the returned slice.
	ReadRow() ([]*string, error)

	//Reset is called after an import.
	//
	//The next call will either be to Init or Close.
	Reset() error

	//Close is called when the Decoder will never be used again.
	Close() error
}
