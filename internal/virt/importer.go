package virt

import (
	"context"
	"errors"
	"io"

	"github.com/jimmyfrasche/etlite/internal/internal/errint"
	"github.com/jimmyfrasche/etlite/internal/token"
	"github.com/jimmyfrasche/etlite/internal/virt/internal/builder"
)

type ImportSpec struct {
	Pos            token.Position
	Internal, Temp bool
	Table, Frame   string
	Header         []string
	Limit, Offset  int
}

func (s ImportSpec) Valid() error {
	if s.Internal && s.Temp {
		return errint.New("internal table marked as temporary (by system, not user)")
	}
	if s.Table == "" {
		return errint.New("table did not provide name")
	}
	return nil
}

func (m *Machine) readHeader(frame string, header []string) ([]string, error) {
	dec := m.decoder
	if dec == nil {
		return nil, errint.New("no decoder available when importing")
	}
	inHeader, err := dec.ReadHeader(frame, header)
	if err != nil {
		return nil, err
	}
	return inHeader, nil
}

func Import(s ImportSpec) Instruction {
	return func(ctx context.Context, m *Machine) error {
		if err := s.Valid(); err != nil {
			return err
		}

		inHeader, err := m.readHeader(s.Frame, s.Header)
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

		ddl := createTable(s.Temp || s.Internal, s.Table, s.Header)
		if err := m.exec(ddl); err != nil {
			return err
		}

		ins := createInserter(s.Table, s.Header)
		if err := m.bulkInsert(ctx, s.Table, ins, s.Limit, s.Offset); err != nil {
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

func createTable(temp bool, name string, header []string) string {
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

func createInserter(name string, header []string) string {
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

func InsertWith(table, frame, inserter string, header []string, limit, offset int) Instruction {
	return func(ctx context.Context, m *Machine) error {
		if _, err := m.readHeader(frame, header); err != nil {
			return err
		}
		return m.bulkInsert(ctx, table, inserter, limit, offset)
	}
}

func (m *Machine) bulkInsert(ctx context.Context, name, ins string, limit, offset int) error {
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

		if rows%bulkCheck == 0 {
			select {
			default:
			case <-ctx.Done():
				if err := bulk.Close(); err != nil {
					return err
				}
				return ctx.Err()
			}
		}
	}

	if err := bulk.Close(); err != nil {
		return err
	}

	return dec.Reset()
}
