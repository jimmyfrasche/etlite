package virt

import (
	"io"

	"github.com/jimmyfrasche/etlite/internal/internal/errint"
	"github.com/jimmyfrasche/etlite/internal/virt/internal/builder"
)

//TODO create an enum of current states and track transitions within methods and instructions
//so we can report an internal error if things are used out of sequence

//DecodeHeader reads the header from the current input device with the current decoder.
func (m *Machine) DecodeHeader(table string, header []string) ([]string, error) {
	// make sure we have a decoder and an input
	if m.input == nil {
		return nil, errint.New("no input when attempting to init decoder")
	}
	dec := m.decoder
	if dec == nil {
		return nil, errint.New("no decoder when attempting to init decoder")
	}
	if m.dframe != "" {
		table = m.dframe
	}

	return dec.ReadHeader(table, header)
}

//CreateTable initializes the decoder then creates table name with header.
//
//InitDecoder must be called before this.
//
//See MkCreateTableFrom and BulkInsert.
func (m *Machine) CreateTable(temp bool, name string, header []string) error {
	//create ddl
	b := builder.New("CREATE")

	if temp {
		b.Push("TEMPORARY")
	}

	b.Push("TABLE", name, "(")

	b.CSV(header, func(h string) {
		b.Push(h, "TEXT")
	})

	b.Push(")")

	return m.exec(b.Join(" "))
}

//BulkInsert from the current decoder into table name with header.
//
//Table name must exist and should have been created by
//either CreateTableFrom or CreateTable, with no intervening
//reads to or changes of the decoder.
//
//Before that DecodeHeader must be called.
func (m *Machine) BulkInsert(name string, header []string, limit, offset int) error {
	//make sure we have a decoder
	dec := m.decoder
	if dec == nil {
		return errint.Newf("no decoder when attempting to import %s", name)
	}

	//build the bulk loader
	b := builder.New("INSERT INTO", name, "(")

	b.CSV(header, func(h string) {
		b.Push(h)
	})

	b.Push(") VALUES (")

	b.CSV(header, func(string) {
		b.Push("?")
	})
	b.Push(")")

	p, err := m.conn.Prepare(b.Join(" "))
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
