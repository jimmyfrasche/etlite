package compile

import (
	"github.com/jimmyfrasche/etlite/internal/ast"
	"github.com/jimmyfrasche/etlite/internal/internal/errusr"
	"github.com/jimmyfrasche/etlite/internal/virt"
)

func (c *compiler) compileDisplay(d *ast.Display) {
	if d.Format == nil && d.Device == nil {
		panic(errusr.New(d.Pos(), "at least one of format or device must be specified on DISPLAY statement"))
	}

	c.compileFormat(d.Format, outputFormat)
	c.compileDevice(d.Device, outputDevice)
	c.push(virt.MkSetEncodingFrame(d.Frame))
}
