package virt

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/jimmyfrasche/etlite/internal/device/file"
	"github.com/jimmyfrasche/etlite/internal/device/std"
	"github.com/jimmyfrasche/etlite/internal/format"
	"github.com/jimmyfrasche/etlite/internal/internal/escape"
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
	return fmt.Sprintf("%s: assertion failure: %s", a.pos, a.msg)
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

func MkSetEncoder(e format.Encoder) Instruction {
	return func(m *Machine) error {
		return m.setEncoder(e)
	}
}

func MkSetDecoder(d format.Decoder) Instruction {
	return func(m *Machine) error {
		return m.setDecoder(d)
	}
}

//MkSetEncodingFrame specifies the data frame (table) to encode,
//if applicable to the current format.
func MkSetEncodingFrame(f string) Instruction {
	return func(m *Machine) error {
		m.eframe = f
		return nil
	}
}

func MkUseStdout() Instruction {
	return func(m *Machine) error {
		return m.setOutput(std.Out)
	}
}

func MkUseStdin() Instruction {
	return func(m *Machine) error {
		return m.setInput(std.In, "[-]")
	}
}

func MkUseFileOutput(fname string) Instruction {
	return func(m *Machine) error {
		f, err := file.NewWriter(fname)
		if err != nil {
			return err
		}
		return m.setOutput(f)
	}
}

func MkUseFileInput(fname string) Instruction {
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

func MkDropTempTables(names []string) Instruction {
	return func(m *Machine) error {
		for _, name := range names {
			if err := m.exec("DROP TABLE temp." + name); err != nil {
				return err
			}
		}
		return nil
	}
}
