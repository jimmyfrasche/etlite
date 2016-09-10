package compile

import (
	"path/filepath"
	"strings"

	"github.com/jimmyfrasche/etlite/internal/ast"
	"github.com/jimmyfrasche/etlite/internal/device/file"
	"github.com/jimmyfrasche/etlite/internal/device/std"
	"github.com/jimmyfrasche/etlite/internal/internal/errint"
	"github.com/jimmyfrasche/etlite/internal/internal/escape"
	"github.com/jimmyfrasche/etlite/internal/virt"
)

const (
	inputDevice  = true
	outputDevice = false
)

func (c *compiler) compileDevice(d ast.Device, read bool) {
	if d == nil {
		return
	}

	switch d := d.(type) {
	default:
		panic(errint.Newf("unrecognized Device type: %T", d))

	case *ast.DeviceStdio: //never pushes
		if read {
			c.push(setStdin)
		} else {
			c.push(setStdout)
		}

	case *ast.DeviceFile: //always pushes filename
		name, ok := d.Name.Unescape()
		if !ok {
			panic(errint.Newf("file device name must be literal or string got %s", d.Name.Kind))
		}
		if read {
			tbl := tblnameFromFilename(name)
			c.push(func(m *virt.Machine) error {
				f, err := file.NewReader(name)
				if err != nil {
					return err
				}
				//table name has to go in a register instead of on the stack
				//due to the evaluation order
				return m.SetInput(f, tbl)
			})
		} else {
			c.push(func(m *virt.Machine) error {
				f, err := file.NewWriter(name)
				if err != nil {
					return err
				}
				return m.SetOutput(f)
			})
		}
	}
}

func tblnameFromFilename(f string) string {
	base := filepath.Base(f)
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
	return escape.String(base)
}

func setStdin(m *virt.Machine) error {
	return m.SetInput(std.In, "[-]")
}

func setStdout(m *virt.Machine) error {
	return m.SetOutput(std.Out)
}
