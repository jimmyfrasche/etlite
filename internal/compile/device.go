package compile

import (
	"github.com/jimmyfrasche/etlite/internal/ast"
	"github.com/jimmyfrasche/etlite/internal/internal/errint"
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
			c.push(virt.MkUseStdin())
		} else {
			c.push(virt.MkUseStdout())
		}

	case *ast.DeviceFile: //always pushes filename
		name, ok := d.Name.Unescape()
		if !ok {
			panic(errint.Newf("file device name must be literal or string got %s", d.Name.Kind))
		}
		if read {
			c.push(virt.MkUseFileInput(name))
		} else {
			c.push(virt.MkUseFileOutput(name))
		}
	}
}
