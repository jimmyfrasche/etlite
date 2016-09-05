package engine

import (
	"github.com/jimmyfrasche/etlite/internal/format"
	"github.com/jimmyfrasche/etlite/internal/format/csvfmt"
	"github.com/jimmyfrasche/etlite/internal/internal/eol"
	"github.com/jimmyfrasche/etlite/internal/internal/errint"
	"github.com/jimmyfrasche/etlite/internal/internal/null"
	"github.com/jimmyfrasche/etlite/internal/sqlite/driver"
	"github.com/jimmyfrasche/etlite/internal/stdio"
)

//Spec specifies the defaults for a Machine.
type Spec struct {
	//Database is the name of the SQLite db file or the empty string
	//for an in-memory database.
	Database string
	//Input is the initial input device, or stdin if nil.
	Input stdio.Reader
	//Output is the initial output device, or stdout if nil.
	Output stdio.Writer
	//Encoder is the initial encoder,
	//or a csv encoder with platform specific EOL encoding if nil.
	Encoder format.Encoder
	//Decoder is the intital decoder,
	//or a csv decoder with platform specific EOL encoding if nil.
	Decoder format.Decoder
	//Environ is the environment to use to populate sys.env,
	//or os.Environ if nil.
	Environ map[string]string
	//Args is used to populate sys.arg.
	//There is no default for Args.
	Args []string
}

//Machine represents an execution context for a script.
type Machine struct {
	name       string
	conn       *driver.Conn
	output     stdio.Writer
	input      stdio.Reader
	encoder    format.Encoder
	decoder    format.Decoder
	saveprefix string

	//number of temporary save point names used
	tmpSP int
	//number of temporary tables used
	tmpT int

	//temporary tables created internally,
	//not at users behest
	tmpTables map[string]bool

	readEnv, tables, tmpFromEnv *driver.Stmt
	savepointStmt, releaseStmt  *driver.Stmt

	stack            []interface{}
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
		o = stdio.Stdout
	}
	in := s.Input
	if in == nil {
		in = stdio.Stdin
	}
	e := s.Encoder
	if e == nil {
		e = &csvfmt.Encoder{UseCRLF: eol.Default}
	}
	dec := s.Decoder
	if dec == nil {
		dec = &csvfmt.Decoder{}
	}
	saveprefix := uniqIdent("-internal-savepoint-", savepoints, map[string]bool{}) //TODO rip this stuff out
	m := &Machine{
		name:       db,
		conn:       c,
		output:     o,
		input:      in,
		encoder:    e,
		decoder:    dec,
		saveprefix: saveprefix,
		tmpTables:  map[string]bool{},
		stack:      make([]interface{}, 0, 128),
	}

	if err = m.createSystab(s.Environ, s.Args); err != nil {
		m.conn.Close()
		return nil, err
	}

	m.tables, err = m.conn.Prepare(`SELECT name FROM sqlite_master WHERE type = 'table'
UNION ALL
SELECT name FROM sqlite_temp_master WHERE type = 'table'`) //TODO rip this stuff out
	if err != nil {
		return nil, errint.Wrap(err)
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
	err(m.readEnv.Close())
	err(m.tables.Close())
	err(m.savepointStmt.Close())
	err(m.releaseStmt.Close())
	err(m.conn.Close())
	return
}

//Name reports the name of the main database.
func (m *Machine) Name() string {
	return m.name
}

func (m *Machine) push(what interface{}) {
	m.stack = append(m.stack, what)
}

func (m *Machine) pop() (val interface{}, err error) {
	if len(m.stack) == 0 {
		return nil, errint.New("stack underflow")
	}
	end := len(m.stack) - 1
	val, m.stack[end] = m.stack[end], nil
	m.stack = m.stack[:end]
	return val, nil
}

//PopString pops a *string off the stack.
func (m *Machine) PopString() (*string, error) {
	v, err := m.pop()
	if err != nil {
		return nil, err
	}
	s, ok := v.(*string)
	if !ok {
		return nil, errint.Newf("expected *string on stack but found %#v", v)
	}
	return s, nil
}

//PopStrings pops a []string off the stack.
func (m *Machine) PopStrings() ([]string, error) {
	v, err := m.pop()
	if err != nil {
		return nil, err
	}
	s, ok := v.([]string)
	if !ok {
		return nil, errint.Newf("expected []string on stack but found %#v", v)
	}
	return s, nil
}

//PopNullEncoding pops a null.Encoding off the stack.
func (m *Machine) PopNullEncoding() (null.Encoding, error) {
	v, err := m.pop()
	if err != nil {
		return "", err
	}
	n, ok := v.(null.Encoding)
	if !ok {
		return "", errint.Newf("expected null encoding on stack but found %#v", v)
	}
	return n, nil
}

func (m *Machine) PopBool() (bool, error) {
	v, err := m.pop()
	if err != nil {
		return false, err
	}
	b, ok := v.(bool)
	if !ok {
		return false, errint.Newf("expected bool on stack but found %#v", v)
	}
	return b, nil
}

func (m *Machine) PopInt() (int, error) {
	v, err := m.pop()
	if err != nil {
		return 0, err
	}
	i, ok := v.(int)
	if !ok {
		return 0, errint.Newf("expected int on stack but found %#v", v)
	}
	return i, nil
}

func (m *Machine) PopRune() (rune, error) {
	v, err := m.pop()
	if err != nil {
		return 0, err
	}
	r, ok := v.(rune)
	if !ok {
		return 0, errint.Newf("expected rune on stack but found %#v", v)
	}
	return r, nil
}

func (m *Machine) SetOutput(o stdio.Writer) error {
	if o == nil {
		return errint.New("no output device specified")
	}
	if m.output == nil {
		return errint.New("no previous output device")
	}
	if err := m.output.Close(); err != nil {
		return err
	}

	m.output = o
	return nil
}

func (m *Machine) SetInput(in stdio.Reader, derivedTableName string) error {
	if in == nil {
		return errint.New("no input device specified")
	}
	if m.input == nil {
		return errint.New("no previous input device")
	}
	if err := m.input.Close(); err != nil {
		return err
	}

	m.input, m.derivedTableName = in, derivedTableName
	return nil
}

//DerivedTableName returns the derivedTableName from the last call to SetInput.
func (m *Machine) DerivedTableName() string { //TODO need to factor into importer
	return m.derivedTableName
}

func (m *Machine) SetDecoder(d format.Decoder) error {
	if d == nil {
		return errint.New("no decoder specified")
	}
	if m.decoder == nil {
		return errint.New("no previous decoder")
	}
	if err := m.decoder.Close(); err != nil {
		return err
	}
	m.decoder = d
	return nil
}

func (m *Machine) SetEncoder(e format.Encoder) error {
	if e == nil {
		return errint.New("no encoder specified")
	}
	if m.encoder == nil {
		return errint.New("no previous encoder")
	}
	if err := m.encoder.Close(); err != nil {
		return err
	}
	m.encoder = e
	return nil
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

func (m *Machine) subquery(q string) (*string, error) {
	s, err := m.conn.Prepare(q)
	if err != nil {
		return nil, err
	}
	defer s.Close()

	return s.Subquery()
}
