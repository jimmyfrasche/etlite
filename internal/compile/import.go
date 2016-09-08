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
	c.pushpush(i.Header)
	i.Table = tbl
	i.Temporary = true
	c.compileImportCommon(i, tempTable)
}

func (c *compiler) compileImport(i *ast.Import) {
	c.pushpush(i.Header)
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
	c.pushpush(i.Frame)
}

func (c *compiler) compileImportCommon(i *ast.Import, kind int) {
	switch kind {
	case normal, noTable, tempTable:
	default:
		panic(errint.Newf("unexpected kind %d", kind))
	}
	c.intOrSub(i.Limit, -1)
	c.intOrSub(i.Offset, -1)

	//these are executed and burn up their share of the stack before the next instruction runs
	c.compileImportDeviceAndFormat(i)

	c.push(func(m *virt.Machine) error {
		frame, err := m.PopString()
		if err != nil {
			return err
		}
		m.SetDecodingFrame(frame)

		offset, err := m.PopInt()
		if err != nil {
			return err
		}

		limit, err := m.PopInt()
		if err != nil {
			return err
		}

		//this has to be on the stack for the CREATE TABLE FROM IMPORT form,
		//for simplicity this supersedes i.Header.
		header, err := m.PopStrings()
		if err != nil {
			return err
		}

		//if noTable the stated name always win; if tempTable the computed name always wins.
		if kind == normal && i.Table == "" {
			i.Table = m.DerivedTableName()
		}

		//decode header from input
		dheader, err := m.DecodeHeader(i.Table, header)
		if err != nil {
			return err
		}
		//if noTable the header is computed at runtime with a query; otherwise the declared header wins.
		if kind != noTable && len(header) == 0 {
			header = dheader
		}

		switch kind {
		case noTable:
			// Table created when the instruction provided by MkCreateFrom ran.
		case tempTable, normal:
			if err := m.CreateTable(kind == tempTable || i.Temporary, i.Table, header); err != nil {
				return err
			}
		}

		return m.BulkInsert(i.Table, header, limit, offset)
	})
}
