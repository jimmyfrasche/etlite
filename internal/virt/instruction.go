package virt

import (
	"fmt"

	"github.com/jimmyfrasche/etlite/internal/internal/errusr"
	"github.com/jimmyfrasche/etlite/internal/token"
)

//An Instruction is a single instruction in the VM.
type Instruction func(*Machine) error

//Run executes all is instructions in execution context m.
func (m *Machine) Run(is []Instruction) error {
	for _, i := range is {
		if err := i(m); err != nil {
			return err
		}
	}
	return nil
}

type assertionError struct {
	pos token.Position
	msg string
}

func (a assertionError) Error() string {
	return fmt.Sprintf("%s: assertion error: %s", a.pos, a.msg)
}

//MkAssert returns an assertion.
func MkAssert(pos token.Position, msg, query string) Instruction {
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

//MkPush returns an instruction that pushes what onto the stack when executed.
func MkPush(what interface{}) Instruction {
	return func(m *Machine) error {
		m.push(what)
		return nil
	}
}

//MkPushSubquery creates an instruction that executes the subquery sq.
//If handle is non-nil, it's used to validate and/or transform the result.
//Then the result of sq is pushed onto the stack.
func MkPushSubquery(sq string, handle func(*string) (interface{}, error)) Instruction {
	return func(m *Machine) error {
		ret, err := m.subquery(sq)
		if err != nil {
			return err
		}
		if handle != nil {
			what, err := handle(ret)
			if err != nil {
				return err
			}
			m.push(what)
			return nil
		}
		m.push(ret)
		return nil
	}
}

func MkSavepoint() Instruction {
	return func(m *Machine) error {
		return m.savepointStmt.Exec()
	}
}

func MkRelease() Instruction {
	return func(m *Machine) error {
		return m.releaseStmt.Exec()
	}
}

//MkCreateTableFrom creates table name from ddl and pushes the header of the table
//on the stack.
//
//See CreateTable and BulkInsert methods.
func MkCreateTableFrom(pos token.Position, name, ddl string) Instruction {
	return func(m *Machine) error {
		if err := m.exec(ddl); err != nil {
			return errusr.Wrap(pos, err)
		}

		p, err := m.conn.Prepare("SELECT * FROM " + name)
		if err != nil {
			return err
		}
		m.push(p.Columns())
		return p.Close()
	}
}

func MkDropTempTable(name string) Instruction {
	return func(m *Machine) error {
		return m.exec("DROP TABLE temp." + name)
	}
}
