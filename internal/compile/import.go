package compile

import (
	"github.com/jimmyfrasche/etlite/internal/ast"
	"github.com/jimmyfrasche/etlite/internal/internal/errint"
	"github.com/jimmyfrasche/etlite/internal/internal/errusr"
	"github.com/jimmyfrasche/etlite/internal/virt"
)

func (c *compiler) compileCreateTableAsImport(name, ddl string, i *ast.Import) {
	if i.Table != "" {
		panic(errusr.New(i.Pos(), "illegal to specify table name in create table from import"))
	}
	if len(i.Header) != 0 {
		panic(errusr.New(i.Pos(), "illegal to specify header in create table from import"))
	}

	c.compileImportDeviceAndFormat(i)
	c.push(virt.MkImport(virt.ImportSpec{
		Pos:    i.Pos(),
		Table:  name,
		Limit:  i.Limit,
		Offset: i.Offset,
		DDL:    ddl,
	}))
}

func (c *compiler) compileSubImport(i *ast.Import, tbl string) {
	if i.Table != "" {
		panic(errusr.New(i.Pos(), "illegal to specify table name for import in subquery"))
	}
	if tbl == "" {
		panic(errint.New("compileSubImport requires table name"))
	}

	c.compileImportDeviceAndFormat(i)
	c.push(virt.MkImport(virt.ImportSpec{
		Pos:      i.Pos(),
		Internal: true,
		Table:    tbl,
		Limit:    i.Limit,
		Offset:   i.Offset,
	}))
}

func (c *compiler) compileImport(i *ast.Import) {
	c.compileImportDeviceAndFormat(i)
	c.push(virt.MkImport(virt.ImportSpec{
		Pos:    i.Pos(),
		Temp:   i.Temporary,
		Table:  i.Table,
		Header: i.Header,
		Limit:  i.Limit,
		Offset: i.Offset,
	}))
}

func (c *compiler) compileImportDeviceAndFormat(i *ast.Import) {
	if c.usedStdin && i.Device != nil {
		if _, ok := i.Device.(*ast.DeviceStdio); ok {
			panic(errusr.New(i.Pos(), "script needs to read from stdin but script itself was read from stdin"))
		}
	}

	if i.Device != nil {
		c.compileDevice(i.Device, inputDevice)
	}
	if i.Format != nil {
		c.compileFormat(i.Format, inputFormat)
	}
}
