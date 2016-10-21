package virt

import (
	"context"
	"errors"
	"io"

	"github.com/jimmyfrasche/etlite/internal/internal/errint"
	"github.com/jimmyfrasche/etlite/internal/internal/errusr"
	"github.com/jimmyfrasche/etlite/internal/token"
	"github.com/jimmyfrasche/etlite/internal/virt/internal/builder"
)

type ImportSpec struct {
	Pos            token.Position
	Internal, Temp bool
	Table, Frame   string
	Header         []string
	DDL            string
	Limit, Offset  int
}

func (s ImportSpec) Valid() error {
	if s.DDL != "" {
		if s.Internal {
			return errint.New("CREATE TABLE FROM marked as internal table")
		}
		if s.Temp {
			return errint.New("CREATE TABLE FROM marked as temporary table (by system, not user)")
		}
		if len(s.Header) > 0 {
			return errint.New("CREATE TABLE FROM provides user defined header")
		}
		if s.Table == "" {
			return errint.New("CREATE TABLE FROM provides no table name")
		}
	} else if s.Internal && s.Temp {
		return errint.New("internal table marked as temporary (by system, not user)")
	}
	if s.Table == "" {
		return errint.New("table did not provide name")
	}
	return nil
}

func Import(s ImportSpec) Instruction {
	return func(ctx context.Context, m *Machine) error {
		if err := s.Valid(); err != nil {
			return err
		}

		//CREATE TABLE FROM
		if s.DDL != "" {
			//build table and compute header
			if err := m.exec(s.DDL); err != nil {
				return errusr.Wrap(s.Pos, err)
			}
			p, err := m.conn.Prepare("SELECT * FROM " + s.Table)
			if err != nil {
				return err
			}
			s.Header = p.Columns()
			if len(s.Header) == 0 {
				return errint.New("could not retrieve columns in CREATE TABLE FROM")
			}
			if err := p.Close(); err != nil {
				return err
			}
		}

		//prep the decoder
		dec := m.decoder
		if dec == nil {
			return errint.New("no decoder available when importing")
		}
		inHeader, err := dec.ReadHeader(s.Frame, s.Header)
		if err != nil {
			return err
		}

		//no header specified, use the one from the decoder
		if len(s.Header) == 0 {
			if len(inHeader) == 0 {
				return errors.New("no header specified and none returned by " + m.decoder.Name() + " format")
			}
			s.Header = inHeader
		}

		//internal tables can be created en masse so have to take care of their own savepointing.
		if !s.Internal {
			if err := m.savepoint(); err != nil {
				return err
			}
		}

		//not a CREATE TABLE FROM so we need to make the table
		if s.DDL == "" {
			ddl := m.createTable(s.Temp || s.Internal, s.Table, s.Header)
			if err := m.exec(ddl); err != nil {
				return err
			}
		}

		ins := m.createInserter(s.Table, s.Header)
		if err := m.bulkInsert(s.Table, ins, s.Limit, s.Offset); err != nil {
			return err
		}

		if !s.Internal {
			if err := m.release(); err != nil {
				return err
			}
		}

		return nil
	}
}

func (m *Machine) createTable(temp bool, name string, header []string) string {
	b := builder.New("CREATE")

	if temp {
		b.Push("TEMPORARY")
	}

	b.Push("TABLE", name, "(")

	b.CSV(header, func(h string) {
		b.Push(h, "TEXT")
	})

	b.Push(")")

	return b.Join(" ")
}

func (m *Machine) createInserter(name string, header []string) string {
	b := builder.New("INSERT INTO", name, "(")

	b.CSV(header, func(h string) {
		b.Push(h)
	})

	b.Push(") VALUES (")

	b.CSV(header, func(string) {
		b.Push("?")
	})
	b.Push(")")

	return b.Join(" ")
}

func (m *Machine) bulkInsert(name, ins string, limit, offset int) error {
	//make sure we have a decoder
	dec := m.decoder
	if dec == nil {
		return errint.Newf("no decoder when attempting to import %s", name)
	}

	//build the bulk loader
	p, err := m.conn.Prepare(ins)
	if err != nil {
		return err
	}
	defer p.Close()

	bulk, err := p.Loader()
	if err != nil {
		return err
	}

	if offset > 0 {
		if err := dec.Skip(offset); err != nil {
			return err
		}
	}

	for rows := 0; limit > 0 && rows == limit || true; rows++ {
		row, err := dec.ReadRow()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if err := bulk.Load(row); err != nil {
			return err
		}
	}

	if err := bulk.Close(); err != nil {
		return err
	}

	return dec.Reset()
}
