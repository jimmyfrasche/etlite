package compile

import (
	"github.com/jimmyfrasche/etlite/internal/ast"
	"github.com/jimmyfrasche/etlite/internal/internal/errusr"
	"github.com/jimmyfrasche/etlite/internal/virt"
)

func (c *compiler) compileDisplay(d *ast.Display) {
	if d.Format == nil && d.Device == nil && d.Frame == "" {
		panic(errusr.New(d, "at least one of format, device, or frame must be specified on DISPLAY statement"))
	}

	c.compileFormat(d.Format, outputFormat)
	c.compileDevice(d.Device, outputDevice)
	c.push(virt.ErrPos(d))
	c.push(virt.SetEncodingFrame(d.Frame))
}
