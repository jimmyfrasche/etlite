package compile

import (
	"strings"

	"github.com/jimmyfrasche/etlite/internal/ast"
	"github.com/jimmyfrasche/etlite/internal/internal/digital"
	"github.com/jimmyfrasche/etlite/internal/internal/errint"
	"github.com/jimmyfrasche/etlite/internal/internal/errusr"
	"github.com/jimmyfrasche/etlite/internal/internal/escape"
	"github.com/jimmyfrasche/etlite/internal/virt"
)

func (c *compiler) compileCreateTableAsImport(name, ddl string, i *ast.Import) {
	if i.Table != "" {
		panic(errusr.New(i.Pos(), "illegal to specify table name in create table from import"))
	}
	if len(i.Header) != 0 {
		panic(errusr.New(i.Pos(), "illegal to specify header in create table from import"))
	}

	c.compileImportCommon(i)
	c.push(virt.Import(virt.ImportSpec{
		Pos:    i.Pos(),
		Table:  name,
		Frame:  i.Frame,
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

	c.compileImportCommon(i)
	c.push(virt.Import(virt.ImportSpec{
		Pos:      i.Pos(),
		Internal: true,
		Table:    tbl,
		Frame:    i.Frame,
		Limit:    i.Limit,
		Offset:   i.Offset,
	}))
}

func (c *compiler) compileInsertFrom(nm string, s *ast.SQL) {
	c.push(virt.Savepoint())

	imp := s.Subqueries[0]
	s.Subqueries = nil //no rewrite placeholders

	hdr := make([]string, len(s.Cols))
	for i, v := range s.Cols {
		hdr[i] = v.Value
	}
	imp.Header = hdr //XXX should we not propagate this header?
	c.compileImportCommon(imp)

	//serialize insert statement and add VALUES (?, ..., ?);
	b := make([]string, 2, len(s.Cols)+2)
	b[0] = c.rewrite(s, nil, false)
	b[1] = "VALUES ("
	for range s.Cols[:len(s.Cols)-1] {
		b = append(b, "?,")
	}
	b = append(b, "?);")
	ins := strings.Join(b, " ")

	c.push(virt.InsertWith(nm, imp.Frame, ins, hdr, imp.Limit, imp.Offset))
	c.push(virt.Release())
}

func (c *compiler) compileImport(i *ast.Import) {
	c.compileImportCommon(i)
	c.push(virt.Import(virt.ImportSpec{
		Pos:    i.Pos(),
		Temp:   i.Temporary,
		Table:  i.Table,
		Frame:  i.Frame,
		Header: i.Header,
		Limit:  i.Limit,
		Offset: i.Offset,
	}))
}

func (c *compiler) compileImportCommon(i *ast.Import) {
	//if we used stdin to read the script we have to make sure
	//that a device has been specified and that stdin is never specified.
	if c.usedStdin {
		if i.Device != nil {
			if _, ok := i.Device.(*ast.DeviceStdio); ok {
				panic(errusr.New(i.Pos(), "script needs to read from stdin but script itself was read from stdin"))
			}
		} else if !c.hadDevice {
			panic(errusr.New(i.Pos(), "no input device specified: default stdin used for script input"))
		}
	}

	if i.Device != nil {
		c.compileDevice(i.Device, inputDevice)
		c.frname = "" //no longer correct since we've changed devices
	}
	if i.Format != nil {
		c.compileFormat(i.Format, inputFormat)
		c.frname = "" //no longer correct since we've changed formats
	}

	//header propagation
	if i.Device != nil || i.Format != nil || i.Frame != "" {
		c.hdr = nil
	}
	if len(i.Header) == 0 {
		i.Header = c.hdr
	} else {
		c.hdr = i.Header
	}

	if i.Frame == "" {
		//if there's a previous frame and we haven't switched frames, propagate
		i.Frame = c.frname
	} else {
		//record new frame so we can propagate or derive table names
		c.frname = i.Frame
	}

	if i.Table == "" {
		if c.dname != "" && !c.nameUsed(c.dname) {
			c.rec(c.dname)
			i.Table = c.dname
		} else if c.frname != "" && !c.nameUsed(c.frname) {
			c.rec(c.frname)
			i.Table = c.frname
		} else {
			panic(errusr.New(i.Pos(), "cannot derive table name"))
		}
		if i.Temporary && digital.String(i.Table) {
			panic(errusr.New(i.Pos(), "derived name for temp table is numeric, which is reserved"))
		}
		i.Table = escape.String(i.Table)
	} else {
		c.rec(i.Table)
	}
}
