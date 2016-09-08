package virt

//MkQuery creates an Instruction out of SQL query q.
//If q returns no columns, it is merely exec'd without export.
//Otherwise, an export is done using the export spec and the current output.
func MkQuery(q string) Instruction {
	return func(m *Machine) error {
		stmt, err := m.conn.Prepare(q)
		if err != nil {
			return err
		}
		defer stmt.Close()

		cols := stmt.Columns()
		//no output, just exec
		if len(cols) == 0 {
			return stmt.Exec()
		}

		e, w := m.encoder, m.output

		if err := e.WriteHeader(m.eframe, cols); err != nil {
			return err
		}

		iter, err := stmt.Iter()
		if err != nil {
			return err
		}
		for iter.Next() {
			if err := e.WriteRow(iter.Row()); err != nil {
				return err
			}
		}
		if err := iter.Err(); err != nil {
			return err
		}

		if err := e.Reset(); err != nil {
			return err
		}

		return w.Flush()
	}
}
