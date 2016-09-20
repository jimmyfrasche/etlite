package virt

import (
	"github.com/jimmyfrasche/etlite/internal/device"
	"github.com/jimmyfrasche/etlite/internal/device/std"
	"github.com/jimmyfrasche/etlite/internal/driver"
	"github.com/jimmyfrasche/etlite/internal/format"
	"github.com/jimmyfrasche/etlite/internal/format/rawfmt"
	"github.com/jimmyfrasche/etlite/internal/internal/eol"
	"github.com/jimmyfrasche/etlite/internal/internal/errint"
	"github.com/jimmyfrasche/etlite/internal/internal/savepoint"
	"github.com/jimmyfrasche/etlite/internal/token"
	"github.com/jimmyfrasche/etlite/internal/virt/internal/sysdb"
)

//Machine represents an execution context for a script.
type Machine struct {
	name    string
	conn    *driver.Conn
	output  device.Writer
	input   device.Reader
	encoder format.Encoder
	decoder format.Decoder

	sys                        *sysdb.Sysdb
	savepointStmt, releaseStmt *driver.Stmt

	eframe, derivedTableName string

	stack *savepoint.Stack
	pos   token.Position
}

//New creates and prepares an execution context.
func New(db string, args, env []string) (*Machine, error) {
	if db == "" {
		db = ":memory:"
	}
	c, err := driver.Open(db)
	if err != nil {
		return nil, err
	}
	m := &Machine{
		name:   db,
		conn:   c,
		output: std.Out,
		input:  std.In,
		encoder: &rawfmt.Encoder{
			UseCRLF:  eol.Default,
			NoHeader: true,
		},
		decoder: &rawfmt.Decoder{
			UseCRLF:  eol.Default,
			NoHeader: true,
		},
		derivedTableName: "[-]",
		stack:            savepoint.New(),
	}
	//Init default dec/enc
	if err := m.decoder.Init(m.input); err != nil {
		return nil, err
	}
	if err := m.encoder.Init(m.output); err != nil {
		return nil, err
	}

	m.sys, err = sysdb.New(m.conn, args, env)
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
