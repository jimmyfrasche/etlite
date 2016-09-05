package engine

import (
	"os"
	"strings"

	"github.com/jimmyfrasche/etlite/internal/internal/errint"
)

//multiexec execs each query in qs in turn,
//failing on first error.
func (m *Machine) multiexec(qs ...string) error {
	for _, q := range qs {
		if err := m.exec(q); err != nil {
			return err
		}
	}
	return nil
}

//createSystab creates and populates the tables in the attached sys db.
//It contains
//	CREATE TABLE sys.env (
//		name  TEXT PRIMARY KEY ON CONFLICT REPLACE,
//		value TEXT NOT NULL
//	) WITHOUT ROWID
//which is populated with the processes environment variables and
//	CREATE TABLE sys.arg (
//		value TEXT NOT NULL
//	)
//which is populated by the processes leftover arguments.
//These may be used and modified directly by the user.
//Additional syntax is sugar for direct access.
//
//If env == nil, os.Environ() is used.
func (m *Machine) createSystab(env map[string]string, args []string) error {
	if env == nil {
		env = map[string]string{}
		for _, e := range os.Environ() {
			p := strings.SplitN(e, "=", 2)
			env[p[0]] = p[1]
		}
	}

	err := m.multiexec(
		`ATTACH ':memory:' AS sys`,
		`CREATE TABLE sys.env (
	name TEXT PRIMARY KEY ON CONFLICT REPLACE,
	value TEXT NOT NULL
) WITHOUT ROWID`,
		`CREATE TABLE sys.arg (value TEXT NOT NULL)`,
	)
	if err != nil {
		return errint.Wrap(err)
	}

	eins, err := m.conn.Prepare(`INSERT INTO sys.env VALUES (?, ?)`)
	if err != nil {
		return errint.Wrap(err)
	}
	defer eins.Close()
	ains, err := m.conn.Prepare(`INSERT INTO sys.arg VALUES (?)`)
	if err != nil {
		return errint.Wrap(err)
	}
	defer ains.Close()

	m.readEnv, err = m.conn.Prepare(`SELECT name || '=' || value FROM sys.env WHERE value IS NOT NULL`)
	if err != nil {
		return errint.Wrap(err)
	}

	m.tmpFromEnv, err = m.conn.Prepare(`SELECT value FROM sys.env WHERE name = 'TMPDIR'`)
	if err != nil {
		return errint.Wrap(err)
	}

	eload, err := eins.Loader()
	if err != nil {
		return err
	}
	for key, value := range env {
		if err := eload.Load([]*string{&key, &value}); err != nil {
			return err
		}
	}
	if err := eload.Close(); err != nil {
		return err
	}

	aload, err := ains.Loader()
	if err != nil {
		return err
	}
	for _, arg := range args {
		if err := aload.Load([]*string{&arg}); err != nil {
			return err
		}
	}
	return aload.Close()
}

//Env dumps the internal sys.env table (which the user is free to modify)
//in Go readable format.
func (m *Machine) Env() (env []string, err error) {
	i, err := m.readEnv.Iter()
	if err != nil {
		return nil, err
	}
	for i.Next() {
		r := i.Row()[0]
		env = append(env, *r)
	}
	return env, i.Err()
}
