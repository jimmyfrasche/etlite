package format

import "github.com/jimmyfrasche/etlite/internal/device"

//Encoder encodes an SQLite table as text.
//For each table,
//	WriteHeader will be called once.
//	WriteRow will be called 0 or more times.
//	Reset will be called once.
//
//When an Encoder is to be retired Close will be called once.
type Encoder interface {
	//Name reports the name of the format being encoded.
	//It may be called at any time.
	Name() string

	//Init encoder to write to w.
	Init(w device.Writer) error

	//WriteHeader may choose to not write the header, depending on format.
	//
	//If the format requires the name of a data frame to write and none is provided,
	//WriteHeader must return ErrFrameRequired.
	WriteHeader(frame string, header []string) error
	WriteRow([]*string) error
	Reset() error
	Close() error
}
