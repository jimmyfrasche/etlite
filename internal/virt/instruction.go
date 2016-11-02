package virt

import (
	"context"
	"fmt"

	"github.com/jimmyfrasche/etlite/internal/device/file"
	"github.com/jimmyfrasche/etlite/internal/device/std"
	"github.com/jimmyfrasche/etlite/internal/format"
	"github.com/jimmyfrasche/etlite/internal/internal/errint"
	"github.com/jimmyfrasche/etlite/internal/internal/errusr"
	"github.com/jimmyfrasche/etlite/internal/token"
)

//An Instruction is a single instruction in the VM.
type Instruction func(context.Context, *Machine) error

//Run executes all is instructions in execution context m.
func (m *Machine) Run(ctx context.Context, is []Instruction) error {
	var err error
loop:
	for _, i := range is {
		if err = i(ctx, m); err != nil {
			break
		}
		select {
		case <-ctx.Done():
			err = ctx.Err()
			break loop
		default:
		}
	}
	if err != nil {
		if m.stack.Open() {
			_ = m.drain(true)
			//TODO handle SQLITE_BUSY somewhere
			_ = m.exec("ROLLBACK;")
		}
		return errusr.Wrap(m.pos, err)
	}
	return m.drain(false)
}

//Assert returns an assertion.
func Assert(pos token.Poser, msg, query string) Instruction {
	return func(ctx context.Context, m *Machine) error {
		ret, err := m.conn.Assert(query)
		if err != nil {
			return err
		}
		if !ret {
			return fmt.Errorf("assertion failure: %s", msg)
		}
		return nil
	}
}

func ErrPos(p token.Poser) Instruction {
	return func(ctx context.Context, m *Machine) error {
		m.pos = p.Pos()
		return nil
	}
}

func SetEncoder(e format.Encoder) Instruction {
	return func(ctx context.Context, m *Machine) error {
		return m.setEncoder(e)
	}
}

func SetDecoder(d format.Decoder) Instruction {
	return func(ctx context.Context, m *Machine) error {
		return m.setDecoder(d)
	}
}

//SetEncodingFrame specifies the data frame (table) to encode,
//if applicable to the current format.
func SetEncodingFrame(f string) Instruction {
	return func(ctx context.Context, m *Machine) error {
		m.eframe = f
		return nil
	}
}

func UseStdout() Instruction {
	return func(ctx context.Context, m *Machine) error {
		return m.setOutput(std.Out)
	}
}

func UseStdin() Instruction {
	return func(ctx context.Context, m *Machine) error {
		return m.setInput(std.In)
	}
}

func UseFileOutput(fname string) Instruction {
	return func(ctx context.Context, m *Machine) error {
		f, err := file.NewWriter(fname)
		if err != nil {
			return err
		}
		return m.setOutput(f)
	}
}

func UseFileInput(fname string) Instruction {
	return func(ctx context.Context, m *Machine) error {
		f, err := file.NewReader(fname)
		if err != nil {
			return err
		}
		return m.setInput(f)
	}
}

func Savepoint() Instruction {
	return func(ctx context.Context, m *Machine) error {
		m.stack.Savepoint("1")
		return m.savepointStmt.Exec()
	}
}

func Release() Instruction {
	return func(ctx context.Context, m *Machine) error {
		if err := m.releaseStmt.Exec(); err != nil {
			return err
		}
		m.stack.Release("1")
		return nil
	}
}

func DropTempTables(names []string) Instruction {
	return func(ctx context.Context, m *Machine) error {
		for _, name := range names {
			if err := m.exec("DROP TABLE temp." + name); err != nil {
				return err
			}
		}
		return nil
	}
}

func Exec(q string) Instruction {
	return func(ctx context.Context, m *Machine) error {
		return m.exec(q) //TODO fastpath this in driver
	}
}

func BeginTransaction(q string) Instruction {
	return func(ctx context.Context, m *Machine) error {
		if err := m.stack.Begin(); err != nil {
			return errint.Wrap(err)
		}
		return m.exec(q)
	}
}

func CommitTransaction(q string) Instruction {
	return func(ctx context.Context, m *Machine) error {
		if err := m.stack.End(); err != nil {
			return errint.Wrap(err)
		}
		if err := m.drain(false); err != nil {
			return err
		}
		return m.exec(q)
	}
}

func UserSavepoint(name, q string) Instruction {
	return func(ctx context.Context, m *Machine) error {
		m.stack.Savepoint(name)
		return m.exec(q)
	}
}

func UserRelease(name, q string) Instruction {
	return func(ctx context.Context, m *Machine) error {
		if err := m.stack.Release(name); err != nil {
			return errint.Wrap(err)
		}
		if err := m.drain(false); err != nil {
			return err
		}
		return m.exec(q)
	}
}
