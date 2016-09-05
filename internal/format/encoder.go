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
	//WriteHeader may choose to not write the header, depending on format.
	WriteHeader([]string, device.Writer) error
	WriteRow([]*string) error
	Reset() error
	Close() error
}
