package virt

import (
	"os"

	"github.com/jimmyfrasche/etlite/internal/device"
	"github.com/jimmyfrasche/etlite/internal/device/std"
	"github.com/jimmyfrasche/etlite/internal/driver"
	"github.com/jimmyfrasche/etlite/internal/format"
	"github.com/jimmyfrasche/etlite/internal/format/rawfmt"
	"github.com/jimmyfrasche/etlite/internal/internal/eol"
	"github.com/jimmyfrasche/etlite/internal/internal/errint"
	"github.com/jimmyfrasche/etlite/internal/virt/internal/sysdb"
)

//Spec specifies the defaults for a Machine.
type Spec struct {
	//Database is the name of the SQLite db file or the empty string
	//for an in-memory database.
	Database string
	//Input is the initial input device, or stdin if nil.
	Input device.Reader
	//Output is the initial output device, or stdout if nil.
	Output device.Writer
	//Encoder is the initial encoder,
	//or a csv encoder with platform specific EOL encoding if nil.
	Encoder format.Encoder
	//Decoder is the intital decoder,
	//or a csv decoder with platform specific EOL encoding if nil.
	Decoder format.Decoder
	//Environ is the environment to use to populate sys.env,
	//or os.Environ if nil.
	Environ []string
	//Args is used to populate sys.arg.
	//There is no default for Args.
	Args []string
}

//Machine represents an execution context for a script.
type Machine struct {
	name    string
	conn    *driver.Conn
	output  device.Writer
	input   device.Reader
	encoder format.Encoder
	decoder format.Decoder

	eframe, dframe string

	sys                        *sysdb.Sysdb
	savepointStmt, releaseStmt *driver.Stmt

	derivedTableName string
}

//New creates and prepares an execution context.
func New(savepoints []string, s Spec) (*Machine, error) {
	db := s.Database
	if db == "" {
		db = ":memory:"
	}
	c, err := driver.Open(db)
	if err != nil {
		return nil, err
	}
	o := s.Output
	if o == nil {
		o = std.Out
	}
	in := s.Input
	if in == nil {
		in = std.In
	}
	e := s.Encoder
	if e == nil {
		e = &rawfmt.Encoder{
			UseCRLF:  eol.Default,
			NoHeader: true,
		}
	}
	dec := s.Decoder
	if dec == nil {
		dec = &rawfmt.Decoder{
			UseCRLF:  eol.Default,
			NoHeader: true,
		}
	}
	m := &Machine{
		name:    db,
		conn:    c,
		output:  o,
		input:   in,
		encoder: e,
		decoder: dec,
	}
	//Init default dec/enc
	if err := m.decoder.Init(m.input); err != nil {
		return nil, err
	}
	if err := m.encoder.Init(m.output); err != nil {
		return nil, err
	}

	if s.Environ == nil {
		s.Environ = os.Environ()
	}

	m.sys, err = sysdb.New(m.conn, s.Args, s.Environ)
	if err != nil {
		return nil, err
	}

	m.savepointStmt, err = m.conn.Prepare(`SAVEPOINT [1]`)
	if err != nil {
		return nil, errint.Wrap(err)
	}
	m.releaseStmt, err = m.conn.Prepare(`RELEASE SAVEPOINT [1]`)
	if err != nil {
		return nil, errint.Wrap(err)
	}

	return m, nil
}

//Close flushes and closes output and cleans up
//all tracked resources associated with the context.
//It does not track resources allocated by Instructions:
//that is the responsibility of an individual Instruction.
func (m *Machine) Close() (errs []error) {
	err := func(err error) {
		if err != nil {
			errs = append(errs, err)
		}
	}
	o := m.output
	err(o.Flush())
	err(o.Close())
	err(m.input.Close())
	err(m.encoder.Close())
	err(m.decoder.Close())
	err(m.sys.Close())
	err(m.savepointStmt.Close())
	err(m.releaseStmt.Close())
	err(m.conn.Close())
	return
}

//Environ dumps sys.env (which the user is free to modify)
//in Go readable format.
func (m *Machine) Environ() ([]string, error) {
	return m.sys.Environ()
}

//Name reports the name of the main database.
func (m *Machine) Name() string {
	return m.name
}

func (m *Machine) setOutput(o device.Writer) error {
	if o == nil {
		return errint.New("no output device specified")
	}
	if m.output == nil {
		return errint.New("no previous output device")
	}
	if m.encoder == nil {
		return errint.New("no previous decoder")
	}

	if err := m.encoder.Close(); err != nil {
		return err
	}

	if err := m.output.Close(); err != nil {
		return err
	}
	m.output = o

	return m.encoder.Init(m.output)
}

func (m *Machine) setInput(in device.Reader, derivedTableName string) error {
	if in == nil {
		return errint.New("no input device specified")
	}
	if m.input == nil {
		return errint.New("no previous input device")
	}
	if m.decoder == nil {
		return errint.New("no previous decoder")
	}

	if err := m.decoder.Close(); err != nil {
		return err
	}

	if err := m.input.Close(); err != nil {
		return err
	}
	m.input, m.derivedTableName = in, derivedTableName

	return m.decoder.Init(m.input)
}

//DerivedTableName returns the derivedTableName from the last call to SetInput.
//
//This is not stored on the stack as multiple imports may read from the same device.
func (m *Machine) DerivedTableName() string {
	return m.derivedTableName
}

func (m *Machine) setDecoder(d format.Decoder) error {
	if d == nil {
		return errint.New("no decoder specified")
	}
	if m.decoder == nil {
		return errint.New("no previous decoder")
	}
	if m.input == nil {
		return errint.New("no previous input device")
	}

	if err := m.decoder.Close(); err != nil {
		return err
	}
	m.decoder = d

	return m.decoder.Init(m.input)
}

func (m *Machine) setEncoder(e format.Encoder) error {
	if e == nil {
		return errint.New("no encoder specified")
	}
	if m.encoder == nil {
		return errint.New("no previous encoder")
	}
	if m.output == nil {
		return errint.New("no previous output device")
	}

	if err := m.encoder.Close(); err != nil {
		return err
	}
	m.encoder = e

	return m.encoder.Init(m.output)
}

//SetDecodingFrame specifies the data frame (table) to decode,
//if applicable to the current format
func (m *Machine) SetDecodingFrame(f string) {
	m.dframe = f
}

//exec q.
func (m *Machine) exec(q string) error {
	s, err := m.conn.Prepare(q)
	if err != nil {
		return err
	}
	defer s.Close()

	return s.Exec()
}

func (m *Machine) savepoint() error {
	return m.savepointStmt.Exec()
}

func (m *Machine) release() error {
	return m.releaseStmt.Exec()
}
