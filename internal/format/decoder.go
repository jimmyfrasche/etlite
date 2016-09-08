package format

import "github.com/jimmyfrasche/etlite/internal/device"

//Decoder specifies the protocol for a format to be imported into a database.
//
//A Decoder will be called with Init to set the input device.
//
//For each table,
//	ReadHeader will be called once.
//	Skip will be called 0 or more times.
//	ReadRow will be called 0 or more times.
//	Reset will be called once.
//
//When a Decoder is retired Close will be called once.
//
//It may be re-initialized later with a different input device.
type Decoder interface {
	//Name reports the name of the format being decoded.
	//It may be called at any time.
	Name() string

	//Init decoder to read from r.
	Init(r device.Reader) error

	//ReadHeader prepares the Decoder to read records from the current device.
	//
	//If ReadHeader is called on a format that contains multiple data frames
	//and no frame is provided, ReadHeader must return ErrFrameRequired.
	//
	//If ReadHeader is called on a format, or instance, that does not contain a header,
	//and no header is provided, ReadHeader must return ErrNoHeader.
	//
	//If a nonempty header is passed to ReadHeader, it must be returned.
	//
	//ReadHeader must never return data and an error.
	//
	//Clients must not assume it is safe to modify the returned slice.
	ReadHeader(frame string, header []string) ([]string, error)

	//Skip rows. Used by OFFSET.
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
