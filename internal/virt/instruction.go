package virt

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/jimmyfrasche/etlite/internal/device/file"
	"github.com/jimmyfrasche/etlite/internal/device/std"
	"github.com/jimmyfrasche/etlite/internal/format"
	"github.com/jimmyfrasche/etlite/internal/internal/errint"
	"github.com/jimmyfrasche/etlite/internal/internal/errusr"
	"github.com/jimmyfrasche/etlite/internal/internal/escape"
	"github.com/jimmyfrasche/etlite/internal/token"
)

//An Instruction is a single instruction in the VM.
type Instruction func(*Machine) error

//Run executes all is instructions in execution context m.
func (m *Machine) Run(is []Instruction) error {
	for _, i := range is {
		if err := i(m); err != nil {
			if m.stack.Open() {
				//TODO handle SQLITE_BUSY somewhere
				_ = m.exec("ROLLBACK;")
			}
			return errusr.Wrap(m.pos, err)
		}
	}
	return nil
}

type assertionError struct {
	pos token.Position
	msg string
}

func (a assertionError) Error() string {
	return fmt.Sprintf("%s: assertion failure: %s", a.pos, a.msg)
}

//Assert returns an assertion.
func Assert(pos token.Position, msg, query string) Instruction {
	return func(m *Machine) error {
		ret, err := m.conn.Assert(query)
		if err != nil {
			return err
		}
		if !ret {
			return assertionError{
				pos: pos,
				msg: msg,
			}
		}
		return nil
	}
}

func ErrPos(p token.Position) Instruction {
	return func(m *Machine) error {
		m.pos = p
		return nil
	}
}

func SetEncoder(e format.Encoder) Instruction {
	return func(m *Machine) error {
		return m.setEncoder(e)
	}
}

func SetDecoder(d format.Decoder) Instruction {
	return func(m *Machine) error {
		return m.setDecoder(d)
	}
}

//SetEncodingFrame specifies the data frame (table) to encode,
//if applicable to the current format.
func SetEncodingFrame(f string) Instruction {
	return func(m *Machine) error {
		m.eframe = f
		return nil
	}
}

func UseStdout() Instruction {
	return func(m *Machine) error {
		return m.setOutput(std.Out)
	}
}

func UseStdin() Instruction {
	return func(m *Machine) error {
		return m.setInput(std.In, "[-]")
	}
}

func UseFileOutput(fname string) Instruction {
	return func(m *Machine) error {
		f, err := file.NewWriter(fname)
		if err != nil {
			return err
		}
		return m.setOutput(f)
	}
}

func UseFileInput(fname string) Instruction {
	return func(m *Machine) error {
		f, err := file.NewReader(fname)
		if err != nil {
			return err
		}
		base := filepath.Base(fname)
		idx := strings.LastIndexByte(base, '.')
		switch {
		case idx < 0:
			// filename
		case idx == 0:
			// .filename
			base = base[1:]
		case idx > 0:
			// filename.ext
			base = base[:idx]
		}
		return m.setInput(f, escape.String(base))
	}
}

func Savepoint() Instruction {
	return func(m *Machine) error {
		return m.savepointStmt.Exec()
	}
}

func Release() Instruction {
	return func(m *Machine) error {
		return m.releaseStmt.Exec()
	}
}

func DropTempTables(names []string) Instruction {
	return func(m *Machine) error {
		for _, name := range names {
			if err := m.exec("DROP TABLE temp." + name); err != nil {
				return err
			}
		}
		return nil
	}
}

func Exec(q string) Instruction {
	return func(m *Machine) error {
		return m.exec(q) //TODO fastpath this in driver
	}
}

func BeginTransaction(q string) Instruction {
	return func(m *Machine) error {
		if err := m.stack.Begin(); err != nil {
			return errint.Wrap(err)
		}
		return m.exec(q)
	}
}

func CommitTransaction(q string) Instruction {
	return func(m *Machine) error {
		if err := m.stack.End(); err != nil {
			return errint.Wrap(err)
		}
		return m.exec(q)
	}
}

func UserSavepoint(name, q string) Instruction {
	return func(m *Machine) error {
		m.stack.Savepoint(name)
		return m.exec(q)
	}
}

func UserRelease(name, q string) Instruction {
	return func(m *Machine) error {
		if err := m.stack.Release(name); err != nil {
			return errint.Wrap(err)
		}
		return m.exec(q)
	}
}
