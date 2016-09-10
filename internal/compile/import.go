package compile

import (
	"github.com/jimmyfrasche/etlite/internal/ast"
	"github.com/jimmyfrasche/etlite/internal/internal/errint"
	"github.com/jimmyfrasche/etlite/internal/internal/errusr"
	"github.com/jimmyfrasche/etlite/internal/virt"
)

const (
	normal = iota
	noTable
	tempTable
)

func (c *compiler) compileCreateTableAsImport(name string, i *ast.Import) { //XXX move into SQL? Reverse?
	if i.Table != "" {
		panic(errusr.New(i.Pos(), "illegal to specify table name in create table from import"))
	}
	i.Table = name
	//header will be next on stack
	c.compileImportCommon(i, noTable)
}

func (c *compiler) compileSubImport(i *ast.Import, tbl string) {
	if i.Table != "" {
		panic(errusr.New(i.Pos(), "illegal to specify table name for import in subquery"))
	}
	if tbl == "" {
		panic(errint.New("compileSubImport requires table name"))
	}
	i.Table = tbl
	i.Temporary = true
	c.compileImportCommon(i, tempTable)
}

func (c *compiler) compileImport(i *ast.Import) {
	c.compileImportCommon(i, normal)
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

func (c *compiler) compileImportCommon(i *ast.Import, kind int) {
	switch kind {
	case normal, noTable, tempTable:
	default:
		panic(errint.Newf("unexpected import kind %d", kind))
	}

	c.compileImportDeviceAndFormat(i)

	c.push(func(m *virt.Machine) error {
		m.SetDecodingFrame(i.Frame)

		//if noTable the stated name always win; if tempTable the computed name always wins.
		if kind == normal && i.Table == "" {
			i.Table = m.DerivedTableName()
		}

		//decode header from input
		dheader, err := m.DecodeHeader(i.Table, i.Header)
		if err != nil {
			return err
		}
		//if noTable the header is computed at runtime with a query; otherwise the declared header wins.
		if kind != noTable && len(i.Header) == 0 {
			i.Header = dheader
		}

		switch kind {
		case noTable:
			// Table created when the instruction provided by MkCreateFrom ran.
		case tempTable, normal:
			if err := m.CreateTable(kind == tempTable || i.Temporary, i.Table, i.Header); err != nil {
				return err
			}
		}

		return m.BulkInsert(i.Table, i.Header, i.Limit, i.Offset)
	})
}
