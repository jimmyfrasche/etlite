package compile

import (
	"github.com/jimmyfrasche/etlite/internal/ast"
	"github.com/jimmyfrasche/etlite/internal/internal/errint"
	"github.com/jimmyfrasche/etlite/internal/internal/errusr"
	"github.com/jimmyfrasche/etlite/internal/internal/synth"
	"github.com/jimmyfrasche/etlite/internal/virt"
)

func colsOf(s *ast.SQL) []string {
	hdr := make([]string, len(s.Cols))
	for i, v := range s.Cols {
		hdr[i] = v.Value
	}
	return hdr
}

func (c *compiler) compileCreateTableAsImport(nm string, s *ast.SQL) {
	imp := s.Subqueries[0]
	s.Subqueries = nil //no rewrite placeholders
	if !imp.Name.Empty() {
		panic(errusr.New(imp.Pos(), "illegal to specify table name in CREATE TABLE FROM IMPORT"))
	}
	if len(imp.Header) != 0 {
		panic(errusr.New(imp.Pos(), "illegal to specify header in CREATE TABLE FROM IMPORT"))
	}

	c.push(virt.Savepoint())

	ddl := c.rewrite(s, nil, false)
	c.push(virt.ErrPos(s.Pos()))
	c.push(virt.Exec(ddl))

	hdr := colsOf(s)
	imp.Header = hdr
	c.compileImportCommon(imp)

	ins := synth.Insert(nm, hdr)
	c.push(virt.InsertWith(nm, imp.Frame, ins, hdr, imp.Limit, imp.Offset))
	c.push(virt.Release())
}

func (c *compiler) compileInsertUsing(nm string, s *ast.SQL) {
	if len(s.Cols) == 0 {
		panic(errusr.New(s.Pos(), "INSERT USING IMPORT requires columns on INSERT"))
	}

	imp := s.Subqueries[0]
	s.Subqueries = nil //no rewrite placeholders
	if !imp.Name.Empty() {
		panic(errusr.New(imp.Pos(), "illegal to specify table name in INSERT USING IMPORT"))
	}
	if len(imp.Header) != 0 {
		panic(errusr.New(imp.Pos(), "illegal to specify header in INSERT USING IMPORT"))
	}

	c.push(virt.Savepoint())

	hdr := colsOf(s)
	imp.Header = hdr
	c.compileImportCommon(imp)

	//serialize insert statement and add VALUES (?, ..., ?);
	q := c.rewrite(s, nil, false)
	ins := synth.Values(q, hdr)

	c.push(virt.InsertWith(nm, imp.Frame, ins, hdr, imp.Limit, imp.Offset))
	c.push(virt.Release())
}

func (c *compiler) compileSubImport(i *ast.Import, tbl string) {
	if !i.Name.Empty() {
		panic(errusr.New(i.Pos(), "illegal to specify table name for import in subquery"))
	}
	if tbl == "" {
		panic(errint.New("compileSubImport requires table name"))
	}
	if i.Temporary {
		panic(errusr.New(i.Pos(), "illegal to specify temporary for import in subquery"))
	}
	i.Temporary = true

	c.compileImportCommon(i)
	if len(i.Header) == 0 {
		c.push(virt.Import(true, tbl, i.Frame, i.Limit, i.Offset))
	} else {
		c.compileImportStatic(i)
	}
}

func (c *compiler) compileImport(i *ast.Import) {
	c.push(virt.Savepoint())
	c.compileImportCommon(i)
	if len(i.Header) == 0 {
		c.push(virt.Import(i.Temporary, i.Name.String(), i.Frame, i.Limit, i.Offset))
	} else {
		c.compileImportStatic(i)
	}
	c.push(virt.Release())
}

func (c *compiler) compileImportStatic(i *ast.Import) {
	tbl := i.Name.String()
	ddl := synth.CreateTable(i.Temporary, tbl, i.Header)
	c.push(virt.Exec(ddl))
	ins := synth.Insert(tbl, i.Header)
	c.push(virt.InsertWith(tbl, i.Frame, ins, i.Header, i.Limit, i.Offset))
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

	if i.Name.Empty() {
		if c.dname != "" && !c.nameUsed(c.dname) {
			c.rec(c.dname)
			i.Name = ast.NameFromString(c.dname)
		} else if c.frname != "" && !c.nameUsed(c.frname) {
			c.rec(c.frname)
			i.Name = ast.NameFromString(c.frname)
		} else {
			panic(errusr.New(i.Pos(), "cannot derive table name"))
		}
		if i.Temporary && i.Name.DigitalObject() {
			panic(errusr.New(i.Pos(), "derived name for temp table is numeric, which is reserved"))
		}
	} else if !i.Name.HasSchema() {
		c.rec(i.Name.Object())
	}

	c.push(virt.ErrPos(i.Pos()))
}
